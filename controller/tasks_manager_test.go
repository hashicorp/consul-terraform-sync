package controller

import (
	"context"
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

// mockDriver sets up a mock driver with the happy path for all methods
func mockDriver(ctx context.Context, d *mocksD.Driver, task *driver.Task) {
	d.On("Task").Return(task).
		On("InitTask", ctx).Return(nil).
		On("TemplateIDs").Return(nil).
		On("RenderTemplate", mock.Anything).Return(true, nil).
		On("ApplyTask", ctx).Return(nil)
}
