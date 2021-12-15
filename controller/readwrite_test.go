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
	"github.com/hashicorp/consul-terraform-sync/logging"
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
		enabledTask   bool
		applyTaskErr  error
		renderTmplErr error
		taskName      string
		addToStore    bool
	}{
		{
			"error on driver.RenderTemplate()",
			true,
			true,
			nil,
			errors.New("error on driver.RenderTemplate()"),
			"task_render_tmpl",
			true,
		},
		{
			"error on driver.ApplyTask()",
			true,
			true,
			errors.New("error on driver.ApplyTask()"),
			nil,
			"task_apply",
			true,
		},
		{
			"error creating new event",
			true,
			true,
			nil,
			nil,
			"",
			false,
		},
		{
			"disabled task",
			false,
			false,
			nil,
			nil,
			"disabled_task",
			false,
		},
		{
			"happy path",
			false,
			true,
			nil,
			nil,
			"task_apply",
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := new(mocksD.Driver)
			drivers := driver.NewDrivers()
			var task *driver.Task
			if tc.enabledTask {
				task = enabledTestTask(t, tc.taskName)
				d.On("RenderTemplate", mock.Anything).
					Return(true, tc.renderTmplErr)
				d.On("ApplyTask", mock.Anything).Return(tc.applyTaskErr)
			} else {
				task = disabledTestTask(t, tc.taskName)
			}
			d.On("Task").Return(task)
			drivers.Add(tc.taskName, d)

			controller := ReadWrite{
				baseController: &baseController{
					drivers: drivers,
					logger:  logging.NewNullLogger(),
				},
				store: event.NewStore(),
			}
			ctx := context.Background()

			_, err := controller.checkApply(ctx, d, false, false)
			data := controller.store.Read(tc.taskName)
			events := data[tc.taskName]

			if !tc.addToStore {
				if tc.expectError {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tc.name)
				}
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

	t.Run("unrendered-scheduled-tasks", func(t *testing.T) {
		// Test the behavior for once-mode and daemon-mode for the situation
		// where a scheduled task's template did not render

		controller := ReadWrite{
			baseController: &baseController{
				drivers: driver.NewDrivers(),
				logger:  logging.NewNullLogger(),
			},
			store: event.NewStore(),
		}

		d := new(mocksD.Driver)
		taskName := "scheduled_task"
		d.On("Task").Return(scheduledTestTask(t, taskName))
		d.On("RenderTemplate", mock.Anything).Return(false, nil)
		controller.drivers.Add(taskName, d)

		// Once-mode - confirm no events are stored
		ctx := context.Background()
		_, err := controller.checkApply(ctx, d, false, true)
		assert.NoError(t, err)
		data := controller.store.Read(taskName)
		events := data[taskName]
		assert.Equal(t, 0, len(events))

		// Daemon-mode - confirm an event is stored
		_, err = controller.checkApply(ctx, d, false, false)
		assert.NoError(t, err)
		data = controller.store.Read(taskName)
		events = data[taskName]
		assert.Equal(t, 1, len(events))
	})
}

func TestReadWrite_CheckApply_Store(t *testing.T) {
	t.Run("mult-checkapply-store", func(t *testing.T) {
		d := new(mocksD.Driver)
		d.On("Task").Return(enabledTestTask(t, "task_a"))
		d.On("RenderTemplate", mock.Anything).Return(true, nil)
		d.On("ApplyTask", mock.Anything).Return(nil)

		disabledD := new(mocksD.Driver)
		disabledD.On("Task").Return(disabledTestTask(t, "task_b"))

		controller := ReadWrite{
			baseController: &baseController{
				drivers: driver.NewDrivers(),
				logger:  logging.NewNullLogger(),
			},
			store: event.NewStore(),
		}

		controller.drivers.Add("task_a", d)
		controller.drivers.Add("task_b", disabledD)
		ctx := context.Background()

		controller.checkApply(ctx, d, false, false)
		controller.checkApply(ctx, disabledD, false, false)
		controller.checkApply(ctx, d, false, false)
		controller.checkApply(ctx, d, false, false)
		controller.checkApply(ctx, d, false, false)
		controller.checkApply(ctx, disabledD, false, false)

		taskStatuses := controller.store.Read("")

		assert.Equal(t, 4, len(taskStatuses["task_a"]))
		assert.Equal(t, 0, len(taskStatuses["task_b"]))
	})
}

func TestOnce(t *testing.T) {
	rw := &ReadWrite{
		store: event.NewStore(),
	}

	testCases := []struct {
		name     string
		once     func(context.Context) error
		numTasks int
	}{
		{
			"consecutive one task",
			rw.Once,
			1,
		}, {
			"consecutive multiple tasks",
			rw.Once,
			10,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errCh := make(chan error)
			var errChRc <-chan error = errCh
			go func() { errCh <- nil }()
			w := new(mocks.Watcher)
			w.On("WaitCh", mock.Anything).Return(errChRc)
			w.On("Size").Return(tc.numTasks)

			rw.baseController = &baseController{
				watcher: w,
				drivers: driver.NewDrivers(),
				newDriver: func(c *config.Config, task *driver.Task, w templates.Watcher) (driver.Driver, error) {
					taskName := task.Name()
					d := new(mocksD.Driver)
					d.On("Task").Return(enabledTestTask(t, taskName)).Twice()
					d.On("RenderTemplate", mock.Anything).Return(false, nil).Once()
					d.On("RenderTemplate", mock.Anything).Return(true, nil).Once()
					d.On("InitTask", mock.Anything, mock.Anything).Return(nil).Once()
					d.On("ApplyTask", mock.Anything).Return(nil).Once()
					return d, nil
				},
				conf:   multipleTaskConfig(tc.numTasks),
				logger: logging.NewNullLogger(),
			}

			ctx := context.Background()
			err := rw.Init(ctx)
			assert.NoError(t, err)

			// testing really starts here...
			done := make(chan error)
			// running in goroutine so I can timeout
			go func() {
				done <- tc.once(ctx)
			}()
			select {
			case err := <-done:
				assert.NoError(t, err, "unexpected error in Once()")
			case <-time.After(time.Second):
				t.Fatal("Once didn't return in expected time")
			}

			for _, d := range rw.drivers.Map() {
				d.(*mocksD.Driver).AssertExpectations(t)
			}
		})
	}
}

