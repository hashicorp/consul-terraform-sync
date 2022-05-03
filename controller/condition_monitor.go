package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/cronexpr"
)

// ConditionMonitor monitors the the conditions for all of the tasks and is
// responsible for triggering a task to execute. It uses the task manager to
// inform of starting / stopping task monitoring as well as executing a task
type ConditionMonitor struct {
	logger logging.Logger

	watcher      templates.Watcher
	tasksManager *TasksManager
}

// NewConditionMonitor configures a new condition monitor
func NewConditionMonitor(tm *TasksManager, w templates.Watcher) *ConditionMonitor {
	logger := logging.Global().Named(tasksManagerSystemName)

	return &ConditionMonitor{
		logger:       logger,
		watcher:      w,
		tasksManager: tm,
	}
}

// WatchDep is a helper method to start watching dependencies to allow templates
// to render. It will run until the caller cancels the context.
func (cm *ConditionMonitor) WatchDep(ctx context.Context) error {
	cm.logger.Trace("starting template dependency monitoring")

	for ix := int64(0); ; ix++ {
		select {
		case err := <-cm.watcher.WaitCh(ctx):
			if err != nil {
				cm.logger.Error("error watching template dependencies", "error", err)
				return err
			}

		case <-ctx.Done():
			// stop for context canceled
			return ctx.Err()
		}
		cm.logDepSize(50, int64(ix))
	}
}

// Run runs the controller in read-write mode by continuously monitoring Consul
// catalog and using the driver to apply network infrastructure changes for
// any work that have been updated.
//
// Blocking call runs the main consul monitoring loop, which identifies triggers
// for dynamic tasks. Scheduled tasks use their own go routine to trigger on
// schedule.
func (cm *ConditionMonitor) Run(ctx context.Context) error {
	// Assumes buffer_period has been set by taskManager

	errCh := make(chan error)
	if cm.watcherCh == nil {
		// Size of channel is larger than just current number of drivers
		// to account for additional tasks created via the API. Adding 10
		// is an arbitrarily chosen value.
		cm.watcherCh = make(chan string, cm.drivers.Len()+10)
	}
	if cm.scheduleStartCh == nil {
		// Size of channel is an arbitrarily chosen value.
		cm.scheduleStartCh = make(chan driver.Driver, 10)
	}
	if cm.deleteCh == nil {
		// Size of channel is an arbitrarily chosen value.
		cm.deleteCh = make(chan string, 10)
	}
	if cm.scheduleStopChs == nil {
		cm.scheduleStopChs = make(map[string](chan struct{}))
	}
	go func() {
		for {
			cm.logger.Trace("starting template dependency monitoring")
			err := cm.watcher.Watch(ctx, cm.watcherCh)
			if err == nil || err == context.Canceled {
				cm.logger.Info("stopping dependency monitoring")
				return
			}
			cm.logger.Error("error monitoring template dependencies", "error", err)
		}
	}()

	for i := int64(1); ; i++ {
		select {
		case tmplID := <-cm.watcherCh:
			d, ok := cm.drivers.GetTaskByTemplate(tmplID)
			if !ok {
				cm.logger.Debug("template was notified for update but the template ID does not match any task", "template_id", tmplID)
				continue
			}

			go cm.runDynamicTask(ctx, d) // errors are logged for now

		case d := <-cm.scheduleStartCh:
			// Run newly created scheduled tasks
			stopCh := make(chan struct{}, 1)
			cm.scheduleStopChs[d.Task().Name()] = stopCh
			go cm.runScheduledTask(ctx, d, stopCh)

		case n := <-cm.deleteCh:
			go cm.deleteTask(ctx, n)

		case err := <-errCh:
			return err

		case <-ctx.Done():
			cm.logger.Info("stopping controller")
			return ctx.Err()
		}

		cm.logDepSize(50, i)
	}
}

