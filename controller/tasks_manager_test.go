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
	mocksS "github.com/hashicorp/consul-terraform-sync/mocks/state"
	mocksTmpl "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/consul-terraform-sync/state"
	"github.com/hashicorp/consul-terraform-sync/state/event"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	validTaskName = "task"
	validTaskConf = config.TaskConfig{
		Enabled: config.Bool(true),
		Name:    config.String(validTaskName),
		Module:  config.String("module"),
		Condition: &config.CatalogServicesConditionConfig{
			CatalogServicesMonitorConfig: config.CatalogServicesMonitorConfig{Regexp: config.String("regex")},
		},
	}
)

func Test_TasksManager_Task(t *testing.T) {
	ctx := context.Background()
	tm := newTestTasksManager()

	t.Run("success", func(t *testing.T) {
		taskConf := validTaskConf
		taskConf.Finalize(config.DefaultBufferPeriodConfig(), "path")

		s := new(mocksS.Store)
		s.On("GetTask", mock.Anything).Return(taskConf, true)
		tm.state = s

		actualConf, err := tm.Task(ctx, *taskConf.Name)
		require.NoError(t, err)

		assert.Equal(t, taskConf, actualConf)

		s.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		s := new(mocksS.Store)
		s.On("GetTask", mock.Anything, mock.Anything).Return(config.TaskConfig{}, false)
		tm.state = s

		_, err := tm.Task(ctx, "non-existent-task")
		assert.Error(t, err)

		s.AssertExpectations(t)
	})
}

func Test_TasksManager_Tasks(t *testing.T) {
	ctx := context.Background()
	tm := newTestTasksManager()

	t.Run("success", func(t *testing.T) {
		taskConfs := config.TaskConfigs{
			{Name: config.String("task_a")},
			{Name: config.String("task_b")},
		}
		taskConfs.Finalize(config.DefaultBufferPeriodConfig(), config.DefaultWorkingDir)

		s := new(mocksS.Store)
		s.On("GetAllTasks", mock.Anything, mock.Anything).Return(taskConfs)
		tm.state = s

		actualConfs := tm.Tasks(ctx)

		assert.Len(t, actualConfs, taskConfs.Len())
		for ix, expectedConf := range taskConfs {
			assert.Equal(t, expectedConf, actualConfs[ix])
		}

		s.AssertExpectations(t)
	})

	t.Run("empty return", func(t *testing.T) {
		s := new(mocksS.Store)
		s.On("GetAllTasks", mock.Anything, mock.Anything).Return(config.TaskConfigs{})
		tm.state = s

		actualConfs := tm.Tasks(ctx)
		assert.Equal(t, config.TaskConfigs{}, actualConfs)

		s.AssertExpectations(t)
	})
}

func Test_TasksManager_TaskCreate(t *testing.T) {
	ctx := context.Background()
	conf := &config.Config{
		BufferPeriod: config.DefaultBufferPeriodConfig(),
		WorkingDir:   config.String(config.DefaultWorkingDir),
	}
	conf.Finalize()
	tm := newTestTasksManager()
	tm.factory.watcher = new(mocksTmpl.Watcher)
	tm.state = state.NewInMemoryStore(conf)

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
		tm.factory.newDriver = func(context.Context, *config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		actual, err := tm.TaskCreate(ctx, taskConf)
		assert.NoError(t, err)
		conf, err := configFromDriverTask(driverTask)
		assert.NoError(t, err)
		assert.Equal(t, conf, actual)

		// Basic check that task was added
		_, ok := tm.drivers.Get(validTaskName)
		assert.True(t, ok, "task should have a driver")

		// Basic check that task was not run
		events := tm.state.GetTaskEvents(validTaskName)
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
		tm.factory.newDriver = func(context.Context, *config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		_, err := tm.TaskCreate(ctx, validTaskConf)
		assert.Error(t, err)

		_, ok := tm.drivers.Get(validTaskName)
		assert.False(t, ok, "errored task should not have a driver added")

		events := tm.state.GetTaskEvents(validTaskName)
		assert.Len(t, events, 0, "no events stored on creation")
	})
}

