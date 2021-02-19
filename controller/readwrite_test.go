package controller

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	"github.com/hashicorp/consul-terraform-sync/handler"
	mocksD "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestReadWrite_CheckApply(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		expectError   bool
		applyTaskErr  error
		renderTmplErr error
		taskName      string
		addToStore    bool
	}{
		{
			"error on driver.RenderTemplate()",
			true,
			nil,
			errors.New("error on driver.RenderTemplate()"),
			"task_render_tmpl",
			true,
		},
		{
			"error on driver.ApplyTask()",
			true,
			errors.New("error on driver.ApplyTask()"),
			nil,
			"task_apply",
			true,
		},
		{
			"error creating new event",
			true,
			nil,
			nil,
			"",
			false,
		},
		{
			"happy path",
			false,
			nil,
			nil,
			"task_apply",
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := new(mocksD.Driver)
			d.On("RenderTemplate", mock.Anything).
				Return(true, tc.renderTmplErr)
			d.On("ApplyTask", mock.Anything).Return(tc.applyTaskErr)

			controller := ReadWrite{
				baseController: &baseController{},
				store:          event.NewStore(),
			}
			u := unit{taskName: tc.taskName, driver: d}
			ctx := context.Background()

			_, err := controller.checkApply(ctx, u, false)
			data := controller.store.Read(tc.taskName)
			events := data[tc.taskName]

			if !tc.addToStore {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.name)
				assert.Equal(t, 0, len(events))
				return
			}

			assert.Equal(t, 1, len(events))
			event := events[0]
			assert.Equal(t, tc.taskName, event.TaskName)
			assert.False(t, event.StartTime.IsZero())
			assert.False(t, event.EndTime.IsZero())

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.name)
				assert.False(t, event.Success)
				require.NotNil(t, event.EventError)
				assert.Contains(t, event.EventError.Message, tc.name)
			} else {
				assert.NoError(t, err)
				assert.True(t, event.Success)
			}
		})
	}
}

func TestReadWrite_CheckApply_Store(t *testing.T) {
	t.Run("mult-checkapply-store", func(t *testing.T) {
		d := new(mocksD.Driver)
		d.On("RenderTemplate", mock.Anything).Return(true, nil)
		d.On("ApplyTask", mock.Anything).Return(nil)

		controller := ReadWrite{
			baseController: &baseController{},
			store:          event.NewStore(),
		}

		unitA := unit{taskName: "task_a", driver: d}
		unitB := unit{taskName: "task_b", driver: d}
		ctx := context.Background()

		controller.checkApply(ctx, unitA, false)
		controller.checkApply(ctx, unitB, false)
		controller.checkApply(ctx, unitA, false)
		controller.checkApply(ctx, unitA, false)
		controller.checkApply(ctx, unitA, false)
		controller.checkApply(ctx, unitB, false)

		taskStatuses := controller.store.Read("")

		assert.Equal(t, 4, len(taskStatuses["task_a"]))
		assert.Equal(t, 2, len(taskStatuses["task_b"]))
	})
}

func TestOnce(t *testing.T) {
	t.Run("init-wraps-units", func(t *testing.T) {
		conf := singleTaskConfig()
		w := new(mocks.Watcher)
		errCh := make(chan error)
		var errChRc <-chan error = errCh
		go func() { errCh <- nil }()
		w.On("WaitCh", mock.Anything).Return(errChRc).Once()
		w.On("Size").Return(5)

		d := new(mocksD.Driver)
		d.On("RenderTemplate", mock.Anything).Return(false, nil).Once()
		d.On("RenderTemplate", mock.Anything).Return(true, nil).Once()
		d.On("InitTask", mock.Anything).Return(nil).Once()
		d.On("ApplyTask", mock.Anything).Return(nil).Once()

		rw := &ReadWrite{
			baseController: &baseController{
				watcher: w,
				newDriver: func(*config.Config, driver.Task, templates.Watcher) (driver.Driver, error) {
					return d, nil
				},
				conf: conf,
			},
			store: event.NewStore(),
		}

		ctx := context.Background()
		_, err := rw.Init(ctx)
		assert.NoError(t, err)

		// testing really starts here...
		once := Oncer(rw)
		done := make(chan error)
		// running in goroutine so I can timeout
		go func() {
			done <- once.Once(ctx)
		}()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal("Unexpected error in Once():", err)
			}
		case <-time.After(time.Second):
			t.Fatal("Once didn't return in expected time")
		}

		// Not sure about these... to far into the "test implementation" zone?
		w.AssertExpectations(t)
		d.AssertExpectations(t)
	})
}

