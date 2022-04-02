package controller

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/handler"
	"github.com/hashicorp/consul-terraform-sync/logging"
	mocksD "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	mocksTmpl "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/consul-terraform-sync/state"
	"github.com/hashicorp/consul-terraform-sync/state/event"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var validTaskConf = config.TaskConfig{
	Enabled: config.Bool(true),
	Name:    config.String("task"),
	Module:  config.String("module"),
	Condition: &config.CatalogServicesConditionConfig{
		CatalogServicesMonitorConfig: config.CatalogServicesMonitorConfig{Regexp: config.String("regex")},
	},
}

func TestServer_Task(t *testing.T) {
	ctx := context.Background()
	tm := TasksManager{
		baseController: &baseController{
			drivers: driver.NewDrivers(),
		},
	}

	t.Run("success", func(t *testing.T) {
		taskConf := validTaskConf
		taskConf.Finalize(config.DefaultBufferPeriodConfig(), "path")
		driverTask, err := driver.NewTask(driver.TaskConfig{
			Enabled:   true,
			Name:      *taskConf.Name,
			Module:    *taskConf.Module,
			Condition: taskConf.Condition,
			BufferPeriod: &driver.BufferPeriod{
				Min: *taskConf.BufferPeriod.Min,
				Max: *taskConf.BufferPeriod.Max,
			},
			WorkingDir:   *taskConf.WorkingDir,
			TFCWorkspace: *taskConf.TFCWorkspace,
		})
		require.NoError(t, err)

		d := new(mocksD.Driver)
		mockDriver(ctx, d, driverTask)
		err = tm.drivers.Add(*taskConf.Name, d)
		require.NoError(t, err)

		actualConf, err := tm.Task(ctx, *taskConf.Name)
		require.NoError(t, err)

		// VarFiles are not stored for the task. Set to empty array.
		actualConf.VarFiles = []string{}
		assert.Equal(t, taskConf, actualConf)
	})

	t.Run("error", func(t *testing.T) {
		// no driver setup because non-existent task

		_, err := tm.Task(ctx, "non-existent-task")
		assert.Error(t, err)
	})
}

