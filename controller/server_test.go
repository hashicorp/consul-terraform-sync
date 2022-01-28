package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	"github.com/hashicorp/consul-terraform-sync/logging"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	mocksD "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	mocksTmpl "github.com/hashicorp/consul-terraform-sync/mocks/templates"
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

func TestServer_TaskCreate(t *testing.T) {
	ctx := context.Background()
	ctrl := ReadWrite{
		baseController: &baseController{
			conf: &config.Config{
				BufferPeriod: config.DefaultBufferPeriodConfig(),
				WorkingDir:   config.String(config.DefaultWorkingDir),
			},
			drivers: driver.NewDrivers(),
			logger:  logging.NewNullLogger(),
			watcher: new(mocksTmpl.Watcher),
		},
		store: event.NewStore(),
	}
	ctrl.conf.Finalize()

	t.Run("success", func(t *testing.T) {
		taskConf := validTaskConf
		driverTask, err := driver.NewTask(driver.TaskConfig{
			Enabled:   true,
			Name:      *taskConf.Name,
			Source:    *taskConf.Module,
			Condition: taskConf.Condition,
			BufferPeriod: &driver.BufferPeriod{
				Min: *ctrl.conf.BufferPeriod.Min,
				Max: *ctrl.conf.BufferPeriod.Max,
			},
			WorkingDir: *ctrl.conf.WorkingDir,
		})
		require.NoError(t, err)

		mockD := new(mocksD.Driver)
		mockD.On("SetBufferPeriod").Return()
		mockDriver(ctx, mockD, driverTask)
		ctrl.newDriver = func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		actual, err := ctrl.TaskCreate(ctx, taskConf)
		assert.NoError(t, err)
		conf, err := configFromDriverTask(driverTask)
		assert.NoError(t, err)
		assert.Equal(t, conf, actual)

		_, ok := ctrl.drivers.Get("task")
		assert.True(t, ok, "task should have a driver")

		events := ctrl.store.Read("task")
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
		ctrl.drivers = driver.NewDrivers()
		ctrl.newDriver = func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		_, err := ctrl.TaskCreate(ctx, validTaskConf)
		assert.Error(t, err)

		_, ok := ctrl.drivers.Get("task")
		assert.False(t, ok, "errored task should not have a driver added")

		events := ctrl.store.Read("task")
		assert.Len(t, events, 0, "no events stored on creation")
	})
}

func TestServer_TaskCreateAndRun(t *testing.T) {
	// TaskCreateAndRun is similar to TaskCreate but with added run behavior.
	// This tests what hasn't been testeed in TestServer_TaskCreate
	ctx := context.Background()

	ctrl := ReadWrite{
		baseController: &baseController{
			conf: &config.Config{
				BufferPeriod: config.DefaultBufferPeriodConfig(),
				WorkingDir:   config.String(config.DefaultWorkingDir),
			},
			logger:  logging.NewNullLogger(),
			watcher: new(mocksTmpl.Watcher),
		},
	}

	t.Run("success", func(t *testing.T) {
		mockD := new(mocksD.Driver)
		mockD.On("SetBufferPeriod").Return()
		task, err := driver.NewTask(driver.TaskConfig{
			Enabled: true,
			Name:    "task",
		})
		require.NoError(t, err)
		mockDriver(ctx, mockD, task)
		ctrl.store = event.NewStore()
		ctrl.drivers = driver.NewDrivers()
		ctrl.newDriver = func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		_, err = ctrl.TaskCreateAndRun(ctx, validTaskConf)
		assert.NoError(t, err)

		_, ok := ctrl.drivers.Get("task")
		assert.True(t, ok)

		events := ctrl.store.Read("task")
		assert.Len(t, events, 1)
		require.Len(t, events["task"], 1)
		assert.Nil(t, events["task"][0].EventError, "unexpected error event")
	})

	t.Run("disabled task", func(t *testing.T) {
		mockD := new(mocksD.Driver)
		mockD.On("SetBufferPeriod").Return()
		task, err := driver.NewTask(driver.TaskConfig{
			Enabled: false,
			Name:    "task",
		})
		require.NoError(t, err)
		mockDriver(ctx, mockD, task)
		ctrl.store = event.NewStore()
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

		events := ctrl.store.Read("task")
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
			On("RenderTemplate", mock.Anything).Return(true, nil).
			On("ApplyTask", ctx).Return(fmt.Errorf("apply err"))
		ctrl.store = event.NewStore()
		ctrl.drivers = driver.NewDrivers()
		ctrl.newDriver = func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			return mockD, nil
		}

		_, err = ctrl.TaskCreateAndRun(ctx, validTaskConf)
		assert.Error(t, err)

		_, ok := ctrl.drivers.Get("task")
		assert.False(t, ok, "driver is only added if the run is successful")

		events := ctrl.store.Read("task")
		assert.Len(t, events, 0, "event is only stored on successful creation and run")
	})
}