func TestReadWrite_Once_error(t *testing.T) {
	// Test once mode error handling when a driver returns an error
	rw := &ReadWrite{
		store: event.NewStore(),
	}
	numTasks := 5
	w := new(mocks.Watcher)
	w.On("WaitCh", mock.Anything).Return(nil)
	w.On("Size").Return(numTasks)

	expectedErr := fmt.Errorf("test error")
	rw.baseController = &baseController{
		watcher: w,
		drivers: driver.NewDrivers(),
		newDriver: func(c *config.Config, task *driver.Task, w templates.Watcher) (driver.Driver, error) {
			taskName := task.Name()
			d := new(mocksD.Driver)
			d.On("Task").Return(enabledTestTask(t, taskName))
			d.On("RenderTemplate", mock.Anything).Return(true, nil)
			d.On("InitTask", mock.Anything, mock.Anything).Return(nil)
			if taskName == "task_03" {
				// Mock an error during apply for a task
				d.On("ApplyTask", mock.Anything).Return(expectedErr)
			} else {
				d.On("ApplyTask", mock.Anything).Return(nil)
			}
			return d, nil
		},
		conf:   multipleTaskConfig(numTasks),
		logger: logging.NewNullLogger(),
	}

	ctx := context.Background()
	err := rw.Init(ctx)
	assert.NoError(t, err)

	// testing really starts here...
	done := make(chan error)
	// running in goroutine so I can timeout
	go func() {
		done <- rw.Once(ctx)
	}()
	select {
	case err := <-done:
		assert.Error(t, err, "task_03 driver error should bubble up")
		assert.Contains(t, err.Error(), expectedErr.Error(), "unexpected error in Once")
	case <-time.After(time.Second):
		t.Fatal("Once didn't return in expected time")
	}
}