func TestServer_TaskCreate(t *testing.T) {
	ctx := context.Background()
	conf := &config.Config{
		BufferPeriod: config.DefaultBufferPeriodConfig(),
		WorkingDir:   config.String(config.DefaultWorkingDir),
	}
	conf.Finalize()
	tm := TasksManager{
		baseController: &baseController{
			drivers: driver.NewDrivers(),
			logger:  logging.NewNullLogger(),
			watcher: new(mocksTmpl.Watcher),
			state:   state.NewInMemoryStore(conf),
		},
	}

	t.Run("success", func(t *testing.T) {
		taskConf := validTaskConf
		driverTask, err := driver.NewTask(driver.TaskConfig{
			Enabled:   true,
			Name:      *taskConf.Name,
			Module:    *taskConf.Module,
			Condition: taskConf.Condition,
			BufferPeriod: &driver.BufferPeriod{
				Min: *conf.BufferPeriod.Min,
				Max: *conf.BufferPeriod.Max,
			},
			WorkingDir: *conf.WorkingDir,
		})
		require.NoError(t, err)

		mockD := new(mocksD.Driver)
		mockD.On("SetBufferPeriod").Return()
		mockD.On("OverrideNotifier").Return()
		mockDriver(ctx, mockD, driverTask)
		tm.newDriver = func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		actual, err := tm.TaskCreate(ctx, taskConf)
		assert.NoError(t, err)
		conf, err := configFromDriverTask(driverTask)
		assert.NoError(t, err)
		assert.Equal(t, conf, actual)

		_, ok := tm.drivers.Get("task")
		assert.True(t, ok, "task should have a driver")

		events := tm.state.GetTaskEvents("task")
		assert.Len(t, events, 0, "no events stored on creation")
	})

	t.Run("invalid config", func(t *testing.T) {
		_, err := tm.TaskCreate(ctx, config.TaskConfig{
			Description: config.String("missing required fields"),
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("create error", func(t *testing.T) {
		mockD := new(mocksD.Driver)
		mockD.On("InitTask", mock.Anything).Return(fmt.Errorf("init err"))
		mockD.On("DestroyTask", mock.Anything).Return()
		tm.drivers = driver.NewDrivers()
		tm.newDriver = func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		_, err := tm.TaskCreate(ctx, validTaskConf)
		assert.Error(t, err)

		_, ok := tm.drivers.Get("task")
		assert.False(t, ok, "errored task should not have a driver added")

		events := tm.state.GetTaskEvents("task")
		assert.Len(t, events, 0, "no events stored on creation")
	})
}

func TestServer_TaskCreateAndRun(t *testing.T) {
	// TaskCreateAndRun is similar to TaskCreate but with added run behavior.
	// This tests what hasn't been testeed in TestServer_TaskCreate
	ctx := context.Background()

	tm := TasksManager{
		baseController: &baseController{
			logger:  logging.NewNullLogger(),
			watcher: new(mocksTmpl.Watcher),
		},
	}

	conf := &config.Config{
		BufferPeriod: config.DefaultBufferPeriodConfig(),
		WorkingDir:   config.String(config.DefaultWorkingDir),
		Driver:       config.DefaultDriverConfig(),
	}

	t.Run("success", func(t *testing.T) {
		mockD := new(mocksD.Driver)
		mockD.On("SetBufferPeriod").Return()
		mockD.On("OverrideNotifier").Return()
		task, err := driver.NewTask(driver.TaskConfig{
			Enabled: true,
			Name:    "task",
		})
		require.NoError(t, err)
		mockDriver(ctx, mockD, task)
		tm.state = state.NewInMemoryStore(conf)
		tm.drivers = driver.NewDrivers()
		tm.newDriver = func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		_, err = tm.TaskCreateAndRun(ctx, validTaskConf)
		assert.NoError(t, err)

		_, ok := tm.drivers.Get("task")
		assert.True(t, ok)

		events := tm.state.GetTaskEvents("task")
		assert.Len(t, events, 1)
		require.Len(t, events["task"], 1)
		assert.Nil(t, events["task"][0].EventError, "unexpected error event")
	})

	t.Run("disabled task", func(t *testing.T) {
		mockD := new(mocksD.Driver)
		mockD.On("SetBufferPeriod").Return()
		mockD.On("OverrideNotifier").Return()
		task, err := driver.NewTask(driver.TaskConfig{
			Enabled: false,
			Name:    "task",
		})
		require.NoError(t, err)
		mockDriver(ctx, mockD, task)
		tm.state = state.NewInMemoryStore(conf)
		tm.drivers = driver.NewDrivers()
		tm.newDriver = func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		taskConf := validTaskConf.Copy()
		taskConf.Enabled = config.Bool(false)

		_, err = tm.TaskCreateAndRun(ctx, *taskConf)
		assert.NoError(t, err)

		_, ok := tm.drivers.Get("task")
		assert.True(t, ok, "driver is created for task even if it's disabled")

		events := tm.state.GetTaskEvents("task")
		assert.Len(t, events, 0, "task is disabled, no run should occur")
	})

	t.Run("apply error", func(t *testing.T) {
		mockD := new(mocksD.Driver)
		task, err := driver.NewTask(driver.TaskConfig{
			Enabled: true,
			Name:    "task",
		})
		require.NoError(t, err)
		mockD.On("Task").Return(task).
			On("InitTask", ctx).Return(nil).
			On("OverrideNotifier").Return().
			On("RenderTemplate", mock.Anything).Return(true, nil).
			On("ApplyTask", ctx).Return(fmt.Errorf("apply err"))
		tm.state = state.NewInMemoryStore(conf)
		tm.drivers = driver.NewDrivers()
		tm.newDriver = func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		_, err = tm.TaskCreateAndRun(ctx, validTaskConf)
		assert.Error(t, err)

		_, ok := tm.drivers.Get("task")
		assert.False(t, ok, "driver is only added if the run is successful")

		events := tm.state.GetTaskEvents("task")
		assert.Len(t, events, 0, "event is only stored on successful creation and run")
	})
}

func TestServer_TaskDelete(t *testing.T) {
	ctx := context.Background()
	tm := TasksManager{
		baseController: &baseController{
			logger: logging.NewNullLogger(),
			state:  state.NewInMemoryStore(nil),
		},
		deleteCh: make(chan string),
	}

	t.Run("happy path", func(t *testing.T) {
		drivers := driver.NewDrivers()
		taskName := "delete_task"

		tm.baseController.drivers = drivers
		go tm.TaskDelete(ctx, taskName)
		select {
		case n := <-tm.deleteCh:
			assert.Equal(t, taskName, n)
		case <-time.After(1 * time.Second):
			t.Log("delete channel did not receive message")
			t.Fail()
		}
		assert.True(t, tm.drivers.IsMarkedForDeletion(taskName))
	})

	t.Run("already marked for deletion", func(t *testing.T) {
		drivers := driver.NewDrivers()
		taskName := "delete_task"
		tm.baseController.drivers = drivers
		tm.drivers.MarkForDeletion(taskName)
		err := tm.TaskDelete(ctx, taskName)
		assert.NoError(t, err)
		assert.True(t, tm.drivers.IsMarkedForDeletion(taskName))
	})
}

func TestServer_TaskUpdate(t *testing.T) {
	t.Parallel()

	conf := &config.Config{}
	conf.Finalize()
	ctx := context.Background()
	tm := TasksManager{
		baseController: &baseController{
			state:   state.NewInMemoryStore(conf),
			drivers: driver.NewDrivers(),
			logger:  logging.NewNullLogger(),
		},
	}

	t.Run("disable-then-enable", func(t *testing.T) {
		taskName := "task_a"
		taskConf := config.TaskConfig{
			Name:   config.String("task_a"),
			Module: config.String("findkim/print/cts"),
			Condition: &config.ServicesConditionConfig{
				ServicesMonitorConfig: config.ServicesMonitorConfig{
					Names: []string{"service"},
				},
			},
		}
		taskConf.Finalize(conf.BufferPeriod, *conf.WorkingDir)
		task, err := newDriverTask(conf, &taskConf, nil)
		require.NoError(t, err)

		d := new(mocksD.Driver)
		mockDriver(ctx, d, task)
		d.On("UpdateTask", mock.Anything, driver.PatchTask{Enabled: false}).
			Return(driver.InspectPlan{ChangesPresent: false, Plan: ""}, nil)
		err = tm.drivers.Add(taskName, d)
		require.NoError(t, err)

		// Set the task to disabled
		updateConf := config.TaskConfig{
			Name:    &taskName,
			Enabled: config.Bool(false),
		}

		changed, plan, _, err := tm.TaskUpdate(ctx, updateConf, "")
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Empty(t, plan)

		// Re-enable the task
		updateConf.Enabled = config.Bool(true)
		d.On("UpdateTask", mock.Anything, driver.PatchTask{Enabled: true}).
			Return(driver.InspectPlan{ChangesPresent: false, Plan: ""}, nil)
		_, plan, _, err = tm.TaskUpdate(ctx, updateConf, "")
		require.NoError(t, err)
		assert.Empty(t, plan)

		// No events since the task did not run
		events := tm.state.GetTaskEvents(taskName)
		assert.Empty(t, events)
	})

	t.Run("task-not-found-error", func(t *testing.T) {
		taskConf := config.TaskConfig{
			Name:    config.String("non-existent-task"),
			Enabled: config.Bool(true),
		}
		_, plan, _, err := tm.TaskUpdate(ctx, taskConf, "")
		require.Error(t, err)
		assert.Empty(t, plan)
	})

	t.Run("task-run-inspect", func(t *testing.T) {
		expectedPlan := driver.InspectPlan{
			ChangesPresent: true,
			Plan:           "plan!",
		}
		// add a driver
		d := new(mocksD.Driver)
		mockDriver(ctx, d, &driver.Task{})
		d.On("UpdateTask", mock.Anything, mock.Anything).Return(expectedPlan, nil).Once()
		err := tm.drivers.Add("task_b", d)
		require.NoError(t, err)

		updateConf := config.TaskConfig{
			Name:    config.String("task_b"),
			Enabled: config.Bool(true),
		}

		changed, plan, _, err := tm.TaskUpdate(ctx, updateConf, driver.RunOptionInspect)

		require.NoError(t, err)
		assert.Equal(t, expectedPlan.Plan, plan)
		assert.Equal(t, expectedPlan.ChangesPresent, changed)

		// No events since the task did not run
		events := tm.state.GetTaskEvents("task_b")
		assert.Empty(t, events)
	})

	t.Run("task-run-now", func(t *testing.T) {
		taskName := "task_c"

		// add a driver
		d := new(mocksD.Driver)
		mockDriver(ctx, d, &driver.Task{})
		d.On("UpdateTask", mock.Anything, mock.Anything).Return(driver.InspectPlan{}, nil).Once()
		err := tm.drivers.Add(taskName, d)
		require.NoError(t, err)

		updateConf := config.TaskConfig{
			Name:    &taskName,
			Enabled: config.Bool(true),
		}

		changed, plan, _, err := tm.TaskUpdate(ctx, updateConf, driver.RunOptionNow)

		require.NoError(t, err)
		assert.Equal(t, "", plan, "run now does not return plan info")
		assert.False(t, changed, "run now does not return plan info")

		events := tm.state.GetTaskEvents(taskName)
		assert.Len(t, events, 1)
	})
}

// mockDriver sets up a mock driver with the happy path for all methods
func mockDriver(ctx context.Context, d *mocksD.Driver, task *driver.Task) {
	d.On("Task").Return(task).
		On("InitTask", ctx).Return(nil).
		On("TemplateIDs").Return(nil).
		On("RenderTemplate", mock.Anything).Return(true, nil).
		On("ApplyTask", ctx).Return(nil)
}

func TestTasksManager_CheckApply(t *testing.T) {
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
			d.On("TemplateIDs").Return(nil)
			drivers.Add(tc.taskName, d)

			controller := newTestController()
			ctx := context.Background()

			_, err := controller.checkApply(ctx, d, false, false)
			data := controller.state.GetTaskEvents(tc.taskName)
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

		controller := newTestController()

		d := new(mocksD.Driver)
		taskName := "scheduled_task"
		d.On("Task").Return(scheduledTestTask(t, taskName))
		d.On("TemplateIDs").Return(nil)
		d.On("RenderTemplate", mock.Anything).Return(false, nil)
		controller.drivers.Add(taskName, d)

		// Once-mode - confirm no events are stored
		ctx := context.Background()
		_, err := controller.checkApply(ctx, d, false, true)
		assert.NoError(t, err)
		data := controller.state.GetTaskEvents(taskName)
		events := data[taskName]
		assert.Equal(t, 0, len(events))

		// Daemon-mode - confirm an event is stored
		_, err = controller.checkApply(ctx, d, false, false)
		assert.NoError(t, err)
		data = controller.state.GetTaskEvents(taskName)
		events = data[taskName]
		assert.Equal(t, 1, len(events))
	})
}

func TestTasksManager_CheckApply_Store(t *testing.T) {
	t.Run("mult-checkapply-store", func(t *testing.T) {
		d := new(mocksD.Driver)
		d.On("Task").Return(enabledTestTask(t, "task_a"))
		d.On("TemplateIDs").Return(nil)
		d.On("RenderTemplate", mock.Anything).Return(true, nil)
		d.On("ApplyTask", mock.Anything).Return(nil)

		disabledD := new(mocksD.Driver)
		disabledD.On("Task").Return(disabledTestTask(t, "task_b"))
		disabledD.On("TemplateIDs").Return(nil)

		controller := newTestController()

		controller.drivers.Add("task_a", d)
		controller.drivers.Add("task_b", disabledD)
		ctx := context.Background()

		controller.checkApply(ctx, d, false, false)
		controller.checkApply(ctx, disabledD, false, false)
		controller.checkApply(ctx, d, false, false)
		controller.checkApply(ctx, d, false, false)
		controller.checkApply(ctx, d, false, false)
		controller.checkApply(ctx, disabledD, false, false)

		taskStatuses := controller.state.GetTaskEvents("")

		assert.Equal(t, 4, len(taskStatuses["task_a"]))
		assert.Equal(t, 0, len(taskStatuses["task_b"]))
	})
}

func Test_once(t *testing.T) {
	rw := &TasksManager{}

	testCases := []struct {
		name     string
		onceFxn  func(context.Context) error
		numTasks int
	}{
		{
			"consecutive one task",
			rw.onceConsecutive,
			1,
		},
		{
			"consecutive multiple tasks",
			rw.onceConsecutive,
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

			// Set up read-write controller with mocks
			conf := multipleTaskConfig(tc.numTasks)
			rw.baseController = &baseController{
				state:   state.NewInMemoryStore(conf),
				watcher: w,
				drivers: driver.NewDrivers(),
				newDriver: func(c *config.Config, task *driver.Task, w templates.Watcher) (driver.Driver, error) {
					taskName := task.Name()
					d := new(mocksD.Driver)
					d.On("Task").Return(enabledTestTask(t, taskName)).Twice()
					d.On("TemplateIDs").Return(nil)
					d.On("RenderTemplate", mock.Anything).Return(false, nil).Once()
					d.On("RenderTemplate", mock.Anything).Return(true, nil).Once()
					d.On("InitTask", mock.Anything, mock.Anything).Return(nil).Once()
					d.On("ApplyTask", mock.Anything).Return(nil).Once()
					return d, nil
				},
				initConf: conf,
				logger:   logging.NewNullLogger(),
			}

			ctx := context.Background()
			err := rw.Init(ctx)
			require.NoError(t, err)

			// testing really starts here...
			done := make(chan error)
			// running in goroutine so I can timeout
			go func() {
				done <- tc.onceFxn(ctx)
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

func TestTasksManager_once_error(t *testing.T) {
	// Test once mode error handling when a driver returns an error
	numTasks := 5
	w := new(mocks.Watcher)
	w.On("WaitCh", mock.Anything).Return(nil)
	w.On("Size").Return(numTasks)

	rw := &TasksManager{}

	testCases := []struct {
		name    string
		onceFxn func(context.Context) error
	}{
		{
			"onceConsecutive",
			rw.onceConsecutive,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expectedErr := fmt.Errorf("test error")

			// Set up read-write controller with mocks
			conf := multipleTaskConfig(numTasks)
			rw.baseController = &baseController{
				state:   state.NewInMemoryStore(conf),
				watcher: w,
				drivers: driver.NewDrivers(),
				newDriver: func(c *config.Config, task *driver.Task, w templates.Watcher) (driver.Driver, error) {
					taskName := task.Name()
					d := new(mocksD.Driver)
					d.On("Task").Return(enabledTestTask(t, taskName))
					d.On("TemplateIDs").Return(nil)
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
				initConf: conf,
				logger:   logging.NewNullLogger(),
			}

			ctx := context.Background()
			err := rw.Init(ctx)
			require.NoError(t, err)

			// testing really starts here...
			done := make(chan error)
			// running in goroutine so I can timeout
			go func() {
				done <- tc.onceFxn(ctx)
			}()
			select {
			case err := <-done:
				assert.Error(t, err, "task_03 driver error should bubble up")
				assert.Contains(t, err.Error(), expectedErr.Error(), "unexpected error in Once")
			case <-time.After(time.Second):
				t.Fatal("Once didn't return in expected time")
			}
		})
	}
}

func TestTasksManager_runDynamicTask(t *testing.T) {
	t.Run("simple-success", func(t *testing.T) {
		controller := newTestController()

		ctx := context.Background()
		d := new(mocksD.Driver)
		mockDriver(ctx, d, enabledTestTask(t, "task"))
		controller.drivers.Add("task", d)

		err := controller.runDynamicTask(ctx, d)
		assert.NoError(t, err)
	})

	t.Run("apply-error", func(t *testing.T) {
		controller := newTestController()

		testErr := fmt.Errorf("could not apply: %s", "test")
		d := new(mocksD.Driver)
		d.On("Task").Return(enabledTestTask(t, "task"))
		d.On("TemplateIDs").Return(nil)
		d.On("InitWork", mock.Anything).Return(nil)
		d.On("RenderTemplate", mock.Anything).Return(true, nil)
		d.On("ApplyTask", mock.Anything).Return(testErr)
		controller.drivers.Add("task", d)

		err := controller.runDynamicTask(context.Background(), d)
		assert.Contains(t, err.Error(), testErr.Error())
	})

	t.Run("skip-scheduled-tasks", func(t *testing.T) {
		controller := newTestController()

		taskName := "scheduled_task"
		d := new(mocksD.Driver)
		d.On("Task").Return(scheduledTestTask(t, taskName))
		d.On("TemplateIDs").Return(nil)
		// no other methods should be called (or mocked)
		controller.drivers.Add(taskName, d)

		err := controller.runDynamicTask(context.Background(), d)
		assert.NoError(t, err)
	})

	t.Run("active-task", func(t *testing.T) {
		controller := newTestController()
		controller.EnableTestMode()

		ctx := context.Background()
		d := new(mocksD.Driver)
		taskName := "task"
		mockDriver(ctx, d, enabledTestTask(t, taskName))
		drivers := controller.drivers
		drivers.Add(taskName, d)
		drivers.SetActive(taskName)

		// Attempt to run the active task
		ch := make(chan error)
		go func() {
			err := controller.runDynamicTask(ctx, d)
			ch <- err
		}()

		// Check that the task did not run while active
		select {
		case <-controller.taskNotify:
			t.Fatal("task ran even though active")
		case <-time.After(250 * time.Millisecond):
			break
		}

		// Set task to inactive, wait for run to happen
		drivers.SetInactive(taskName)
		select {
		case <-time.After(250 * time.Millisecond):
			t.Fatal("task did not run after it became inactive")
		case <-controller.taskNotify:
			break
		}
	})

}

func TestTasksManager_runScheduledTask(t *testing.T) {
	t.Run("happy-path", func(t *testing.T) {
		tm := newTestController()

		taskName := "scheduled_task"
		d := new(mocksD.Driver)
		d.On("Task").Return(scheduledTestTask(t, taskName)).Twice()
		d.On("RenderTemplate", mock.Anything).Return(true, nil).Once()
		d.On("ApplyTask", mock.Anything).Return(nil).Once()
		d.On("TemplateIDs").Return(nil)
		tm.drivers.Add(taskName, d)

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error)
		stopCh := make(chan struct{}, 1)
		go func() {
			err := tm.runScheduledTask(ctx, d, stopCh)
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
		tm := newTestController()

		taskName := "dynamic_task"
		d := new(mocksD.Driver)
		d.On("Task").Return(enabledTestTask(t, taskName)).Once()

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error)
		stopCh := make(chan struct{}, 1)
		go func() {
			err := tm.runScheduledTask(ctx, d, stopCh)
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

	t.Run("stop-scheduled-task", func(t *testing.T) {
		// Tests that signaling to the stop channel stops the function
		tm := newTestController()

		taskName := "scheduled_task"
		d := new(mocksD.Driver)
		d.On("Task").Return(scheduledTestTask(t, taskName)).Once()
		d.On("TemplateIDs").Return(nil)
		tm.drivers.Add(taskName, d)

		ctx := context.Background()
		errCh := make(chan error)
		stopCh := make(chan struct{}, 1)
		go func() {
			err := tm.runScheduledTask(ctx, d, stopCh)
			errCh <- err
		}()
		stopCh <- struct{}{}

		select {
		case err := <-errCh:
			assert.NoError(t, err)
		case <-time.After(time.Second * 5):
			t.Fatal("runScheduledTask did not exit as expected")
		}
	})

	t.Run("deleted-scheduled-task", func(t *testing.T) {
		// Tests that a scheduled task stops if it no longer is in the
		// list of drivers
		tm := newTestController()

		taskName := "scheduled_task"
		d := new(mocksD.Driver)
		d.On("Task").Return(scheduledTestTask(t, taskName)).Once()
		// driver is not added to drivers map

		ctx := context.Background()
		errCh := make(chan error)
		stopCh := make(chan struct{}, 1)
		tm.scheduleStopChs[taskName] = stopCh
		done := make(chan bool)
		go func() {
			err := tm.runScheduledTask(ctx, d, stopCh)
			if err != nil {
				errCh <- err
			}
			done <- true
		}()

		select {
		case <-errCh:
			t.Fatal("runScheduledTask did not exit properly when task is not in map of drivers")
		case <-done:
			// runScheduledTask exited as expected
			d.AssertExpectations(t)
			_, ok := tm.scheduleStopChs[taskName]
			assert.False(t, ok, "expected scheduled task stop channel to be removed")
		case <-time.After(time.Second * 5):
			t.Fatal("runScheduledTask did not exit as expected")
		}
	})
}

func TestTasksManagerRun_context_cancel(t *testing.T) {
	w := new(mocks.Watcher)
	w.On("Watch", mock.Anything, mock.Anything).Return(nil).
		On("Size").Return(5).
		On("Stop").Return()

	ctl := TasksManager{
		baseController: &baseController{
			drivers: driver.NewDrivers(),
			watcher: w,
			logger:  logging.NewNullLogger(),
			state:   state.NewInMemoryStore(nil),
		},
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

func TestTasksManager_once_then_Run(t *testing.T) {
	// Tests Run behaviors as expected with triggers after once completes

	d := new(mocksD.Driver)
	d.On("Task").Return(enabledTestTask(t, "task_a")).
		On("TemplateIDs").Return([]string{"tmpl_a"}).
		On("RenderTemplate", mock.Anything).Return(true, nil).
		On("ApplyTask", mock.Anything).Return(nil).
		On("SetBufferPeriod")

	tm := TasksManager{
		baseController: &baseController{
			drivers: driver.NewDrivers(),
			logger:  logging.NewNullLogger(),
			state:   state.NewInMemoryStore(nil),
		},
		watcherCh: make(chan string, 5),
	}
	tm.drivers.Add("task_a", d)

	testCases := []struct {
		name    string
		onceFxn func(context.Context) error
	}{
		{
			"onceConsecutive",
			tm.onceConsecutive,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			completedTasksCh := tm.EnableTestMode()
			errCh := make(chan error)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			w := new(mocks.Watcher)
			w.On("Size").Return(5)
			w.On("Watch", ctx, tm.watcherCh).Return(nil)
			tm.watcher = w

			go func() {
				err := tc.onceFxn(ctx)
				if err != nil {
					errCh <- err
					return
				}

				err = tm.Run(ctx)
				if err != nil {
					errCh <- err
				}
			}()

			// Emulate triggers to evaluate task completion
			for i := 0; i < 5; i++ {
				tm.watcherCh <- "tmpl_a"
				select {
				case taskName := <-completedTasksCh:
					assert.Equal(t, "task_a", taskName)
				case err := <-errCh:
					require.NoError(t, err)
				case <-ctx.Done():
					assert.NoError(t, ctx.Err(), "Context should not timeout. Once and Run usage of Watcher does not match the expected triggers.")
				}
			}
		})
	}
}

func TestTasksManager_Run_ActiveTask(t *testing.T) {
	// Set up controller with two tasks
	tm := TasksManager{
		baseController: &baseController{
			drivers: driver.NewDrivers(),
			logger:  logging.NewNullLogger(),
			state:   state.NewInMemoryStore(nil),
		},
		watcherCh: make(chan string, 5),
	}
	for _, n := range []string{"task_a", "task_b"} {
		d := new(mocksD.Driver)
		d.On("Task").Return(enabledTestTask(t, n)).
			On("TemplateIDs").Return([]string{"tmpl_" + n}).
			On("RenderTemplate", mock.Anything).Return(true, nil).
			On("ApplyTask", mock.Anything).Return(nil).
			On("SetBufferPeriod")
		tm.drivers.Add(n, d)
	}
	completedTasksCh := tm.EnableTestMode()

	// Set up watcher for controller
	ctx := context.Background()
	w := new(mocks.Watcher)
	w.On("Size").Return(5)
	w.On("Watch", ctx, tm.watcherCh).Return(nil)
	tm.watcher = w

	// Start Run
	errCh := make(chan error)
	go func() {
		err := tm.Run(ctx)
		if err != nil {
			errCh <- err
		}
	}()

	// Set task_a to active
	tm.drivers.SetActive("task_a")

	// Trigger twice on active task_a, task should not complete
	for i := 0; i < 2; i++ {
		tm.watcherCh <- "tmpl_task_a"
	}
	select {
	case <-completedTasksCh:
		t.Fatal("task should not have completed")
	case <-time.After(time.Millisecond * 250):
		break // expected case
	}

	// Trigger on inactive task_b, task should complete
	tm.watcherCh <- "tmpl_task_b"
	select {
	case taskName := <-completedTasksCh:
		assert.Equal(t, "task_b", taskName)
	case <-time.After(time.Millisecond * 250):
		t.Fatal("task should have completed")
	}

	// Set task_a to inactive, should expect two tasks to complete
	tm.drivers.SetInactive("task_a")
	for i := 0; i < 2; i++ {
		select {
		case taskName := <-completedTasksCh:
			assert.Equal(t, "task_a", taskName)
		case <-time.After(time.Millisecond * 250):
			t.Fatal("task should have completed")
		}
	}

	// Notify on task_a again, should complete
	tm.watcherCh <- "tmpl_task_a"
	select {
	case taskName := <-completedTasksCh:
		assert.Equal(t, "task_a", taskName)
	case <-time.After(time.Millisecond * 250):
		t.Fatal("task should have completed")
	}
}

func TestTasksManager_Run_ScheduledTasks(t *testing.T) {
	t.Run("startup_task", func(t *testing.T) {
		tm := TasksManager{
			baseController: &baseController{
				drivers: driver.NewDrivers(),
				logger:  logging.NewNullLogger(),
				state:   state.NewInMemoryStore(nil),
			},
			watcherCh:       make(chan string, 5),
			scheduleStopChs: make(map[string](chan struct{})),
		}
		tm.EnableTestMode()

		// Add initial task
		taskName := "scheduled_task"
		d := new(mocksD.Driver)
		d.On("Task").Return(scheduledTestTask(t, taskName)).
			On("TemplateIDs").Return([]string{"tmpl_a"}).
			On("RenderTemplate", mock.Anything).Return(true, nil).
			On("ApplyTask", mock.Anything).Return(nil).
			On("SetBufferPeriod")
		tm.drivers.Add(taskName, d)

		// Set up watcher for controller
		ctx := context.Background()
		w := new(mocks.Watcher)
		w.On("Size").Return(5)
		w.On("Watch", ctx, tm.watcherCh).Return(nil)
		tm.watcher = w

		go tm.Run(context.Background())

		// Check that the task ran
		select {
		case n := <-tm.taskNotify:
			assert.Equal(t, taskName, n)
		case <-time.After(5 * time.Second):
			t.Fatal("scheduled task did not run")
		}

		stopCh, ok := tm.scheduleStopChs[taskName]
		assert.True(t, ok, "scheduled task stop channel not added to map")
		assert.NotNil(t, stopCh, "expected stop channel not to be nil")
	})

	t.Run("created_task", func(t *testing.T) {
		tm := TasksManager{
			baseController: &baseController{
				drivers: driver.NewDrivers(),
				logger:  logging.NewNullLogger(),
				state:   state.NewInMemoryStore(nil),
			},
			watcherCh:       make(chan string, 5),
			scheduleStartCh: make(chan driver.Driver, 1),
			scheduleStopChs: make(map[string](chan struct{})),
		}
		tm.EnableTestMode()

		// Set up watcher for controller
		ctx := context.Background()
		w := new(mocks.Watcher)
		w.On("Size").Return(5)
		w.On("Watch", ctx, tm.watcherCh).Return(nil)
		tm.watcher = w

		go tm.Run(context.Background())

		createdTaskName := "created_scheduled_task"
		createdDriver := new(mocksD.Driver)
		createdDriver.On("Task").Return(scheduledTestTask(t, createdTaskName)).
			On("TemplateIDs").Return([]string{"tmpl_b"}).
			On("RenderTemplate", mock.Anything).Return(true, nil).
			On("ApplyTask", mock.Anything).Return(nil).
			On("SetBufferPeriod")
		tm.drivers.Add(createdTaskName, createdDriver)
		tm.scheduleStartCh <- createdDriver

		select {
		case n := <-tm.taskNotify:
			assert.Equal(t, createdTaskName, n)
		case <-time.After(5 * time.Second):
			t.Fatal("scheduled task did not run")
		}
		stopCh, ok := tm.scheduleStopChs[createdTaskName]
		assert.True(t, ok, "scheduled task stop channel not added to map")
		assert.NotNil(t, stopCh, "expected stop channel not to be nil")
	})
}

func TestTasksManager_deleteTask(t *testing.T) {
	ctx := context.Background()
	mockD := new(mocksD.Driver)

	testCases := []struct {
		name  string
		setup func(*driver.Drivers)
	}{
		{
			"success",
			func(d *driver.Drivers) {
				mockD.On("TemplateIDs").Return(nil)
				d.Add("success", mockD)
				mockD.On("Task").Return(enabledTestTask(t, "success"))
				mockD.On("DestroyTask", ctx).Return()
			},
		},
		{
			"does_not_exist",
			func(*driver.Drivers) {},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			drivers := driver.NewDrivers()

			tc.setup(drivers)
			tm := TasksManager{
				baseController: &baseController{
					logger: logging.NewNullLogger(),
					state:  state.NewInMemoryStore(nil),
				},
			}
			tm.baseController.drivers = drivers
			tm.state.AddTaskEvent(event.Event{TaskName: "success"})

			err := tm.deleteTask(ctx, tc.name)

			assert.NoError(t, err)
			_, exists := drivers.Get(tc.name)
			assert.False(t, exists, "task should no longer exist")
			events := tm.state.GetTaskEvents(tc.name)
			assert.Empty(t, events, "task events should no longer exist")
		})
	}

	t.Run("scheduled_task", func(t *testing.T) {
		// Tests that deleting a scheduled task sends a stop notification

		// Setup controller and drivers
		taskName := "scheduled_task"
		scheduledDriver := new(mocksD.Driver)
		scheduledDriver.On("Task").Return(scheduledTestTask(t, taskName))
		scheduledDriver.On("DestroyTask", ctx).Return()
		scheduledDriver.On("TemplateIDs").Return(nil)
		tm := newTestController()
		tm.drivers.Add(taskName, scheduledDriver)
		stopCh := make(chan struct{}, 1)
		tm.scheduleStopChs[taskName] = stopCh

		// Delete task
		err := tm.deleteTask(ctx, taskName)
		assert.NoError(t, err)

		// Verify the stop channel received message
		select {
		case <-time.After(1 * time.Second):
			t.Fatal("scheduled task was not notified to stop")
		case <-stopCh:
			break // expected case
		}
		_, ok := tm.scheduleStopChs[taskName]
		assert.False(t, ok, "scheduled task stop channel still in map")
	})

	t.Run("active_task", func(t *testing.T) {
		// Set up drivers with active task
		drivers := driver.NewDrivers()
		taskName := "active_task"
		activeDriver := new(mocksD.Driver)
		activeDriver.On("Task").Return(enabledTestTask(t, taskName))
		activeDriver.On("DestroyTask", ctx).Return()
		activeDriver.On("TemplateIDs").Return(nil)
		drivers.Add(taskName, activeDriver)
		drivers.SetActive(taskName)

		// Set up controller with drivers and store
		tm := TasksManager{
			baseController: &baseController{
				logger: logging.NewNullLogger(),
				state:  state.NewInMemoryStore(nil),
			},
		}
		tm.baseController.drivers = drivers
		tm.state.AddTaskEvent(event.Event{TaskName: taskName})

		// Attempt to delete the active task
		ch := make(chan error)
		go func() {
			err := tm.deleteTask(ctx, taskName)
			ch <- err
		}()

		// Check that the task is not deleted while active
		time.Sleep(500 * time.Millisecond)
		_, exists := drivers.Get(taskName)
		assert.True(t, exists, "task deleted when active")
		events := tm.state.GetTaskEvents(taskName)
		assert.NotEmpty(t, events, "task events should still exist")

		// Set task to inactive, wait for deletion to happen
		drivers.SetInactive(taskName)
		select {
		case err := <-ch:
			assert.NoError(t, err)
		case <-time.After(1 * time.Second):
			t.Fatal("task was not deleted after it became inactive")
		}

		// Check that task removed from drivers and store
		_, exists = drivers.Get(taskName)
		assert.False(t, exists, "task should no longer exist")
		events = tm.state.GetTaskEvents(taskName)
		assert.Empty(t, events, "task events should no longer exist")
	})

}
func TestTasksManager_waitForTaskInactive(t *testing.T) {
	ctx := context.Background()
	t.Run("active_task", func(t *testing.T) {
		// Set up drivers with active task
		tm := newTestController()
		taskName := "inactive_task"
		mockD := new(mocksD.Driver)
		mockD.On("TemplateIDs").Return(nil)
		tm.drivers.Add(taskName, mockD)
		tm.drivers.SetActive(taskName)

		// Wait for task to become inactive
		ch := make(chan error)
		go func() {
			err := tm.waitForTaskInactive(ctx, taskName)
			ch <- err
		}()

		// Check that the wait does not return early
		select {
		case <-ch:
			t.Fatal("wait completed when task was still active")
		case <-time.After(250 * time.Millisecond):
			break
		}

		// Set task to inactive, wait should complete
		tm.drivers.SetInactive(taskName)
		select {
		case err := <-ch:
			assert.NoError(t, err)
		case <-time.After(1 * time.Second):
			t.Fatal("wait should have completed because task is inactive")
		}
	})

	t.Run("inactive_task", func(t *testing.T) {
		// Set up drivers with inactive task
		tm := newTestController()
		taskName := "inactive_task"
		mockD := new(mocksD.Driver)
		mockD.On("TemplateIDs").Return(nil)
		tm.drivers.Add(taskName, mockD)
		tm.drivers.SetInactive(taskName)

		// Wait for task to be inactive, should return immediately
		err := tm.waitForTaskInactive(ctx, taskName)
		require.NoError(t, err)
	})
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
				Description:        config.String("automate services for X to do Y"),
				Name:               config.String("task"),
				DeprecatedServices: []string{"serviceA", "serviceB", "serviceC"},
				Providers:          []string{"X", handler.TerraformProviderFake},
				Module:             config.String("Y"),
				Version:            config.String("v1"),
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
			Name: config.String(fmt.Sprintf("task_%02d", i)),
			Condition: &config.ServicesConditionConfig{
				ServicesMonitorConfig: config.ServicesMonitorConfig{
					Names: []string{fmt.Sprintf("service_%02d", i)},
				},
			},
			Module: config.String("Y"),
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

func newTestController() TasksManager {
	return TasksManager{
		baseController: &baseController{
			drivers: driver.NewDrivers(),
			logger:  logging.NewNullLogger(),
			state:   state.NewInMemoryStore(nil),
		},
		scheduleStopChs: make(map[string](chan struct{})),
	}
}

func TestTasksManagerRun(t *testing.T) {
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

			tm := &TasksManager{baseController: &baseController{
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
			err := tm.drivers.Add("task", d)
			require.NoError(t, err)

			err = tm.Run(context.Background())
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

func TestTasksManagerRun_context_cancel(t *testing.T) {
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

	tm := TasksManager{baseController: &baseController{
		watcher:  w,
		resolver: r,
		drivers:  drivers,
		logger:   logging.NewNullLogger(),
	}}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		err := tm.Run(ctx)
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
