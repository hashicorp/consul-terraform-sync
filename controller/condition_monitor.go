// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
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

	watcherCh chan string

	// scheduleStopChs is a map of channels used to stop scheduled tasks
	scheduleStopChs map[string](chan struct{})
}

// NewConditionMonitor configures a new condition monitor
func NewConditionMonitor(tm *TasksManager, w templates.Watcher) *ConditionMonitor {
	logger := logging.Global().Named(tasksManagerSystemName)

	return &ConditionMonitor{
		logger:          logger,
		watcher:         w,
		tasksManager:    tm,
		scheduleStopChs: make(map[string](chan struct{})),
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
		cm.logDepSize(50, ix)
	}
}

// Run runs a continuous loop which monitors the Consul catalog. On changes, Run
// uses the TasksManager to execute tasks to update network infrastructure.
//
// The blocking call runs the main Consul monitoring loop, which identifies triggers
// for dynamic tasks. Scheduled tasks use their own go routine to trigger on
// schedule.
func (cm *ConditionMonitor) Run(ctx context.Context) error {
	// Assumes buffer_period was set by tasksManager when adding task to CTS

	if cm.watcherCh == nil {
		// Size of channel is the number of CTS tasks configured at initialization
		// +10 for any additional tasks created during runtime. 10 arbitrarily chosen
		tasks := cm.tasksManager.Tasks(ctx)
		cm.watcherCh = make(chan string, tasks.Len()+10)
	}
	if cm.scheduleStopChs == nil {
		cm.scheduleStopChs = make(map[string](chan struct{}))
	}

	// This wait prevents timing issues where repeated calls to Run()
	// and cancelling a context would have overlapping Watch calls.
	var waitForWatchCancel sync.WaitGroup
	waitForWatchCancel.Add(1)
	go func() {
		defer waitForWatchCancel.Done()
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
			taskName, ok := cm.tasksManager.TaskByTemplate(tmplID)
			if !ok {
				cm.logger.Debug("template was notified for update but the template ID does not match any task", "template_id", tmplID)
				continue
			}

			go cm.runDynamicTask(ctx, taskName) // errors are logged for now

		case taskName := <-cm.tasksManager.WatchCreatedScheduleTasks():
			// Cancel existing goroutines before creating the new scheduled task.
			if stopCh, ok := cm.scheduleStopChs[taskName]; ok && stopCh != nil {
				stopCh <- struct{}{}
			}
			// Run newly created scheduled tasks
			stopCh := make(chan struct{}, 1)
			cm.scheduleStopChs[taskName] = stopCh
			go cm.runScheduledTask(ctx, taskName, stopCh)

		case taskName := <-cm.tasksManager.WatchDeletedScheduleTask():
			// Stop deleted scheduled tasks
			stopCh := cm.scheduleStopChs[taskName]
			if stopCh != nil {
				stopCh <- struct{}{}
			}
			delete(cm.scheduleStopChs, taskName)

		case <-ctx.Done():
			cm.logger.Info("stop monitoring tasks")
			waitForWatchCancel.Wait()

			// Delete all stop channels. They do not need to be notified, since the shared-context is closed.
			for taskName := range cm.scheduleStopChs {
				delete(cm.scheduleStopChs, taskName)
			}
			return ctx.Err()
		}

		cm.logDepSize(50, i)
	}
}

// runDynamicTask will execute the task as necessary
func (cm *ConditionMonitor) runDynamicTask(ctx context.Context, taskName string) error {
	logger := cm.logger.With(taskNameLogKey, taskName)

	task, err := cm.tasksManager.Task(ctx, taskName)
	if err != nil {
		logger.Warn("dynamic task cannot be run. task may have been deleted",
			"error", err)
		return err
	}

	if _, ok := task.Condition.(*config.ScheduleConditionConfig); ok {
		logger.Error("unexpected scheduled condition while running a dynamic " +
			"condition")
		return fmt.Errorf("error: expected a dynamic condition but got " +
			"a scheduled condition type")
	}

	if err := cm.tasksManager.TaskRunNow(ctx, taskName); err != nil {
		logger.Error("error running task", "error", err)
		return err
	}

	return nil
}

// runScheduledTask starts up a go-routine for a given scheduled task/driver.
// The go-routine will manage the task's schedule and trigger the task on time.
// If there are dependency changes since the task's last run time, then the task
// will also apply.
func (cm *ConditionMonitor) runScheduledTask(ctx context.Context, taskName string, stopCh chan struct{}) error {
	logger := cm.logger.With(taskNameLogKey, taskName)

	task, err := cm.tasksManager.Task(ctx, taskName)
	if err != nil {
		logger.Warn("scheduled task cannot be run. task may have been deleted",
			"error", err)
		return err
	}

	cond, ok := task.Condition.(*config.ScheduleConditionConfig)
	if !ok {
		logger.Error("unexpected condition while running a scheduled "+
			"condition", "condition_type", fmt.Sprintf("%T", task.Condition))
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
	logger.Info("scheduled task next run time", "wait_time", waitTime,
		"next_runtime", nextTime)

	for {
		select {
		case <-time.After(waitTime):
			if _, err := cm.tasksManager.Task(ctx, taskName); err != nil {
				// Should not happen in the typical workflow, but stopping if
				// in this state
				logger.Debug("scheduled task no longer exists")
				logger.Info("stopping deleted scheduled task")
				delete(cm.scheduleStopChs, taskName)
				return nil
			}

			if err := cm.tasksManager.TaskRunNow(ctx, taskName); err != nil {
				// print error but continue
				logger.Error("error running task", "error", err)
			}

			nextTime := expr.Next(time.Now())
			waitTime = time.Until(nextTime)
			logger.Info("scheduled task next run time", "wait_time", waitTime,
				"next_runtime", nextTime)
		case <-stopCh:
			logger.Info("stopping scheduled task")
			return nil
		case <-ctx.Done():
			logger.Info("stopping scheduled task")
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