func TestReadWrite_runDynamicTasks(t *testing.T) {
	t.Run("simple-success", func(t *testing.T) {
		controller := ReadWrite{
			baseController: &baseController{
				drivers: driver.NewDrivers(),
				logger:  logging.NewNullLogger(),
			},
			store: event.NewStore(),
		}

		d := new(mocksD.Driver)
		d.On("Task").Return(enabledTestTask(t, "task"))
		d.On("InitWork", mock.Anything).Return(nil)
		d.On("RenderTemplate", mock.Anything).Return(true, nil)
		d.On("ApplyTask", mock.Anything).Return(nil)
		d.On("ApplyTask", mock.Anything).Return(fmt.Errorf("test"))
		controller.drivers.Add("task", d)

		ctx := context.Background()
		errCh := controller.runDynamicTasks(ctx)
		err := <-errCh
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("apply-error", func(t *testing.T) {
		controller := ReadWrite{
			baseController: &baseController{
				drivers: driver.NewDrivers(),
				logger:  logging.NewNullLogger(),
			},
			store: event.NewStore(),
		}

		d := new(mocksD.Driver)
		d.On("Task").Return(enabledTestTask(t, "task"))
		d.On("InitWork", mock.Anything).Return(nil)
		d.On("RenderTemplate", mock.Anything).Return(true, nil)
		d.On("ApplyTask", mock.Anything).Return(fmt.Errorf("test"))
		controller.drivers.Add("task", d)

		ctx := context.Background()
		errCh := controller.runDynamicTasks(ctx)
		err := <-errCh
		testErr := fmt.Errorf("could not apply: %s", "test")
		if errors.Is(err, testErr) {
			t.Error(
				fmt.Sprintf("bad error, expected '%v', got '%v'", testErr, err))
		}
	})

	t.Run("skip-scheduled-tasks", func(t *testing.T) {
		controller := ReadWrite{
			baseController: &baseController{
				drivers: driver.NewDrivers(),
			},
			store: event.NewStore(),
		}

		taskName := "scheduled_task"
		d := new(mocksD.Driver)
		d.On("Task").Return(scheduledTestTask(t, taskName))
		// no other methods should be called (or mocked)
		controller.drivers.Add(taskName, d)

		ctx := context.Background()
		errCh := controller.runDynamicTasks(ctx)
		err := <-errCh
		if err != nil {
			t.Error(err)
		}
	})
}

func TestReadWrite_runScheduledTask(t *testing.T) {
	t.Run("happy-path", func(t *testing.T) {
		ctrl := ReadWrite{
			baseController: &baseController{
				drivers: driver.NewDrivers(),
				logger:  logging.NewNullLogger(),
			},
			store: event.NewStore(),
		}

		taskName := "scheduled_task"
		d := new(mocksD.Driver)
		d.On("Task").Return(scheduledTestTask(t, taskName)).Twice()
		d.On("RenderTemplate", mock.Anything).Return(true, nil).Once()
		d.On("ApplyTask", mock.Anything).Return(nil).Once()

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error)
		go func() {
			err := ctrl.runScheduledTask(ctx, d)
			if err != nil {
				errCh <- err
			}
		}()
		time.Sleep(3 * time.Second)
		cancel()

		select {
		case err := <-errCh:
			assert.Equal(t, context.Canceled, err)
		case <-time.After(time.Second * 5):
			t.Fatal("runScheduledTask did not exit properly from cancelling context")
		}

		d.AssertExpectations(t)
	})

	t.Run("dynamic-task-errors", func(t *testing.T) {
		ctrl := ReadWrite{
			baseController: &baseController{
				drivers: driver.NewDrivers(),
				logger:  logging.NewNullLogger(),
			},
			store: event.NewStore(),
		}

		taskName := "dynamic_task"
		d := new(mocksD.Driver)
		d.On("Task").Return(enabledTestTask(t, taskName)).Once()

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error)
		go func() {
			err := ctrl.runScheduledTask(ctx, d)
			if err != nil {
				errCh <- err
			}
		}()
		time.Sleep(1 * time.Second)
		cancel()

		select {
		case err := <-errCh:
			assert.Contains(t, err.Error(), "expected a schedule condition")
		case <-time.After(time.Second * 5):
			t.Fatal("runScheduledTask did not exit properly from cancelling context")
		}

		d.AssertExpectations(t)
	})
}

