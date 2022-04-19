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
	mocksS "github.com/hashicorp/consul-terraform-sync/mocks/store"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/consul-terraform-sync/state"
	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_TasksManager_runDynamicTask(t *testing.T) {
	t.Run("simple-success", func(t *testing.T) {
		tm := newTestTasksManager()

		ctx := context.Background()
		d := new(mocksD.Driver)
		mockDriver(ctx, d, enabledTestTask(t, "task"))
		tm.drivers.Add("task", d)

		err := tm.runDynamicTask(ctx, d)
		assert.NoError(t, err)
	})

	t.Run("apply-error", func(t *testing.T) {
		tm := newTestTasksManager()

		testErr := fmt.Errorf("could not apply: %s", "test")
		d := new(mocksD.Driver)
		d.On("Task").Return(enabledTestTask(t, "task"))
		d.On("TemplateIDs").Return(nil)
		d.On("InitWork", mock.Anything).Return(nil)
		d.On("RenderTemplate", mock.Anything).Return(true, nil)
		d.On("ApplyTask", mock.Anything).Return(testErr)
		tm.drivers.Add("task", d)

		err := tm.runDynamicTask(context.Background(), d)
		assert.Contains(t, err.Error(), testErr.Error())
	})

	t.Run("skip-scheduled-tasks", func(t *testing.T) {
		tm := newTestTasksManager()

		taskName := "scheduled_task"
		d := new(mocksD.Driver)
		d.On("Task").Return(scheduledTestTask(t, taskName))
		d.On("TemplateIDs").Return(nil)
		// no other methods should be called (or mocked)
		tm.drivers.Add(taskName, d)

		err := tm.runDynamicTask(context.Background(), d)
		assert.NoError(t, err)
	})

	t.Run("active-task", func(t *testing.T) {
		tm := newTestTasksManager()
		tm.EnableTestMode()

		ctx := context.Background()
		d := new(mocksD.Driver)
		taskName := "task"
		mockDriver(ctx, d, enabledTestTask(t, taskName))
		drivers := tm.drivers
		drivers.Add(taskName, d)
		drivers.SetActive(taskName)

		// Attempt to run the active task
		ch := make(chan error)
		go func() {
			err := tm.runDynamicTask(ctx, d)
			ch <- err
		}()

		// Check that the task did not run while active
		select {
		case <-tm.taskNotify:
			t.Fatal("task ran even though active")
		case <-time.After(250 * time.Millisecond):
			break
		}

		// Set task to inactive, wait for run to happen
		drivers.SetInactive(taskName)
		select {
		case <-time.After(250 * time.Millisecond):
			t.Fatal("task did not run after it became inactive")
		case <-tm.taskNotify:
			break
		}
	})

}

func Test_TasksManager_runScheduledTask(t *testing.T) {
	t.Run("happy-path", func(t *testing.T) {
		tm := newTestTasksManager()

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
		tm := newTestTasksManager()

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
		tm := newTestTasksManager()

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
		tm := newTestTasksManager()

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

func Test_TasksManager_Run_context_cancel(t *testing.T) {
	w := new(mocks.Watcher)
	w.On("Watch", mock.Anything, mock.Anything).Return(nil).
		On("Size").Return(5).
		On("Stop").Return()

	tm := newTestTasksManager()
	tm.watcher = w

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

func Test_TasksManager_Run_ActiveTask(t *testing.T) {
	// Set up tm with two tasks
	tm := newTestTasksManager()
	tm.watcherCh = make(chan string, 5)

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

	// Set up watcher for tm
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

func Test_TasksManager_Run_ScheduledTasks(t *testing.T) {
	t.Run("startup_task", func(t *testing.T) {
		tm := newTestTasksManager()
		tm.watcherCh = make(chan string, 5)
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

		// Set up watcher for tm
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
		tm := newTestTasksManager()
		tm.watcherCh = make(chan string, 5)
		tm.scheduleStartCh = make(chan driver.Driver, 1)
		tm.EnableTestMode()

		// Set up watcher for tm
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

func Test_TasksManager_EnableTestMode(t *testing.T) {
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

	// Test EnableTestMode
	channel := tm.EnableTestMode()
	assert.Equal(t, 2, cap(channel))
	s.AssertExpectations(t)
}

func Test_TasksManager_RunInspect(t *testing.T) {
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

			tm := newTestTasksManager()
			tm.watcher = w

			d := new(mocksD.Driver)
			d.On("Task").Return(enabledTestTask(t, "task"))
			d.On("TemplateIDs").Return(nil)
			d.On("RenderTemplate", mock.Anything).
				Return(true, tc.renderTmplErr)
			d.On("InspectTask", mock.Anything).
				Return(driver.InspectPlan{}, tc.inspectTaskErr)
			err := tm.drivers.Add("task", d)
			require.NoError(t, err)

			err = tm.RunInspect(context.Background())
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

func Test_TasksManager_RunInspect_context_cancel(t *testing.T) {
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

	tm := newTestTasksManager()
	tm.watcher = w
	tm.drivers = drivers
	tm.baseController.resolver = r

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		err := tm.RunInspect(ctx)
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

func Test_TasksManager_WatchDep_context_cancel(t *testing.T) {
	t.Parallel()

	t.Run("cancel exits successfully", func(t *testing.T) {
		tm := newTestTasksManager()

		// Mock watcher
		w := new(mocks.Watcher)
		waitErrCh := make(chan error, 1)
		var waitErrChRc <-chan error = waitErrCh
		waitErrCh <- nil
		w.On("WaitCh", mock.Anything).Return(waitErrChRc)
		w.On("Size", mock.Anything).Return(1)
		tm.watcher = w

		errCh := make(chan error)
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			if err := tm.WatchDep(ctx); err != nil {
				errCh <- err
			}
		}()
		cancel()

		select {
		case err := <-errCh:
			// Confirm that exit is due to context cancel
			assert.Equal(t, err, context.Canceled)
		case <-time.After(time.Second * 5):
			t.Fatal("WatchDep did not exit properly from cancelling context")
		}

		w.AssertExpectations(t)
	})

	t.Run("error exits successfully", func(t *testing.T) {
		tm := newTestTasksManager()

		// Mock watcher
		w := new(mocks.Watcher)
		waitErrCh := make(chan error, 1)
		var waitErrChRc <-chan error = waitErrCh
		waitErrCh <- errors.New("error!")
		w.On("WaitCh", mock.Anything).Return(waitErrChRc)
		tm.watcher = w

		errCh := make(chan error)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			if err := tm.WatchDep(ctx); err != nil {
				errCh <- err
			}
		}()

		select {
		case err := <-errCh:
			// Confirm error was received and successfully exit
			assert.Contains(t, err.Error(), "error!")
		case <-time.After(time.Second * 5):
			t.Fatal("WatchDep did not error and exit properly")
		}

		w.AssertExpectations(t)
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
		TerraformProviders: &config.TerraformProviderConfigs{
			{"X": map[string]interface{}{}},
			{
				handler.TerraformProviderFake: map[string]interface{}{
					"name": "fake-provider",
				},
			},
		},
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

func newTestTasksManager() TasksManager {
	return TasksManager{
		logger: logging.NewNullLogger(),
		baseController: &baseController{
			logger: logging.NewNullLogger(),
		},
		drivers:         driver.NewDrivers(),
		state:           state.NewInMemoryStore(nil),
		scheduleStopChs: make(map[string](chan struct{})),
	}
}
