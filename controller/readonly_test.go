package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	mocksD "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestReadOnlyRun(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name              string
		expectError       bool
		inspectTaskErr    error
		resolverRunErr    error
		templateRenderErr error
		watcherWaitErr    error
		config            *config.Config
	}{
		{
			"error on resolver.Run()",
			true,
			nil,
			errors.New("error on resolver.Run()"),
			nil,
			nil,
			singleTaskConfig(),
		},
		{
			"error on driver.InspectTask()",
			true,
			errors.New("error on driver.InspectTask()"),
			nil,
			nil,
			nil,
			singleTaskConfig(),
		},
		{
			"happy path",
			false,
			nil,
			nil,
			nil,
			nil,
			singleTaskConfig(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			tmpl := new(mocks.Template)
			tmpl.On("Render", mock.Anything).Return(hcat.RenderResult{}, tc.templateRenderErr).Once()

			r := new(mocks.Resolver)
			r.On("Run", mock.Anything, mock.Anything).
				Return(hcat.ResolveEvent{Complete: true}, tc.resolverRunErr)

			w := new(mocks.Watcher)
			w.On("Wait", mock.Anything).Return(tc.watcherWaitErr)

			d := new(mocksD.Driver)
			d.On("InspectTask", mock.Anything).Return(tc.inspectTaskErr)

			ctrl := ReadOnly{baseController: &baseController{
				watcher:  w,
				resolver: r,
				units:    []unit{{template: tmpl, driver: d}},
			}}
			ctx := context.Background()

			err := ctrl.Run(ctx)
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
		On("Stop").Return()

	ctrl := ReadOnly{baseController: &baseController{
		watcher:  w,
		resolver: r,
		units:    []unit{{template: new(mocks.Template)}},
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