func TestReadWriteRun_context_cancel(t *testing.T) {
	w := new(mocks.Watcher)
	w.On("WaitCh", mock.Anything, mock.Anything).Return(nil).
		On("Size").Return(5).
		On("Stop").Return()

	ctl := ReadWrite{
		baseController: &baseController{
			drivers: driver.NewDrivers(),
			watcher: w,
			logger:  logging.NewNullLogger(),
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

func TestReadWrite_OnceAndRun(t *testing.T) {
	// Tests Run behaviors as expected with triggers after Once completes
	d := new(mocksD.Driver)
	d.On("Task").Return(enabledTestTask(t, "task_a")).
		On("RenderTemplate", mock.Anything).Return(true, nil).
		On("ApplyTask", mock.Anything).Return(nil).
		On("SetBufferPeriod")

	ctrl := ReadWrite{
		baseController: &baseController{
			drivers: driver.NewDrivers(),
			logger:  logging.NewNullLogger(),
		},
		store: event.NewStore(),
	}
	ctrl.drivers.Add("task_a", d)

	watcherCh := make(chan error, 5)
	var watcherChRc <-chan error = watcherCh
	w := new(mocks.Watcher)
	w.On("Size").Return(5)
	ctrl.watcher = w

	completedTasksCh := ctrl.EnableTestMode()
	errCh := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		err := ctrl.Once(ctx)
		if err != nil {
			errCh <- err
			return
		}

		// Once is expected to complete during first iteration with the
		// mocked values, so we don't initialize Wait until after. Otherwise,
		// the test would panic on missing mock method.
		w.On("WaitCh", mock.Anything, mock.Anything).Return(watcherChRc)

		err = ctrl.Run(ctx)
		if err != nil {
			errCh <- err
		}
	}()

	// Emulate triggers to evaluate task completion
	for i := 0; i < 5; i++ {
		watcherCh <- nil
		select {
		case taskName := <-completedTasksCh:
			assert.Equal(t, "task_a", taskName)
		case err := <-errCh:
			require.NoError(t, err)
		case <-ctx.Done():
			assert.NoError(t, ctx.Err(), "Context should not timeout. Once and Run usage of Watcher does not match the expected triggers.")
		}
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
				Log:  config.Bool(true),
				Path: config.String("path"),
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

func multipleTaskConfig(numTasks int) *config.Config {
	tasks := make(config.TaskConfigs, numTasks)
	for i := 0; i < numTasks; i++ {
		tasks[i] = &config.TaskConfig{
			Name:     config.String(fmt.Sprintf("task_%02d", i)),
			Services: []string{fmt.Sprintf("service_%02d", i)},
			Source:   config.String("Y"),
		}
	}
	c := &config.Config{Tasks: &tasks}
	c.Finalize()
	return c
}

func enabledTestTask(tb testing.TB, name string) *driver.Task {
	task, err := driver.NewTask(driver.TaskConfig{
		Name:    name,
		Enabled: true,
	})
	require.NoError(tb, err)
	return task
}

func disabledTestTask(tb testing.TB, name string) *driver.Task {
	task, err := driver.NewTask(driver.TaskConfig{
		Name:    name,
		Enabled: false,
	})
	require.NoError(tb, err)
	return task
}

func scheduledTestTask(tb testing.TB, name string) *driver.Task {
	task, err := driver.NewTask(driver.TaskConfig{
		Name:        name,
		Description: "runs every 3 seconds",
		Enabled:     true,
		Condition: &config.ScheduleConditionConfig{
			Cron: config.String("*/3 * * * * * *"),
		},
	})
	require.NoError(tb, err)
	return task
}