func Test_TasksManager_TaskCreateAndRun(t *testing.T) {
	// TaskCreateAndRun is similar to TaskCreate but with added run behavior.
	// This tests what hasn't been testeed in Test_TasksManager_TaskCreate
	ctx := context.Background()

	tm := newTestTasksManager()
	tm.factory.watcher = new(mocksTmpl.Watcher)

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
			Name:    validTaskName,
		})
		require.NoError(t, err)
		mockDriver(ctx, mockD, task)
		tm.state = state.NewInMemoryStore(conf)
		tm.drivers = driver.NewDrivers()
		tm.factory.newDriver = func(context.Context, *config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		_, err = tm.TaskCreateAndRun(ctx, validTaskConf)
		assert.NoError(t, err)

		// Basic check that task was added
		_, ok := tm.drivers.Get(validTaskName)
		assert.True(t, ok)

		// Basic check that task was ran
		events := tm.state.GetTaskEvents(validTaskName)
		assert.Len(t, events, 1)
		require.Len(t, events[validTaskName], 1)
		assert.Nil(t, events[validTaskName][0].EventError, "unexpected error event")
	})

	t.Run("disabled task", func(t *testing.T) {
		mockD := new(mocksD.Driver)
		mockD.On("SetBufferPeriod").Return()
		mockD.On("OverrideNotifier").Return()
		task, err := driver.NewTask(driver.TaskConfig{
			Enabled: false,
			Name:    validTaskName,
		})
		require.NoError(t, err)
		mockDriver(ctx, mockD, task)
		tm.state = state.NewInMemoryStore(conf)
		tm.drivers = driver.NewDrivers()
		tm.factory.newDriver = func(context.Context, *config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		taskConf := validTaskConf.Copy()
		taskConf.Enabled = config.Bool(false)

		_, err = tm.TaskCreateAndRun(ctx, *taskConf)
		assert.NoError(t, err)

		_, ok := tm.drivers.Get(validTaskName)
		assert.True(t, ok, "driver is created for task even if it's disabled")

		events := tm.state.GetTaskEvents(validTaskName)
		assert.Len(t, events, 0, "task is disabled, no run should occur")
	})

	t.Run("apply error", func(t *testing.T) {
		mockD := new(mocksD.Driver)
		task, err := driver.NewTask(driver.TaskConfig{
			Enabled: true,
			Name:    validTaskName,
		})
		require.NoError(t, err)
		mockD.On("Task").Return(task).
			On("InitTask", ctx).Return(nil).
			On("OverrideNotifier").Return().
			On("RenderTemplate", mock.Anything).Return(true, nil).
			On("ApplyTask", ctx).Return(fmt.Errorf("apply err"))
		tm.state = state.NewInMemoryStore(conf)
		tm.drivers = driver.NewDrivers()
		tm.factory.newDriver = func(context.Context, *config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		_, err = tm.TaskCreateAndRun(ctx, validTaskConf)
		assert.Error(t, err)

		_, ok := tm.drivers.Get(validTaskName)
		assert.False(t, ok, "driver is only added if the run is successful")

		events := tm.state.GetTaskEvents(validTaskName)
		assert.Len(t, events, 0, "event is only stored on successful creation and run")
	})
}

func Test_TasksManager_TaskDelete(t *testing.T) {
	ctx := context.Background()
	tm := newTestTasksManager()
	deletedCh := tm.EnableTaskDeletedNotify()

	t.Run("happy path", func(t *testing.T) {
		drivers := driver.NewDrivers()
		taskName := "delete_task"

		mockD := new(mocksD.Driver)
		mockD.On("TemplateIDs").Return(nil)
		mockD.On("Task").Return(enabledTestTask(t, "delete_task"))
		mockD.On("DestroyTask", ctx).Return()
		drivers.Add(taskName, mockD)

		tm.drivers = drivers

		go tm.TaskDelete(ctx, taskName)
		select {
		case n := <-deletedCh:
			assert.Equal(t, taskName, n)
		case <-time.After(1 * time.Second):
			t.Log("delete channel did not receive message")
			t.Fail()
		}
		assert.Equal(t, 0, drivers.Len())
	})

	t.Run("already marked for deletion", func(t *testing.T) {
		drivers := driver.NewDrivers()
		taskName := "delete_task"
		tm.drivers = drivers
		tm.drivers.MarkForDeletion(taskName)
		err := tm.TaskDelete(ctx, taskName)
		assert.NoError(t, err)
		assert.True(t, tm.drivers.IsMarkedForDeletion(taskName))
	})
}

func Test_TasksManager_TaskUpdate(t *testing.T) {
	t.Parallel()

	conf := &config.Config{}
	conf.Finalize()
	ctx := context.Background()
	tm := newTestTasksManager()
	tm.state = state.NewInMemoryStore(conf)

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

		taskName := "task_b"

		// add a driver
		d := new(mocksD.Driver)
		mockDriver(ctx, d, &driver.Task{})
		d.On("UpdateTask", mock.Anything, mock.Anything).Return(expectedPlan, nil).Once()
		err := tm.drivers.Add(taskName, d)
		require.NoError(t, err)

		// add to state
		err = tm.state.SetTask(config.TaskConfig{
			Name:    &taskName,
			Enabled: config.Bool(false),
		})
		require.NoError(t, err, "unexpected error while setting task state")

		updateConf := config.TaskConfig{
			Name:    config.String(taskName),
			Enabled: config.Bool(true),
		}

		changed, plan, _, err := tm.TaskUpdate(ctx, updateConf, driver.RunOptionInspect)

		require.NoError(t, err)
		assert.Equal(t, expectedPlan.Plan, plan)
		assert.Equal(t, expectedPlan.ChangesPresent, changed)

		// No events since the task did not run
		events := tm.state.GetTaskEvents(taskName)
		assert.Empty(t, events)

		// Confirm task stayed disabled in state
		stateTask, exists := tm.state.GetTask(taskName)
		require.True(t, exists)
		assert.False(t, *stateTask.Enabled)
	})

	t.Run("task-run-now", func(t *testing.T) {
		taskName := "task_c"

		// add a driver
		d := new(mocksD.Driver)
		mockDriver(ctx, d, &driver.Task{})
		d.On("UpdateTask", mock.Anything, mock.Anything).Return(driver.InspectPlan{}, nil).Once()
		err := tm.drivers.Add(taskName, d)
		require.NoError(t, err)

		// add to state
		err = tm.state.SetTask(config.TaskConfig{
			Name:    &taskName,
			Enabled: config.Bool(false),
		})
		require.NoError(t, err, "unexpected error while setting task state")

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

		// Confirm task became enabled in state
		stateTask, exists := tm.state.GetTask(taskName)
		require.True(t, exists)
		assert.True(t, *stateTask.Enabled)
	})

	t.Run("task-no-option", func(t *testing.T) {
		taskName := "task_d"

		// add a driver
		d := new(mocksD.Driver)
		mockDriver(ctx, d, &driver.Task{})
		d.On("UpdateTask", mock.Anything, mock.Anything).Return(driver.InspectPlan{}, nil).Once()
		err := tm.drivers.Add(taskName, d)
		require.NoError(t, err)

		// add to state
		err = tm.state.SetTask(config.TaskConfig{
			Name:    &taskName,
			Enabled: config.Bool(false),
		})
		require.NoError(t, err, "unexpected error while setting task state")

		updateConf := config.TaskConfig{
			Name:    &taskName,
			Enabled: config.Bool(true),
		}

		changed, plan, _, err := tm.TaskUpdate(ctx, updateConf, "")
		require.NoError(t, err)
		assert.Equal(t, "", plan, "no option does not return plan info")
		assert.False(t, changed, "no option does not return plan info")

		events := tm.state.GetTaskEvents(taskName)
		assert.Len(t, events, 0)

		// Confirm task became enabled in state
		stateTask, exists := tm.state.GetTask(taskName)
		require.True(t, exists)
		assert.True(t, *stateTask.Enabled)
	})
}

