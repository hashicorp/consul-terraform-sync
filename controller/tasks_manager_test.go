package controller

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/logging"
	mocksD "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	mocksS "github.com/hashicorp/consul-terraform-sync/mocks/store"
	mocksTmpl "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/consul-terraform-sync/state"
	"github.com/hashicorp/consul-terraform-sync/state/event"
	"github.com/hashicorp/consul-terraform-sync/templates"
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
	ctrl := ReadWrite{
		baseController: &baseController{},
	}

	t.Run("success", func(t *testing.T) {
		taskConf := validTaskConf
		taskConf.Finalize(config.DefaultBufferPeriodConfig(), "path")

		s := new(mocksS.Store)
		s.On("GetTask", mock.Anything).Return(taskConf, true)
		ctrl.state = s

		actualConf, err := ctrl.Task(ctx, *taskConf.Name)
		require.NoError(t, err)

		assert.Equal(t, taskConf, actualConf)

		s.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		s := new(mocksS.Store)
		s.On("GetTask", mock.Anything, mock.Anything).Return(config.TaskConfig{}, false)
		ctrl.state = s

		_, err := ctrl.Task(ctx, "non-existent-task")
		assert.Error(t, err)

		s.AssertExpectations(t)
	})
}

func TestServer_Tasks(t *testing.T) {
	ctx := context.Background()
	ctrl := ReadWrite{
		baseController: &baseController{},
	}

	t.Run("success", func(t *testing.T) {
		taskConfs := config.TaskConfigs{
			{Name: config.String("task_a")},
			{Name: config.String("task_b")},
		}
		taskConfs.Finalize(config.DefaultBufferPeriodConfig(), config.DefaultWorkingDir)

		s := new(mocksS.Store)
		s.On("GetAllTasks", mock.Anything, mock.Anything).Return(taskConfs)
		ctrl.state = s

		actualConfs, err := ctrl.Tasks(ctx)
		require.NoError(t, err)

		assert.Len(t, actualConfs, taskConfs.Len())
		for ix, expectedConf := range taskConfs {
			assert.Equal(t, *expectedConf, actualConfs[ix])
		}

		s.AssertExpectations(t)
	})

	t.Run("empty return", func(t *testing.T) {
		s := new(mocksS.Store)
		s.On("GetAllTasks", mock.Anything, mock.Anything).Return(config.TaskConfigs{})
		ctrl.state = s

		actualConfs, err := ctrl.Tasks(ctx)
		require.NoError(t, err)
		assert.Equal(t, []config.TaskConfig{}, actualConfs)

		s.AssertExpectations(t)
	})
}

