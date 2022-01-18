package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/logging"
	mocksD "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestReadOnlyRun(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		expectError    bool
		inspectTaskErr error
		renderTmplErr  error
		config         *config.Config
	}{
		{
			"error on driver.RenderTemplate()",
			true,
			nil,
			errors.New("error on driver.RenderTemplate()"),
			singleTaskConfig(),
		},
		{
			"error on driver.InspectTask()",
			true,
			errors.New("error on driver.InspectTask()"),
			nil,
			singleTaskConfig(),
		},
		{
			"happy path",
			false,
			nil,
			nil,
			singleTaskConfig(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := new(mocks.Watcher)
			w.On("Size").Return(5)

			ctrl := ReadOnly{baseController: &baseController{
				watcher: w,
				drivers: driver.NewDrivers(),
				logger:  logging.NewNullLogger(),
			}}

			d := new(mocksD.Driver)
			d.On("Task").Return(enabledTestTask(t, "task"))
			d.On("TemplateIDs").Return(nil)
			d.On("RenderTemplate", mock.Anything).
				Return(true, tc.renderTmplErr)
			d.On("InspectTask", mock.Anything).
				Return(driver.InspectPlan{}, tc.inspectTaskErr)
			err := ctrl.drivers.Add("task", d)
			require.NoError(t, err)

			err = ctrl.Run(context.Background())
			if tc.expectError {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tc.name)
				}
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestReadOnlyRun_context_cancel(t *testing.T) {
	r := new(mocks.Resolver)
	r.On("Run", mock.Anything, mock.Anything).
		Return(hcat.ResolveEvent{Complete: false}, nil)

	w := new(mocks.Watcher)
	w.On("WaitCh", mock.Anything, mock.Anything).Return(nil).
		On("Size").Return(5).
		On("Stop").Return()

	d := new(mocksD.Driver)
	d.On("Task").Return(enabledTestTask(t, "task"))
	d.On("TemplateIDs").Return(nil)
	d.On("RenderTemplate", mock.Anything).Return(false, nil)
	drivers := driver.NewDrivers()
	err := drivers.Add("task", d)
	require.NoError(t, err)

	ctrl := ReadOnly{baseController: &baseController{
		watcher:  w,
		resolver: r,
		drivers:  drivers,
		logger:   logging.NewNullLogger(),
	}}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		err := ctrl.Run(ctx)
		if err != nil {
			errCh <- err
		}
	}()
	cancel()

	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Error("wanted 'context canceled', got:", err)
		}
	case <-time.After(time.Second * 5):
		t.Fatal("Run did not exit properly from cancelling context")
	}
}
