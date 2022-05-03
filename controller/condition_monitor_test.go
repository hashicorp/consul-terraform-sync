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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	schedTaskName = "scheduled_task"
	schedTaskConf = config.TaskConfig{
		Enabled: config.Bool(true),
		Name:    config.String(schedTaskName),
		Module:  config.String("module"),
		Condition: &config.ScheduleConditionConfig{
			Cron: config.String("*/3 * * * * * *"),
		},
	}
)

func Test_ConditionMonitor_runDynamicTask(t *testing.T) {
	t.Run("simple-success", func(t *testing.T) {
		tm := newTestTasksManager()
		tm.state.SetTask(validTaskConf)

		ctx := context.Background()
		d := new(mocksD.Driver)
		d.On("Task").Return(enabledTestTask(t, validTaskName)).
			On("TemplateIDs").Return(nil).
			On("RenderTemplate", mock.Anything).Return(true, nil).
			On("ApplyTask", ctx).Return(nil)
		tm.drivers.Add(validTaskName, d)

		cm := newTestConditionMonitor(tm)
		err := cm.runDynamicTask(ctx, validTaskName)
		assert.NoError(t, err)
		d.AssertExpectations(t)
	})

	t.Run("apply-error", func(t *testing.T) {
		tm := newTestTasksManager()
		tm.state.SetTask(validTaskConf)

		testErr := fmt.Errorf("could not apply: %s", "test")
		d := new(mocksD.Driver)
		d.On("Task").Return(enabledTestTask(t, validTaskName))
		d.On("TemplateIDs").Return(nil)
		d.On("RenderTemplate", mock.Anything).Return(true, nil)
		d.On("ApplyTask", mock.Anything).Return(testErr)
		tm.drivers.Add(validTaskName, d)

		cm := newTestConditionMonitor(tm)
		err := cm.runDynamicTask(context.Background(), validTaskName)
		assert.Contains(t, err.Error(), testErr.Error())
		d.AssertExpectations(t)
	})

	t.Run("skip-scheduled-tasks", func(t *testing.T) {
		tm := newTestTasksManager()
		tm.state.SetTask(schedTaskConf)

		d := new(mocksD.Driver)
		d.On("TemplateIDs").Return(nil)
		// no other methods should be called (or mocked)
		tm.drivers.Add(schedTaskName, d)

		cm := newTestConditionMonitor(tm)
		err := cm.runDynamicTask(context.Background(), schedTaskName)
		assert.Error(t, err)
		d.AssertExpectations(t)
	})

	t.Run("active-task", func(t *testing.T) {
		tm := newTestTasksManager()
		tm.EnableTestMode()
		tm.state.SetTask(validTaskConf)

		ctx := context.Background()
		d := new(mocksD.Driver)
		d.On("Task").Return(enabledTestTask(t, validTaskName)).
			On("TemplateIDs").Return(nil).
			On("RenderTemplate", mock.Anything).Return(true, nil).
			On("ApplyTask", ctx).Return(nil)
		drivers := tm.drivers
		drivers.Add(validTaskName, d)
		drivers.SetActive(validTaskName)

		cm := newTestConditionMonitor(tm)

		// Attempt to run the active task
		ch := make(chan error)
		go func() {
			err := cm.runDynamicTask(ctx, validTaskName)
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
		drivers.SetInactive(validTaskName)
		select {
		case <-time.After(250 * time.Millisecond):
			t.Fatal("task did not run after it became inactive")
		case <-tm.taskNotify:
			break
		}

		d.AssertExpectations(t)
	})

}

func Test_ConditionMonitor_runScheduledTask(t *testing.T) {
	t.Run("happy-path", func(t *testing.T) {
		tm := newTestTasksManager()
		tm.state.SetTask(schedTaskConf)

		d := new(mocksD.Driver)
		d.On("Task").Return(scheduledTestTask(t, schedTaskName)).Once()
		d.On("RenderTemplate", mock.Anything).Return(true, nil).Once()
		d.On("ApplyTask", mock.Anything).Return(nil).Once()
		d.On("TemplateIDs").Return(nil)
		tm.drivers.Add(schedTaskName, d)

		cm := newTestConditionMonitor(tm)

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error)
		stopCh := make(chan struct{}, 1)
		go func() {
			err := cm.runScheduledTask(ctx, schedTaskName, stopCh)
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
		tm.state.SetTask(validTaskConf)
		// No calls to driver are made

		cm := newTestConditionMonitor(tm)

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error)
		stopCh := make(chan struct{}, 1)
		go func() {
			err := cm.runScheduledTask(ctx, validTaskName, stopCh)
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
	})

	t.Run("stop-scheduled-task", func(t *testing.T) {
		// Tests that signaling to the stop channel stops the function
		tm := newTestTasksManager()
		tm.state.SetTask(schedTaskConf)

		d := new(mocksD.Driver)
		d.On("TemplateIDs").Return(nil)
		tm.drivers.Add(schedTaskName, d)

		cm := newTestConditionMonitor(tm)

		ctx := context.Background()
		errCh := make(chan error)
		stopCh := make(chan struct{}, 1)
		go func() {
			err := cm.runScheduledTask(ctx, schedTaskName, stopCh)
			errCh <- err
		}()
		stopCh <- struct{}{}

		select {
		case err := <-errCh:
			assert.NoError(t, err)
		case <-time.After(time.Second * 5):
			t.Fatal("runScheduledTask did not exit as expected")
		}

		d.AssertExpectations(t)
	})

	t.Run("deleted-scheduled-task", func(t *testing.T) {
		// Tests that a scheduled task stops if it no longer is in the state
		tm := newTestTasksManager()
		tm.state.SetTask(schedTaskConf)

		cm := newTestConditionMonitor(tm)

		ctx := context.Background()
		errCh := make(chan error)
		stopCh := make(chan struct{}, 1)
		cm.scheduleStopChs[schedTaskName] = stopCh
		done := make(chan bool)
		go func() {
			err := cm.runScheduledTask(ctx, schedTaskName, stopCh)
			if err != nil {
				errCh <- err
			}
			done <- true
		}()

		// Wait for scheduled task to start cadence, then remove task from state
		time.Sleep(1 * time.Second)
		tm.state.DeleteTask(schedTaskName)

		select {
		case <-errCh:
			t.Fatal("runScheduledTask did not exit properly when task is not in map of drivers")
		case <-done:
			// runScheduledTask exited as expected
			_, ok := cm.scheduleStopChs[schedTaskName]
			assert.False(t, ok, "expected scheduled task stop channel to be removed")
		case <-time.After(time.Second * 5):
			t.Fatal("runScheduledTask did not exit as expected")
		}
	})
}

func Test_ConditionMonitor_Run_context_cancel(t *testing.T) {
	w := new(mocks.Watcher)
	w.On("Watch", mock.Anything, mock.Anything).Return(nil).
		On("Size").Return(5).
		On("Stop").Return()

	cm := newTestConditionMonitor(nil)
	cm.watcher = w

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		err := cm.Run(ctx)
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

func Test_ConditionMonitor_Run_ActiveTask(t *testing.T) {
	// Set up tm with two tasks
	tm := newTestTasksManager()
	completedTasksCh := tm.EnableTestMode()

	for _, n := range []string{"task_a", "task_b"} {
		d := new(mocksD.Driver)
		d.On("Task").Return(enabledTestTask(t, n)).
			On("TemplateIDs").Return([]string{"tmpl_" + n}).
			On("RenderTemplate", mock.Anything).Return(true, nil).
			On("ApplyTask", mock.Anything).Return(nil).
			On("SetBufferPeriod")
		tm.drivers.Add(n, d)

		conf := validTaskConf
		conf.Name = config.String(n)
		tm.state.SetTask(conf)
	}

	// Set up condition monitor
	cm := newTestConditionMonitor(tm)
	cm.watcherCh = make(chan string, 5)

	// Set up watcher for tm
	ctx := context.Background()
	w := new(mocks.Watcher)
	w.On("Size").Return(5)
	w.On("Watch", ctx, cm.watcherCh).Return(nil)
	cm.watcher = w

	// Start Run
	errCh := make(chan error)
	go func() {
		err := cm.Run(ctx)
		if err != nil {
			errCh <- err
		}
	}()

	// Set task_a to active
	tm.drivers.SetActive("task_a")

	// Trigger twice on active task_a, task should not complete
	for i := 0; i < 2; i++ {
		cm.watcherCh <- "tmpl_task_a"
	}
	select {
	case <-completedTasksCh:
		t.Fatal("already active task should not have completed trigger")
	case <-time.After(time.Millisecond * 250):
		break // expected case
	}

	// Trigger on inactive task_b, task should complete
	cm.watcherCh <- "tmpl_task_b"
	select {
	case taskName := <-completedTasksCh:
		assert.Equal(t, "task_b", taskName)
	case <-time.After(time.Millisecond * 250):
		t.Fatal("inactive task should have completed when triggered")
	}

	// Set task_a to inactive, should expect two tasks to complete
	tm.drivers.SetInactive("task_a")
	for i := 0; i < 2; i++ {
		select {
		case taskName := <-completedTasksCh:
			assert.Equal(t, "task_a", taskName)
		case <-time.After(time.Millisecond * 250):
			t.Fatal("inactive task_a should have completed")
		}
	}

	// Notify on task_a again, should complete
	cm.watcherCh <- "tmpl_task_a"
	select {
	case taskName := <-completedTasksCh:
		assert.Equal(t, "task_a", taskName)
	case <-time.After(time.Millisecond * 250):
		t.Fatal("inactive task_a should have complete again")
	}
}

func Test_ConditionMonitor_Run_ScheduledTasks(t *testing.T) {
	tm := newTestTasksManager()
	tm.createdScheduleCh = make(chan string, 1)
	tm.EnableTestMode()

	// Set up condition monitor
	cm := newTestConditionMonitor(tm)
	cm.watcherCh = make(chan string, 5)

	// Set up watcher for condition monitor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := new(mocks.Watcher)
	w.On("Size").Return(5)
	w.On("Watch", ctx, cm.watcherCh).Return(nil)
	cm.watcher = w

	go cm.Run(ctx)

	createdTaskName := "created_scheduled_task"
	createdDriver := new(mocksD.Driver)
	createdDriver.On("Task").Return(scheduledTestTask(t, createdTaskName)).
		On("TemplateIDs").Return([]string{"tmpl_b"}).
		On("RenderTemplate", mock.Anything).Return(true, nil).
		On("ApplyTask", mock.Anything).Return(nil).
		On("SetBufferPeriod")
	_, err := tm.addTask(ctx, createdDriver)
	require.NoError(t, err)

	select {
	case n := <-tm.taskNotify:
		assert.Equal(t, createdTaskName, n)
	case <-time.After(5 * time.Second):
		t.Fatal("scheduled task did not run")
	}
	stopCh, ok := cm.scheduleStopChs[createdTaskName]
	assert.True(t, ok, "scheduled task stop channel not added to map")
	assert.NotNil(t, stopCh, "expected stop channel not to be nil")
}

func Test_ConditionMonitor_Run_ScheduledTasks_Stop(t *testing.T) {
	t.Parallel()
	// Tests receiving a stop notification for deleting a scheduled task

	// Setup task manager
	taskName := "scheduled_task"
	tm := newTestTasksManager()
	tm.deletedScheduleCh = make(chan string, 1)

	// Setup condition monitor
	cm := newTestConditionMonitor(tm)
	stopCh := make(chan struct{}, 1)
	cm.scheduleStopChs[taskName] = stopCh

	// Set up watcher for condition monitor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := new(mocks.Watcher)
	w.On("Size").Return(5)
	w.On("Watch", mock.Anything, mock.Anything).Return(nil)
	cm.watcher = w

	go cm.Run(ctx)

	// Send notification that task was deleted
	tm.deletedScheduleCh <- taskName

	// Verify the stop channel received message
	select {
	case <-time.After(1 * time.Second):
		t.Fatal("scheduled task was not notified to stop")
	case <-stopCh:
		// expected case
		_, ok := cm.scheduleStopChs[taskName]
		assert.False(t, ok, "scheduled task stop channel still in map")
	}
}

func Test_ConditionMonitor_WatchDep_context_cancel(t *testing.T) {
	t.Parallel()

	t.Run("cancel exits successfully", func(t *testing.T) {
		cm := newTestConditionMonitor(nil)

		// Mock watcher
		w := new(mocks.Watcher)
		waitErrCh := make(chan error, 1)
		var waitErrChRc <-chan error = waitErrCh
		waitErrCh <- nil
		w.On("WaitCh", mock.Anything).Return(waitErrChRc)
		w.On("Size", mock.Anything).Return(1)
		cm.watcher = w

		errCh := make(chan error)
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			if err := cm.WatchDep(ctx); err != nil {
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

		// Don't w.AssertExpectations(). Race condition on when cancel() is
		// called if and if watcher.Size() is called
	})

	t.Run("error exits successfully", func(t *testing.T) {
		cm := newTestConditionMonitor(nil)

		// Mock watcher
		w := new(mocks.Watcher)
		waitErrCh := make(chan error, 1)
		var waitErrChRc <-chan error = waitErrCh
		waitErrCh <- errors.New("error!")
		w.On("WaitCh", mock.Anything).Return(waitErrChRc)
		cm.watcher = w

		errCh := make(chan error)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			if err := cm.WatchDep(ctx); err != nil {
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
				Name:               config.String(validTaskName),
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

func newTestConditionMonitor(tm *TasksManager) *ConditionMonitor {
	cm := &ConditionMonitor{
		logger:          logging.NewNullLogger(),
		tasksManager:    newTestTasksManager(),
		scheduleStopChs: make(map[string](chan struct{})),
	}

	if tm != nil {
		cm.tasksManager = tm
	}

	return cm
}
