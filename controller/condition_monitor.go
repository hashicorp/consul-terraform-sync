package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/cronexpr"
)

// ConditionMonitor monitors the the conditions for all of the tasks and is
// responsible for triggering a task to execute. It uses the task manager to
// inform of starting / stopping task monitoring as well as executing a task
type ConditionMonitor struct {
	// TODO: placeholder. Will convert TaskManager methods to ConditionMonitor
}

// WatchDep is a helper method to start watching dependencies to allow templates
// to render. It will run until the caller cancels the context.
func (tm *TasksManager) WatchDep(ctx context.Context) error {
	tm.logger.Trace("starting template dependency monitoring")

	for ix := int64(0); ; ix++ {
		select {
		case err := <-tm.watcher.WaitCh(ctx):
			if err != nil {
				tm.logger.Error("error watching template dependencies", "error", err)
				return err
			}

		case <-ctx.Done():
			// stop for context canceled
			return ctx.Err()
		}
		tm.logDepSize(50, int64(ix))
	}
}

// Run runs the controller in read-write mode by continuously monitoring Consul
// catalog and using the driver to apply network infrastructure changes for
// any work that have been updated.
//
// Blocking call runs the main consul monitoring loop, which identifies triggers
// for dynamic tasks. Scheduled tasks use their own go routine to trigger on
// schedule.
func (tm *TasksManager) Run(ctx context.Context) error {
	// Assumes buffer_period has been set by taskManager

	errCh := make(chan error)
	if tm.watcherCh == nil {
		// Size of channel is larger than just current number of drivers
		// to account for additional tasks created via the API. Adding 10
		// is an arbitrarily chosen value.
		tm.watcherCh = make(chan string, tm.drivers.Len()+10)
	}
	if tm.scheduleStartCh == nil {
		// Size of channel is an arbitrarily chosen value.
		tm.scheduleStartCh = make(chan driver.Driver, 10)
	}
	if tm.deleteCh == nil {
		// Size of channel is an arbitrarily chosen value.
		tm.deleteCh = make(chan string, 10)
	}
	if tm.scheduleStopChs == nil {
		tm.scheduleStopChs = make(map[string](chan struct{}))
	}
	go func() {
		for {
			tm.logger.Trace("starting template dependency monitoring")
			err := tm.watcher.Watch(ctx, tm.watcherCh)
			if err == nil || err == context.Canceled {
				tm.logger.Info("stopping dependency monitoring")
				return
			}
			tm.logger.Error("error monitoring template dependencies", "error", err)
		}
	}()

	for i := int64(1); ; i++ {
		select {
		case tmplID := <-tm.watcherCh:
			d, ok := tm.drivers.GetTaskByTemplate(tmplID)
			if !ok {
				tm.logger.Debug("template was notified for update but the template ID does not match any task", "template_id", tmplID)
				continue
			}

			go tm.runDynamicTask(ctx, d) // errors are logged for now

		case d := <-tm.scheduleStartCh:
			// Run newly created scheduled tasks
			stopCh := make(chan struct{}, 1)
			tm.scheduleStopChs[d.Task().Name()] = stopCh
			go tm.runScheduledTask(ctx, d, stopCh)

		case n := <-tm.deleteCh:
			go tm.deleteTask(ctx, n)

		case err := <-errCh:
			return err

		case <-ctx.Done():
			tm.logger.Info("stopping controller")
			return ctx.Err()
		}

		tm.logDepSize(50, i)
	}
}

// runDynamicTask will try to render the template and apply the task if necessary.
func (tm *TasksManager) runDynamicTask(ctx context.Context, d driver.Driver) error {
	task := d.Task()
	taskName := task.Name()
	if task.IsScheduled() {
		// Schedule tasks are not dynamic and run in a different process
		return nil
	}
	if tm.drivers.IsMarkedForDeletion(taskName) {
		tm.logger.Trace("task is marked for deletion, skipping", taskNameLogKey, taskName)
		return nil
	}

	err := tm.waitForTaskInactive(ctx, taskName)
	if err != nil {
		return err
	}
	complete, err := tm.checkApply(ctx, d, true, false)
	if err != nil {
		tm.logger.Error("error applying task", taskNameLogKey, taskName, "error", err)
		return err
	}

	if tm.taskNotify != nil && complete {
		tm.taskNotify <- taskName
	}
	return nil
}