func TestServer_TaskDelete(t *testing.T) {
	ctx := context.Background()
	mockD := new(mocksD.Driver)
	ctrl := ReadWrite{
		baseController: &baseController{
			logger: logging.NewNullLogger(),
		},
		store: event.NewStore(),
	}

	testCases := []struct {
		name   string
		setup  func(*driver.Drivers)
		errMsg string
	}{
		{
			"success",
			func(d *driver.Drivers) {
				mockD.On("TemplateIDs").Return(nil)
				d.Add("success", mockD)
				mockD.On("DestroyTask", ctx).Return()
			},
			"",
		}, {
			"does_not_exist",
			func(*driver.Drivers) {},
			"",
		}, {
			"active",
			func(d *driver.Drivers) {
				mockD.On("TemplateIDs").Return(nil)
				d.Add("active", mockD)
				d.SetActive("active")
			},
			"running and cannot be deleted",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			drivers := driver.NewDrivers()

			tc.setup(drivers)
			ctrl.baseController.drivers = drivers

			err := ctrl.TaskDelete(ctx, tc.name)

			if tc.errMsg == "" {
				assert.NoError(t, err)
				_, exists := drivers.Get(tc.name)
				assert.False(t, exists, "task should no longer exist")
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errMsg)
			_, exists := drivers.Get(tc.name)
			assert.True(t, exists, "unexpected delete")
		})
	}
}

func TestServer_TaskUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := ReadWrite{
		baseController: &baseController{
			conf:    &config.Config{},
			drivers: driver.NewDrivers(),
			logger:  logging.NewNullLogger(),
		},
		store: event.NewStore(),
	}
	ctrl.conf.Finalize()

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
		taskConf.Finalize(ctrl.conf.BufferPeriod, *ctrl.conf.WorkingDir)
		task, err := newDriverTask(ctrl.conf, &taskConf, nil)
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
		events := ctrl.store.Read(taskName)
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
		// add a driver
		d := new(mocksD.Driver)
		mockDriver(ctx, d, &driver.Task{})
		d.On("UpdateTask", mock.Anything, mock.Anything).Return(expectedPlan, nil).Once()
		err := ctrl.drivers.Add("task_b", d)
		require.NoError(t, err)

		updateConf := config.TaskConfig{
			Name:    config.String("task_b"),
			Enabled: config.Bool(true),
		}

		changed, plan, _, err := ctrl.TaskUpdate(ctx, updateConf, driver.RunOptionInspect)

		require.NoError(t, err)
		assert.Equal(t, expectedPlan.Plan, plan)
		assert.Equal(t, expectedPlan.ChangesPresent, changed)

		// No events since the task did not run
		events := ctrl.store.Read("task_b")
		assert.Empty(t, events)
	})

	t.Run("task-run-now", func(t *testing.T) {
		taskName := "task_c"

		// add a driver
		d := new(mocksD.Driver)
		mockDriver(ctx, d, &driver.Task{})
		d.On("UpdateTask", mock.Anything, mock.Anything).Return(driver.InspectPlan{}, nil).Once()
		err := ctrl.drivers.Add(taskName, d)
		require.NoError(t, err)

		updateConf := config.TaskConfig{
			Name:    &taskName,
			Enabled: config.Bool(true),
		}

		changed, plan, _, err := ctrl.TaskUpdate(ctx, updateConf, driver.RunOptionNow)

		require.NoError(t, err)
		assert.Equal(t, "", plan, "run now does not return plan info")
		assert.False(t, changed, "run now does not return plan info")

		events := ctrl.store.Read(taskName)
		assert.Len(t, events, 1)
	})
}

// mockDriver sets up a mock driver with the happy path for all methods
func mockDriver(ctx context.Context, d *mocks.Driver, task *driver.Task) {
	d.On("Task").Return(task).
		On("InitTask", ctx).Return(nil).
		On("TemplateIDs").Return(nil).
		On("RenderTemplate", mock.Anything).Return(true, nil).
		On("ApplyTask", ctx).Return(nil)
}