func Test_TasksManager_addTask(t *testing.T) {
	t.Parallel()

	tm := newTestTasksManager()
	tm.createdScheduleCh = make(chan string, 1)

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
		tm.drivers = driver.NewDrivers()

		// Mock state
		s := new(mocksS.Store)
		s.On("SetTask", mock.Anything).Return(nil).Once()
		tm.state = s

		// Test addTask
		ctx := context.Background()
		taskConf, err := tm.addTask(ctx, d)
		require.NoError(t, err)

		// Basic check of returned conf
		assert.Equal(t, "task_a", *taskConf.Name)

		// Confirm driver added to drivers list
		assert.Equal(t, tm.drivers.Len(), 1)

		// Confirm state's Set called
		s.AssertExpectations(t)
		d.AssertExpectations(t)

		// Confirm received from createdScheduleCh
		select {
		case <-tm.createdScheduleCh:
			break
		case <-time.After(time.Second * 5):
			t.Fatal("did not receive from createdScheduleCh as expected")
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
		d.On("DestroyTask", mock.Anything).Return().Once()

		// Create an error by already adding the task to the drivers list
		tm.drivers.Add(taskName, d)

		// Test addTask
		ctx := context.Background()
		_, err = tm.addTask(ctx, d)
		require.Error(t, err)
		d.AssertExpectations(t)

		// Confirm that this second driver was not added to drivers list
		assert.Equal(t, 1, tm.drivers.Len())
	})
}