// runScheduledTask starts up a go-routine for a given scheduled task/driver.
// The go-routine will manage the task's schedule and trigger the task on time.
// If there are dependency changes since the task's last run time, then the task
// will also apply.
func (tm *TasksManager) runScheduledTask(ctx context.Context, d driver.Driver, stopCh chan struct{}) error {
	task := d.Task()
	taskName := task.Name()

	cond, ok := task.Condition().(*config.ScheduleConditionConfig)
	if !ok {
		tm.logger.Error("unexpected condition while running a scheduled "+
			"condition", taskNameLogKey, taskName, "condition_type",
			fmt.Sprintf("%T", task.Condition()))
		return fmt.Errorf("error: expected a schedule condition but got "+
			"condition type %T", task.Condition())
	}

	expr, err := cronexpr.Parse(*cond.Cron)
	if err != nil {
		tm.logger.Error("error parsing task cron", taskNameLogKey, taskName,
			"cron", *cond.Cron, "error", err)
		return err
	}

	nextTime := expr.Next(time.Now())
	waitTime := time.Until(nextTime)
	tm.logger.Info("scheduled task next run time", taskNameLogKey, taskName,
		"wait_time", waitTime, "next_runtime", nextTime)

	for {
		select {
		case <-time.After(waitTime):
			if _, ok := tm.drivers.Get(taskName); !ok {
				// Should not happen in the typical workflow, but stopping if in this state
				tm.logger.Debug("scheduled task no longer exists", taskNameLogKey, taskName)
				tm.logger.Info("stopping deleted scheduled task", taskNameLogKey, taskName)
				delete(tm.scheduleStopChs, taskName)
				return nil
			}

			if tm.drivers.IsMarkedForDeletion(taskName) {
				tm.logger.Trace("task is marked for deletion, skipping", taskNameLogKey, taskName)
				return nil
			}

			tm.logger.Info("time for scheduled task", taskNameLogKey, taskName)
			if tm.drivers.IsActive(taskName) {
				// The driver is currently active with the task, initiated by an ad-hoc run.
				tm.logger.Trace("task is active", taskNameLogKey, taskName)
				continue
			}

			complete, err := tm.checkApply(ctx, d, true, false)
			if err != nil {
				// print error but continue
				tm.logger.Error("error applying task", taskNameLogKey, taskName, "error", err)
			}

			if tm.taskNotify != nil && complete {
				tm.taskNotify <- taskName
			}

			nextTime := expr.Next(time.Now())
			waitTime = time.Until(nextTime)
			tm.logger.Info("scheduled task next run time", taskNameLogKey, taskName,
				"wait_time", waitTime, "next_runtime", nextTime)
		case <-stopCh:
			tm.logger.Info("stopping scheduled task", taskNameLogKey, taskName)
			return nil
		case <-ctx.Done():
			tm.logger.Info("stopping scheduled task", taskNameLogKey, taskName)
			return ctx.Err()
		}
	}
}

// EnableTestMode is a helper for testing which tasks were triggered and
// executed. Callers of this method must consume from TaskNotify channel to
// prevent the buffered channel from filling and causing a dead lock.
func (tm *TasksManager) EnableTestMode() <-chan string {
	tasks := tm.state.GetAllTasks()
	tm.taskNotify = make(chan string, tasks.Len())
	return tm.taskNotify
}

// logDepSize logs the watcher dependency size every nth iteration. Set the
// iterator to a negative value to log each iteration.
func (tm *TasksManager) logDepSize(n uint, i int64) {
	depSize := tm.watcher.Size()
	if i%int64(n) == 0 || i < 0 {
		tm.logger.Debug("watching dependencies", "dependency_size", depSize)
		if depSize > templates.DepSizeWarning {
			tm.logger.Warn(fmt.Sprintf(" watching more than %d dependencies could "+
				"DDoS your Consul cluster: %d", templates.DepSizeWarning, depSize))
		}
	}
}