func TestReadWriteUnits(t *testing.T) {
	t.Run("simple-success", func(t *testing.T) {
		d := new(mocksD.Driver)
		d.On("InitWork", mock.Anything).Return(nil)
		d.On("RenderTemplate", mock.Anything).Return(true, nil)
		d.On("ApplyTask", mock.Anything).Return(nil)
		d.On("ApplyTask", mock.Anything).Return(fmt.Errorf("test"))

		u := unit{taskName: "foo", driver: d}
		controller := ReadWrite{
			baseController: &baseController{
				units: []unit{u},
			},
			store: event.NewStore(),
		}

		ctx := context.Background()
		errCh := controller.runUnits(ctx)
		err := <-errCh
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("apply-error", func(t *testing.T) {
		d := new(mocksD.Driver)
		d.On("InitWork", mock.Anything).Return(nil)
		d.On("RenderTemplate", mock.Anything).Return(true, nil)
		d.On("ApplyTask", mock.Anything).Return(fmt.Errorf("test"))

		u := unit{taskName: "foo", driver: d}
		controller := ReadWrite{
			baseController: &baseController{
				units: []unit{u},
			},
			store: event.NewStore(),
		}

		ctx := context.Background()
		errCh := controller.runUnits(ctx)
		err := <-errCh
		testErr := fmt.Errorf("could not apply: %s", "test")
		if errors.Is(err, testErr) {
			t.Error(
				fmt.Sprintf("bad error, expected '%v', got '%v'", testErr, err))
		}
	})
}

func TestReadWriteRun_context_cancel(t *testing.T) {
	w := new(mocks.Watcher)
	w.On("WaitCh", mock.Anything, mock.Anything).Return(nil).
		On("Size").Return(5).
		On("Stop").Return()

	ctl := ReadWrite{
		baseController: &baseController{
			units:   []unit{},
			watcher: w,
		},
		store: event.NewStore(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		err := ctl.Run(ctx)
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

// singleTaskConfig returns a happy path config that has a single task
func singleTaskConfig() *config.Config {
	c := &config.Config{
		Consul: &config.ConsulConfig{
			Address: config.String("consul-example.com"),
		},
		Driver: &config.DriverConfig{
			Terraform: &config.TerraformConfig{
				Log:        config.Bool(true),
				Path:       config.String("path"),
				WorkingDir: config.String("working"),
			},
		},
		Tasks: &config.TaskConfigs{
			{
				Description: config.String("automate services for X to do Y"),
				Name:        config.String("task"),
				Services:    []string{"serviceA", "serviceB", "serviceC"},
				Providers:   []string{"X", handler.TerraformProviderFake},
				Source:      config.String("Y"),
				Version:     config.String("v1"),
			},
		},
		Services: &config.ServiceConfigs{
			{
				ID:          config.String("serviceA_id"),
				Name:        config.String("serviceA"),
				Description: config.String("descriptionA"),
			}, {
				ID:          config.String("serviceB_id"),
				Name:        config.String("serviceB"),
				Namespace:   config.String("teamB"),
				Description: config.String("descriptionB"),
			},
		},
		TerraformProviders: &config.TerraformProviderConfigs{{
			"X": map[string]interface{}{},
			handler.TerraformProviderFake: map[string]interface{}{
				"name": "fake-provider",
			},
		}},
	}

	c.Finalize()
	return c
}