func Test_TasksManager_TaskRunNow(t *testing.T) {
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

			tm := newTestTasksManager()
			tm.drivers = drivers

			ctx := context.Background()
			err := tm.TaskRunNow(ctx, tc.taskName)
			data := tm.state.GetTaskEvents(tc.taskName)
			events := data[tc.taskName]

			if !tc.addToStore {
				if tc.expectError {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tc.name)
				}
				assert.Equal(t, 0, len(events))
				return
			}

			require.Len(t, events, 1)
			e := events[0]
			assert.Equal(t, tc.taskName, e.TaskName)
			assert.False(t, e.StartTime.IsZero())
			assert.False(t, e.EndTime.IsZero())

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.name)
				assert.False(t, e.Success)
				require.NotNil(t, e.EventError)
				assert.Contains(t, e.EventError.Message, tc.name)
			} else {
				assert.NoError(t, err)
				assert.True(t, e.Success)
			}
		})
	}

	t.Run("unrendered-scheduled-tasks", func(t *testing.T) {
		// Test the behavior for the situation where a scheduled task's
		// template did not render

		tm := newTestTasksManager()

		d := new(mocksD.Driver)
		d.On("Task").Return(scheduledTestTask(t, schedTaskName))
		d.On("TemplateIDs").Return(nil)
		d.On("RenderTemplate", mock.Anything).Return(false, nil)
		tm.drivers.Add(schedTaskName, d)

		// Daemon-mode - confirm an event is stored
		ctx := context.Background()
		err := tm.TaskRunNow(ctx, schedTaskName)
		require.NoError(t, err)
		data := tm.state.GetTaskEvents(schedTaskName)
		events := data[schedTaskName]
		assert.Len(t, events, 1)
	})

	t.Run("marked-for-deletion", func(t *testing.T) {
		// Tests that drivers marked for deletion are not run

		// Confirms that other Run-type driver methods are not called
		d := new(mocksD.Driver)
		d.On("TemplateIDs").Return(nil)

		tm := newTestTasksManager()
		tm.drivers.Add(schedTaskName, d)

		tm.drivers.MarkForDeletion(schedTaskName)

		ctx := context.Background()
		err := tm.TaskRunNow(ctx, schedTaskName)
		assert.NoError(t, err)
		d.AssertExpectations(t)

		// Confirm no event stored
		data := tm.state.GetTaskEvents(schedTaskName)
		events := data[schedTaskName]
		assert.Empty(t, events)
	})

	t.Run("active-scheduled-tasks", func(t *testing.T) {
		// Tests that active scheduled task drivers do not run

		// Confirms that other Run-type driver methods are not called
		d := new(mocksD.Driver)
		d.On("TemplateIDs").Return(nil)
		d.On("Task").Return(scheduledTestTask(t, schedTaskName))

		tm := newTestTasksManager()
		tm.drivers.Add(schedTaskName, d)

		tm.drivers.SetActive(schedTaskName)

		ctx := context.Background()
		err := tm.TaskRunNow(ctx, schedTaskName)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is active")
		d.AssertExpectations(t)

		// Confirm no event stored
		data := tm.state.GetTaskEvents(schedTaskName)
		events := data[schedTaskName]
		assert.Empty(t, events)
	})

	t.Run("active-dynamic-task", func(t *testing.T) {
		// Tests that active dynamic task drivers will wait for inactive

		tm := newTestTasksManager()
		tm.EnableTaskRanNotify()
		err := tm.state.SetTask(validTaskConf)
		require.NoError(t, err, "unexpected error while setting task state")

		ctx := context.Background()
		d := new(mocksD.Driver)
		d.On("Task").Return(enabledTestTask(t, validTaskName)).
			On("TemplateIDs").Return(nil).
			On("RenderTemplate", mock.Anything).Return(true, nil).
			On("ApplyTask", ctx).Return(nil)
		drivers := tm.drivers
		drivers.Add(validTaskName, d)
		drivers.SetActive(validTaskName)

		// Attempt to run the active task
		ch := make(chan error)
		go func() {
			err := tm.TaskRunNow(ctx, validTaskName)
			ch <- err
		}()

		// Check that the task did not run while active
		select {
		case <-tm.ranTaskNotify:
			t.Fatal("task ran even though active")
		case <-time.After(250 * time.Millisecond):
			break
		}

		// Set task to inactive, wait for run to happen
		drivers.SetInactive(validTaskName)
		select {
		case <-time.After(250 * time.Millisecond):
			t.Fatal("task did not run after it became inactive")
		case <-tm.ranTaskNotify:
			break
		}
	})
}