func TestServer_TaskCreate(t *testing.T) {
	ctx := context.Background()
	conf := &config.Config{
		BufferPeriod: config.DefaultBufferPeriodConfig(),
		WorkingDir:   config.String(config.DefaultWorkingDir),
	}
	conf.Finalize()
	ctrl := ReadWrite{
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
		ctrl.newDriver = func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		actual, err := ctrl.TaskCreate(ctx, taskConf)
		assert.NoError(t, err)
		conf, err := configFromDriverTask(driverTask)
		assert.NoError(t, err)
		assert.Equal(t, conf, actual)

		// Basic check that task was added
		_, ok := ctrl.drivers.Get("task")
		assert.True(t, ok, "task should have a driver")

		// Basic check that task was not run
		events := ctrl.state.GetTaskEvents("task")
		assert.Len(t, events, 0, "no events stored on creation")
	})

	t.Run("invalid config", func(t *testing.T) {
		_, err := ctrl.TaskCreate(ctx, config.TaskConfig{
			Description: config.String("missing required fields"),
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("create error", func(t *testing.T) {
		mockD := new(mocksD.Driver)
		mockD.On("InitTask", mock.Anything).Return(fmt.Errorf("init err"))
		mockD.On("DestroyTask", mock.Anything).Return()
		ctrl.drivers = driver.NewDrivers()
		ctrl.newDriver = func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		_, err := ctrl.TaskCreate(ctx, validTaskConf)
		assert.Error(t, err)

		_, ok := ctrl.drivers.Get("task")
		assert.False(t, ok, "errored task should not have a driver added")

		events := ctrl.state.GetTaskEvents("task")
		assert.Len(t, events, 0, "no events stored on creation")
	})
}

func TestServer_TaskCreateAndRun(t *testing.T) {
	// TaskCreateAndRun is similar to TaskCreate but with added run behavior.
	// This tests what hasn't been testeed in TestServer_TaskCreate
	ctx := context.Background()

	ctrl := ReadWrite{
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
		ctrl.state = state.NewInMemoryStore(conf)
		ctrl.drivers = driver.NewDrivers()
		ctrl.newDriver = func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		_, err = ctrl.TaskCreateAndRun(ctx, validTaskConf)
		assert.NoError(t, err)

		// Basic check that task was added
		_, ok := ctrl.drivers.Get("task")
		assert.True(t, ok)

		// Basic check that task was ran
		events := ctrl.state.GetTaskEvents("task")
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
		ctrl.state = state.NewInMemoryStore(conf)
		ctrl.drivers = driver.NewDrivers()
		ctrl.newDriver = func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		taskConf := validTaskConf.Copy()
		taskConf.Enabled = config.Bool(false)

		_, err = ctrl.TaskCreateAndRun(ctx, *taskConf)
		assert.NoError(t, err)

		_, ok := ctrl.drivers.Get("task")
		assert.True(t, ok, "driver is created for task even if it's disabled")

		events := ctrl.state.GetTaskEvents("task")
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
		ctrl.state = state.NewInMemoryStore(conf)
		ctrl.drivers = driver.NewDrivers()
		ctrl.newDriver = func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		_, err = ctrl.TaskCreateAndRun(ctx, validTaskConf)
		assert.Error(t, err)

		_, ok := ctrl.drivers.Get("task")
		assert.False(t, ok, "driver is only added if the run is successful")

		events := ctrl.state.GetTaskEvents("task")
		assert.Len(t, events, 0, "event is only stored on successful creation and run")
	})
}

func TestServer_TaskDelete(t *testing.T) {
	ctx := context.Background()
	ctrl := ReadWrite{
		baseController: &baseController{
			logger: logging.NewNullLogger(),
			state:  state.NewInMemoryStore(nil),
		},
		deleteCh: make(chan string),
	}

	t.Run("happy path", func(t *testing.T) {
		drivers := driver.NewDrivers()
		taskName := "delete_task"

		ctrl.drivers = drivers
		go ctrl.TaskDelete(ctx, taskName)
		select {
		case n := <-ctrl.deleteCh:
			assert.Equal(t, taskName, n)
		case <-time.After(1 * time.Second):
			t.Log("delete channel did not receive message")
			t.Fail()
		}
		assert.True(t, ctrl.drivers.IsMarkedForDeletion(taskName))
	})

	t.Run("already marked for deletion", func(t *testing.T) {
		drivers := driver.NewDrivers()
		taskName := "delete_task"
		ctrl.drivers = drivers
		ctrl.drivers.MarkForDeletion(taskName)
		err := ctrl.TaskDelete(ctx, taskName)
		assert.NoError(t, err)
		assert.True(t, ctrl.drivers.IsMarkedForDeletion(taskName))
	})
}

func TestServer_TaskUpdate(t *testing.T) {
	t.Parallel()

	conf := &config.Config{}
	conf.Finalize()
	ctx := context.Background()
	ctrl := ReadWrite{
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
		err = ctrl.drivers.Add(taskName, d)
		require.NoError(t, err)

		// Set the task to disabled
		updateConf := config.TaskConfig{
			Name:    &taskName,
			Enabled: config.Bool(false),
		}

		changed, plan, _, err := ctrl.TaskUpdate(ctx, updateConf, "")
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Empty(t, plan)

		// Re-enable the task
		updateConf.Enabled = config.Bool(true)
		d.On("UpdateTask", mock.Anything, driver.PatchTask{Enabled: true}).
			Return(driver.InspectPlan{ChangesPresent: false, Plan: ""}, nil)
		_, plan, _, err = ctrl.TaskUpdate(ctx, updateConf, "")
		require.NoError(t, err)
		assert.Empty(t, plan)

		// No events since the task did not run
		events := ctrl.state.GetTaskEvents(taskName)
		assert.Empty(t, events)
	})

	t.Run("task-not-found-error", func(t *testing.T) {
		taskConf := config.TaskConfig{
			Name:    config.String("non-existent-task"),
			Enabled: config.Bool(true),
		}
		_, plan, _, err := ctrl.TaskUpdate(ctx, taskConf, "")
		require.Error(t, err)
		assert.Empty(t, plan)
	})

	t.Run("task-run-inspect", func(t *testing.T) {
		expectedPlan := driver.InspectPlan{
			ChangesPresent: true,
			Plan:           "plan!",
		}

		taskName := "task_b"

		// add a driver
		d := new(mocksD.Driver)
		mockDriver(ctx, d, &driver.Task{})
		d.On("UpdateTask", mock.Anything, mock.Anything).Return(expectedPlan, nil).Once()
		err := ctrl.drivers.Add(taskName, d)
		require.NoError(t, err)

		// add to state
		ctrl.state.SetTask(config.TaskConfig{
			Name:    &taskName,
			Enabled: config.Bool(false),
		})

		updateConf := config.TaskConfig{
			Name:    config.String(taskName),
			Enabled: config.Bool(true),
		}

		changed, plan, _, err := ctrl.TaskUpdate(ctx, updateConf, driver.RunOptionInspect)

		require.NoError(t, err)
		assert.Equal(t, expectedPlan.Plan, plan)
		assert.Equal(t, expectedPlan.ChangesPresent, changed)

		// No events since the task did not run
		events := ctrl.state.GetTaskEvents(taskName)
		assert.Empty(t, events)

		// Confirm task stayed disabled in state
		stateTask, exists := ctrl.state.GetTask(taskName)
		require.True(t, exists)
		assert.False(t, *stateTask.Enabled)
	})

	t.Run("task-run-now", func(t *testing.T) {
		taskName := "task_c"

		// add a driver
		d := new(mocksD.Driver)
		mockDriver(ctx, d, &driver.Task{})
		d.On("UpdateTask", mock.Anything, mock.Anything).Return(driver.InspectPlan{}, nil).Once()
		err := ctrl.drivers.Add(taskName, d)
		require.NoError(t, err)

		// add to state
		ctrl.state.SetTask(config.TaskConfig{
			Name:    &taskName,
			Enabled: config.Bool(false),
		})

		updateConf := config.TaskConfig{
			Name:    &taskName,
			Enabled: config.Bool(true),
		}

		changed, plan, _, err := ctrl.TaskUpdate(ctx, updateConf, driver.RunOptionNow)

		require.NoError(t, err)
		assert.Equal(t, "", plan, "run now does not return plan info")
		assert.False(t, changed, "run now does not return plan info")

		events := ctrl.state.GetTaskEvents(taskName)
		assert.Len(t, events, 1)

		// Confirm task became enabled in state
		stateTask, exists := ctrl.state.GetTask(taskName)
		require.True(t, exists)
		assert.True(t, *stateTask.Enabled)
	})

	t.Run("task-no-option", func(t *testing.T) {
		taskName := "task_d"

		// add a driver
		d := new(mocksD.Driver)
		mockDriver(ctx, d, &driver.Task{})
		d.On("UpdateTask", mock.Anything, mock.Anything).Return(driver.InspectPlan{}, nil).Once()
		err := ctrl.drivers.Add(taskName, d)
		require.NoError(t, err)

		// add to state
		ctrl.state.SetTask(config.TaskConfig{
			Name:    &taskName,
			Enabled: config.Bool(false),
		})

		updateConf := config.TaskConfig{
			Name:    &taskName,
			Enabled: config.Bool(true),
		}

		changed, plan, _, err := ctrl.TaskUpdate(ctx, updateConf, "")
		require.NoError(t, err)
		assert.Equal(t, "", plan, "no option does not return plan info")
		assert.False(t, changed, "no option does not return plan info")

		events := ctrl.state.GetTaskEvents(taskName)
		assert.Len(t, events, 0)

		// Confirm task became enabled in state
		stateTask, exists := ctrl.state.GetTask(taskName)
		require.True(t, exists)
		assert.True(t, *stateTask.Enabled)
	})
}

func TestServer_addTask(t *testing.T) {
	t.Parallel()

	ctrl := ReadWrite{
		deleteCh:        make(chan string, 1),
		scheduleStartCh: make(chan driver.Driver, 1),
		baseController: &baseController{
			logger: logging.NewNullLogger(),
		},
	}

	t.Run("success", func(t *testing.T) {
		// Set up driver's task object
		driverTask, err := driver.NewTask(driver.TaskConfig{
			Name:      "task_a",
			Condition: &config.ScheduleConditionConfig{},
		})
		require.NoError(t, err)

		// Mock driver
		d := new(mocksD.Driver)
		d.On("SetBufferPeriod").Return().Once()
		d.On("Task").Return(driverTask)
		d.On("TemplateIDs").Return(nil).Once()
		ctrl.drivers = driver.NewDrivers()

		// Mock state
		s := new(mocksS.Store)
		s.On("SetTask", mock.Anything).Return().Once()
		ctrl.state = s

		// Test addTask
		ctx := context.Background()
		taskConf, err := ctrl.addTask(ctx, d)
		require.NoError(t, err)

		// Basic check of returned conf
		assert.Equal(t, "task_a", *taskConf.Name)

		// Confirm driver added to drivers list
		assert.Equal(t, ctrl.drivers.Len(), 1)

		// Confirm state's Set called
		s.AssertExpectations(t)
		d.AssertExpectations(t)

		// Confirm received from scheduleStartCh
		select {
		case <-ctrl.scheduleStartCh:
			break
		case <-time.After(time.Second * 5):
			t.Fatal("did not receive from scheduleStartCh as expected")
		}
	})

	t.Run("error adding driver", func(t *testing.T) {
		taskName := "task_a"

		// Set up driver's task object
		driverTask, err := driver.NewTask(driver.TaskConfig{
			Name: taskName,
		})
		require.NoError(t, err)

		// Mock driver
		d := new(mocksD.Driver)
		d.On("SetBufferPeriod").Return().Once()
		d.On("Task").Return(driverTask)

		// Create an error by already adding the task to the drivers list
		ctrl.drivers.Add(taskName, d)

		// Test addTask
		ctx := context.Background()
		_, err = ctrl.addTask(ctx, d)
		require.Error(t, err)
		d.AssertExpectations(t)

		// Confirm that this second driver was not added to drivers list
		assert.Equal(t, 1, ctrl.drivers.Len())

		// Confirm received from delete channel for cleanup
		select {
		case <-ctrl.scheduleStartCh:
			t.Fatal("should not have received from scheduleStartCh")
		case <-ctrl.deleteCh:
			break
		case <-time.After(time.Second * 5):
			t.Fatal("did not receive from deleteCh as expected")
		}
	})
}

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

func TestReadWrite_CheckApply_Store(t *testing.T) {
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
	rw := &ReadWrite{}

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
			w := new(mocksTmpl.Watcher)
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

func TestReadWrite_deleteTask(t *testing.T) {
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
			ctrl := ReadWrite{
				baseController: &baseController{
					logger: logging.NewNullLogger(),
					state:  state.NewInMemoryStore(nil),
				},
			}
			ctrl.baseController.drivers = drivers
			ctrl.state.AddTaskEvent(event.Event{TaskName: "success"})

			err := ctrl.deleteTask(ctx, tc.name)

			assert.NoError(t, err)
			_, exists := drivers.Get(tc.name)
			assert.False(t, exists, "driver should no longer exist")

			events := ctrl.state.GetTaskEvents(tc.name)
			assert.Empty(t, events, "task events should no longer exist")

			_, exists = ctrl.state.GetTask(tc.name)
			assert.False(t, exists, "task should no longer exist in state")
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
		ctrl := newTestController()
		ctrl.drivers.Add(taskName, scheduledDriver)
		stopCh := make(chan struct{}, 1)
		ctrl.scheduleStopChs[taskName] = stopCh

		// Delete task
		err := ctrl.deleteTask(ctx, taskName)
		assert.NoError(t, err)

		// Verify the stop channel received message
		select {
		case <-time.After(1 * time.Second):
			t.Fatal("scheduled task was not notified to stop")
		case <-stopCh:
			break // expected case
		}
		_, ok := ctrl.scheduleStopChs[taskName]
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
		ctrl := ReadWrite{
			baseController: &baseController{
				logger: logging.NewNullLogger(),
				state:  state.NewInMemoryStore(nil),
			},
		}
		ctrl.baseController.drivers = drivers
		ctrl.state.AddTaskEvent(event.Event{TaskName: taskName})

		// Attempt to delete the active task
		ch := make(chan error)
		go func() {
			err := ctrl.deleteTask(ctx, taskName)
			ch <- err
		}()

		// Check that the task is not deleted while active
		time.Sleep(500 * time.Millisecond)
		_, exists := drivers.Get(taskName)
		assert.True(t, exists, "task deleted when active")
		events := ctrl.state.GetTaskEvents(taskName)
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
		assert.False(t, exists, "driver should no longer exist")

		events = ctrl.state.GetTaskEvents(taskName)
		assert.Empty(t, events, "task events should no longer exist")

		_, exists = ctrl.state.GetTask(taskName)
		assert.False(t, exists, "task should no longer exist in state")
	})

}
func TestReadWrite_waitForTaskInactive(t *testing.T) {
	ctx := context.Background()
	t.Run("active_task", func(t *testing.T) {
		// Set up drivers with active task
		ctrl := newTestController()
		taskName := "inactive_task"
		mockD := new(mocksD.Driver)
		mockD.On("TemplateIDs").Return(nil)
		ctrl.drivers.Add(taskName, mockD)
		ctrl.drivers.SetActive(taskName)

		// Wait for task to become inactive
		ch := make(chan error)
		go func() {
			err := ctrl.waitForTaskInactive(ctx, taskName)
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
		ctrl.drivers.SetInactive(taskName)
		select {
		case err := <-ch:
			assert.NoError(t, err)
		case <-time.After(1 * time.Second):
			t.Fatal("wait should have completed because task is inactive")
		}
	})

	t.Run("inactive_task", func(t *testing.T) {
		// Set up drivers with inactive task
		ctrl := newTestController()
		taskName := "inactive_task"
		mockD := new(mocksD.Driver)
		mockD.On("TemplateIDs").Return(nil)
		ctrl.drivers.Add(taskName, mockD)
		ctrl.drivers.SetInactive(taskName)

		// Wait for task to be inactive, should return immediately
		err := ctrl.waitForTaskInactive(ctx, taskName)
		require.NoError(t, err)
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
