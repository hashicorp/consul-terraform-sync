package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/state"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/cronexpr"
)

// ConditionMonitor monitors task condition in order to trigger tasks
type ConditionMonitor struct {
	logger logging.Logger

	state        state.Store
	tasksManager *TasksManager
	watcher      templates.Watcher

	// scheduleStopChs is a map of channels used to stop scheduled tasks
	scheduleStopChs map[string](chan struct{})

	// deleteCh is used to delete tasks
	deleteCh chan string

	// taskNotify is only initialized if EnableTestMode() is used. It provides
	// tests insight into which tasks were triggered and had completed
	taskNotify chan string
}

// NewConditionMonitor configures a condition monitor that will trigger tasks
// to run
func NewConditionMonitor(state state.Store, tasksManager *TasksManager,
	watcher templates.Watcher) (*ConditionMonitor, error) {

	logger := logging.Global().Named(ctrlSystemName)

	errStr := "error creating task runner %s is nil"
	if state == nil {
		return nil, fmt.Errorf(errStr, "state store")
	}
	if tasksManager == nil {
		return nil, fmt.Errorf(errStr, "task manager")
	}
	if watcher == nil {
		return nil, fmt.Errorf(errStr, "watcher")
	}

	return &ConditionMonitor{
		logger:       logger,
		state:        state,
		tasksManager: tasksManager,
		watcher:      watcher,

		// Size of channel is an arbitrarily chosen value.
		deleteCh:        make(chan string, 10),
		scheduleStopChs: make(map[string](chan struct{})),
	}, nil
}

// Start starts monitoring the condition to trigger a task
//
// A blocking call continuously monitors Consul catalog using hcat watcher to
// know when to trigger dynamic tasks.
//
// The blocking call also identifies new scheduled tasks which use their own go
// routine to trigger on schedule
func (cm *ConditionMonitor) Start(ctx context.Context) error {
	// Size of channel is the number of CTS tasks configured at initialization
	// +10 for any additional tasks created during runtime. 10 arbitrarily chosen
	conf := cm.state.GetConfig()
	watcherCh := make(chan string, conf.Tasks.Len()+10)

	go func() {
		for {
			cm.logger.Trace("starting template dependency monitoring")
			err := cm.watcher.Watch(ctx, watcherCh)
			if err == nil || err == context.Canceled {
				cm.logger.Info("stopping dependency monitoring")
				return
			}
			cm.logger.Error("error monitoring template dependencies", "error", err)
		}
	}()

	errCh := make(chan error)
	for i := int64(1); ; i++ {
		select {
		case cmplID := <-watcherCh:
			taskName, ok := cm.tasksManager.TaskByTemplate(cmplID)
			if !ok {
				cm.logger.Debug("received template notification for update but"+
					" the template ID does not match any task", "template_id", cmplID)
				continue
			}
			go cm.runDynamicTask(ctx, taskName) // errors are logged for now

		case taskName := <-cm.tasksManager.WatchNewScheduleTaskCh():
			cm.logger.Info("received scheduled task to monitor", taskNameLogKey, taskName)
			stopCh := make(chan struct{}, 1)
			cm.scheduleStopChs[taskName] = stopCh
			go cm.runScheduledTask(ctx, taskName, stopCh)

		case taskName := <-cm.tasksManager.WatchDeletedScheduleTaskCh():
			cm.logger.Info("stop monitoring scheduled task", taskNameLogKey, taskName)
			stopCh := cm.scheduleStopChs[taskName]
			stopCh <- struct{}{}
			delete(cm.scheduleStopChs, taskName)

		case err := <-errCh:
			return err

		case <-ctx.Done():
			cm.logger.Info("stopping controller")
			return ctx.Err()
		}

		logDepSize(cm.watcher, cm.logger, 50, i)
	}
}

// runDynamicTask will try to render the template and apply the task if necessary.
func (cm *ConditionMonitor) runDynamicTask(ctx context.Context, taskName string) error {
	task, ok := cm.state.GetTask(taskName)
	if !ok {
		// double check that task isn't up for deletion
		cm.logger.Trace("task is removed from state and in the process of cleanup, skipping", taskNameLogKey, taskName)
		return nil
	}

	if _, ok := task.Condition.(*config.ScheduleConditionConfig); ok {
		// Precautionary check: schedule tasks are not dynamic and run in a
		// different process
		return nil
	}

	complete, err := cm.tasksManager.TaskRunNow(ctx, taskName)
	if err != nil {
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
func (cm *ConditionMonitor) runScheduledTask(ctx context.Context, taskName string, stopCh chan struct{}) error {
	logger := cm.logger.With(taskNameLogKey, taskName)

	task, ok := cm.state.GetTask(taskName)
	if !ok {
		logger.Trace("task is removed from state and in the process of cleanup, skipping")
		return nil
	}

	cond, ok := task.Condition.(*config.ScheduleConditionConfig)
	if !ok {
		cm.logger.Error("unexpected condition while running a scheduled "+
			"condition", taskNameLogKey, taskName, "condition_type",
			fmt.Sprintf("%T", task.Condition))
		return fmt.Errorf("error: expected a schedule condition but got "+
			"condition type %T", task.Condition)
	}

	expr, err := cronexpr.Parse(*cond.Cron)
	if err != nil {
		logger.Error("error parsing task cron", "cron", *cond.Cron, "error", err)
		return err
	}

	nextTime := expr.Next(time.Now())
	waitTime := time.Until(nextTime)
	cm.logger.Info("scheduled task next run time", "wait_time", waitTime,
		"next_runtime", nextTime)

	for {
		select {
		case <-time.After(waitTime):
			cm.logger.Info("time for scheduled task", taskNameLogKey, taskName)
			complete, err := cm.tasksManager.TaskRunNow(ctx, taskName)
			if err != nil {
				// print error but continue
				cm.logger.Error("error applying task %q: %s",
					taskNameLogKey, taskName, "error", err)
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

// EnableTestMode is a helper for testing which tasks were triggered and
// executed. Callers of this method must consume from TaskNotify channel to
// prevent the buffered channel from filling and causing a dead lock.
func (cm *ConditionMonitor) EnableTestMode() <-chan string {
	conf := cm.state.GetConfig()
	cm.taskNotify = make(chan string, conf.Tasks.Len())
	return cm.taskNotify
}

// logDepSize logs the watcher dependency size every nth iteration. Set the
// iterator to a negative value to log each iteration.
func logDepSize(watcher templates.Watcher, logger logging.Logger, n uint, i int64) {
	depSize := watcher.Size()
	if i%int64(n) == 0 || i < 0 {
		logger.Debug("watching dependencies", "dependency_size", depSize)
		if depSize > templates.DepSizeWarning {
			logger.Warn(fmt.Sprintf(" watching more than %d dependencies could "+
				"DDoS your Consul cluster: %d", templates.DepSizeWarning, depSize))
		}
	}
}