func Test_TasksManager_TaskRunNow_Store(t *testing.T) {
	t.Run("mult-checkapply-store", func(t *testing.T) {
		d := new(mocksD.Driver)
		d.On("Task").Return(enabledTestTask(t, "task_a"))
		d.On("TemplateIDs").Return(nil)
		d.On("RenderTemplate", mock.Anything).Return(true, nil)
		d.On("ApplyTask", mock.Anything).Return(nil)

		disabledD := new(mocksD.Driver)
		disabledD.On("Task").Return(disabledTestTask(t, "task_b"))
		disabledD.On("TemplateIDs").Return(nil)

		tm := newTestTasksManager()

		tm.drivers.Add("task_a", d)
		tm.drivers.Add("task_b", disabledD)
		ctx := context.Background()

		tm.TaskRunNow(ctx, "task_a")
		tm.TaskRunNow(ctx, "task_b")
		tm.TaskRunNow(ctx, "task_a")
		tm.TaskRunNow(ctx, "task_a")
		tm.TaskRunNow(ctx, "task_a")
		tm.TaskRunNow(ctx, "task_b")

		taskStatuses := tm.state.GetTaskEvents("")

		assert.Equal(t, 4, len(taskStatuses["task_a"]))
		assert.Equal(t, 0, len(taskStatuses["task_b"]))
	})
}

func Test_ConditionMonitor_EnableTaskRanNotify(t *testing.T) {
	t.Parallel()

	// Mock state store
	taskConfs := config.TaskConfigs{
		{Name: config.String("task_a")},
		{Name: config.String("task_b")},
	}
	s := new(mocksS.Store)
	s.On("GetAllTasks", mock.Anything, mock.Anything).Return(taskConfs)

	// Set up tasks manager
	tm := newTestTasksManager()
	tm.state = s

	// Test EnableTaskRanNotify
	channel := tm.EnableTaskRanNotify()
	assert.Equal(t, 2, cap(channel))
	s.AssertExpectations(t)
}