// runDynamicTask will try to render the template and apply the task if necessary.
func (cm *ConditionMonitor) runDynamicTask(ctx context.Context, d driver.Driver) error {
	task := d.Task()
	taskName := task.Name()
	if task.IsScheduled() {
		// Schedule tasks are not dynamic and run in a different process
		return nil
	}
	if cm.drivers.IsMarkedForDeletion(taskName) {
		cm.logger.Trace("task is marked for deletion, skipping", taskNameLogKey, taskName)
		return nil
	}

	err := cm.waitForTaskInactive(ctx, taskName)
	if err != nil {
		return err
	}
	complete, err := cm.checkApply(ctx, d, true, false)
	if err != nil {
		tm.logger.Error("error applying task", taskNameLogKey, taskName, "error", err)
		return err
	}

	if cm.taskNotify != nil && complete {
		cm.taskNotify <- taskName
	}
	return nil
}

// runScheduledTask starts up a go-routine for a given scheduled task/driver.
// The go-routine will manage the task's schedule and trigger the task on time.
// If there are dependency changes since the task's last run time, then the task
// will also apply.
func (cm *ConditionMonitor) runScheduledTask(ctx context.Context, d driver.Driver, stopCh chan struct{}) error {
	task := d.Task()
	taskName := task.Name()

	cond, ok := task.Condition().(*config.ScheduleConditionConfig)
	if !ok {
		cm.logger.Error("unexpected condition while running a scheduled "+
			"condition", taskNameLogKey, taskName, "condition_type",
			fmt.Sprintf("%T", task.Condition()))
		return fmt.Errorf("error: expected a schedule condition but got "+
			"condition type %T", task.Condition())
	}

	expr, err := cronexpr.Parse(*cond.Cron)
	if err != nil {
		cm.logger.Error("error parsing task cron", taskNameLogKey, taskName,
			"cron", *cond.Cron, "error", err)
		return err
	}

	nextTime := expr.Next(time.Now())
	waitTime := time.Until(nextTime)
	cm.logger.Info("scheduled task next run time", taskNameLogKey, taskName,
		"wait_time", waitTime, "next_runtime", nextTime)

	for {
		select {
		case <-time.After(waitTime):
			if _, ok := cm.drivers.Get(taskName); !ok {
				// Should not happen in the typical workflow, but stopping if in this state
				cm.logger.Debug("scheduled task no longer exists", taskNameLogKey, taskName)
				cm.logger.Info("stopping deleted scheduled task", taskNameLogKey, taskName)
				delete(cm.scheduleStopChs, taskName)
				return nil
			}

			if cm.drivers.IsMarkedForDeletion(taskName) {
				cm.logger.Trace("task is marked for deletion, skipping", taskNameLogKey, taskName)
				return nil
			}

			cm.logger.Info("time for scheduled task", taskNameLogKey, taskName)
			if cm.drivers.IsActive(taskName) {
				// The driver is currently active with the task, initiated by an ad-hoc run.
				cm.logger.Trace("task is active", taskNameLogKey, taskName)
				continue
			}

			complete, err := cm.checkApply(ctx, d, true, false)
			if err != nil {
				// print error but continue
				cm.logger.Error("error applying task", taskNameLogKey, taskName, "error", err)
			}

			if cm.taskNotify != nil && complete {
				cm.taskNotify <- taskName
			}

			nextTime := expr.Next(time.Now())
			waitTime = time.Until(nextTime)
			cm.logger.Info("scheduled task next run time", taskNameLogKey, taskName,
				"wait_time", waitTime, "next_runtime", nextTime)
		case <-stopCh:
			cm.logger.Info("stopping scheduled task", taskNameLogKey, taskName)
			return nil
		case <-ctx.Done():
			cm.logger.Info("stopping scheduled task", taskNameLogKey, taskName)
			return ctx.Err()
		}
	}
}

// logDepSize logs the watcher dependency size every nth iteration. Set the
// iterator to a negative value to log each iteration.
func (cm *ConditionMonitor) logDepSize(n uint, i int64) {
	depSize := cm.watcher.Size()
	if i%int64(n) == 0 || i < 0 {
		cm.logger.Debug("watching dependencies", "dependency_size", depSize)
		if depSize > templates.DepSizeWarning {
			cm.logger.Warn(fmt.Sprintf(" watching more than %d dependencies could "+
				"DDoS your Consul cluster: %d", templates.DepSizeWarning, depSize))
		}
	}
}