func Test_TasksManager_TaskFailSilently(t *testing.T) {
	// TaskFailSilently is similar to TaskCreateAndRun but with modified error
	// handling. This tests what hasn't been tested in Test_TasksManager_TaskCreate
	ctx := context.Background()

	tm := newTestTasksManager()
	tm.factory.watcher = new(mocksTmpl.Watcher)

	conf := &config.Config{
		BufferPeriod: config.DefaultBufferPeriodConfig(),
		WorkingDir:   config.String(config.DefaultWorkingDir),
		Driver:       config.DefaultDriverConfig(),
	}

	t.Run("run error", func(t *testing.T) {
		mockD := new(mocksD.Driver)
		task, err := driver.NewTask(driver.TaskConfig{
			Enabled: true,
			Name:    validTaskName,
		})
		require.NoError(t, err)
		mockD.On("Task").Return(task).
			On("InitTask", ctx).Return(nil).
			On("OverrideNotifier").Return().
			On("RenderTemplate", mock.Anything).Return(true, nil).
			On("ApplyTask", ctx).Return(fmt.Errorf("apply err")).
			On("SetBufferPeriod").Return().Once().
			On("TemplateIDs").Return(nil).Once()

		tm.state = state.NewInMemoryStore(conf)
		tm.drivers = driver.NewDrivers()
		tm.factory.newDriver = func(context.Context, *config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		tm.TaskFailSilently(ctx, validTaskConf)

		_, ok := tm.drivers.Get(validTaskName)
		assert.True(t, ok, "driver is added even if run is unsuccessful")

		events := tm.state.GetTaskEvents(validTaskName)
		assert.Len(t, events, 1, "event is stored even on failed run")
	})
}

func Test_TasksManager_deleteTask(t *testing.T) {
	ctx := context.Background()
	mockD := new(mocksD.Driver)

	testCases := []struct {
		name         string
		setupDrivers func() *driver.Drivers
	}{
		{
			"success",
			func() *driver.Drivers {
				mockD.On("TemplateIDs").Return(nil)
				mockD.On("Task").Return(enabledTestTask(t, "success"))
				mockD.On("DestroyTask", ctx).Return()
				d := driver.NewDrivers()
				d.Add("success", mockD)
				return d
			},
		},
		{
			"does_not_exist",
			func() *driver.Drivers {
				return driver.NewDrivers()
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tm := newTestTasksManager()
			tm.drivers = tc.setupDrivers()

			tm.state.AddTaskEvent(event.Event{TaskName: "success"})

			err := tm.deleteTask(ctx, tc.name)

			assert.NoError(t, err)
			_, exists := tm.drivers.Get(tc.name)
			assert.False(t, exists, "driver should no longer exist")

			events := tm.state.GetTaskEvents(tc.name)
			assert.Empty(t, events, "task events should no longer exist")

			_, exists = tm.state.GetTask(tc.name)
			assert.False(t, exists, "task should no longer exist in state")
		})
	}

	t.Run("scheduled_task", func(t *testing.T) {
		// Tests that deleting a scheduled task sends a deleted notification

		// Setup tm and drivers
		scheduledDriver := new(mocksD.Driver)
		scheduledDriver.On("Task").Return(scheduledTestTask(t, schedTaskName))
		scheduledDriver.On("DestroyTask", ctx).Return()
		scheduledDriver.On("TemplateIDs").Return(nil)
		tm := newTestTasksManager()
		tm.drivers.Add(schedTaskName, scheduledDriver)
		tm.deletedScheduleCh = make(chan string, 1)

		// Delete task
		err := tm.deleteTask(ctx, schedTaskName)
		assert.NoError(t, err)

		// Verify the deleted schedule channel received message
		select {
		case <-time.After(1 * time.Second):
			t.Fatal("scheduled task was not notified to stop")
		case name := <-tm.WatchDeletedScheduleTask():
			assert.Equal(t, schedTaskName, name)
		}
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

		// Set up tm with drivers and store
		tm := newTestTasksManager()
		tm.drivers = drivers
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
		assert.False(t, exists, "driver should no longer exist")

		events = tm.state.GetTaskEvents(taskName)
		assert.Empty(t, events, "task events should no longer exist")

		_, exists = tm.state.GetTask(taskName)
		assert.False(t, exists, "task should no longer exist in state")
	})

}
func Test_TasksManager_waitForTaskInactive(t *testing.T) {
	ctx := context.Background()
	t.Run("active_task", func(t *testing.T) {
		// Set up drivers with active task
		tm := newTestTasksManager()
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
		tm := newTestTasksManager()
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

// mockDriver sets up a mock driver with the happy path for task create and
// update methods
func mockDriver(ctx context.Context, d *mocksD.Driver, task *driver.Task) {
	d.On("Task").Return(task).
		On("InitTask", ctx).Return(nil).
		On("TemplateIDs").Return(nil).
		On("RenderTemplate", mock.Anything).Return(true, nil).
		On("ApplyTask", ctx).Return(nil)
}

func newTestTasksManager() *TasksManager {
	return &TasksManager{
		logger: logging.NewNullLogger(),
		factory: &driverFactory{
			logger: logging.NewNullLogger(),
		},
		drivers: driver.NewDrivers(),
		state:   state.NewInMemoryStore(nil),
	}
}
