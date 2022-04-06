package controller

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	"github.com/hashicorp/consul-terraform-sync/config"
// 	"github.com/hashicorp/consul-terraform-sync/driver"
// 	"github.com/hashicorp/cronexpr"
// )

// type ScheduleTaskRunner struct {
// 	driver.Driver

// 	// newScheduleCh is used to add new scheduled tasks
// 	// during runtime only?
// 	newScheduleCh chan driver.Driver
// 	// scheduleStopChs is a map of channels used to stop scheduled tasks
// 	scheduleStopChs map[string](chan struct{})
// }

// func NewScheduleTasksRunner() *ScheduleTaskRunner {
// 	return &ScheduleTaskRunner{}
// }

// func (s *ScheduleTaskRunner) StartNewTask(ctx context.Context, d driver.Driver) {
// 	stopCh := make(chan struct{}, 1)
// 	s.scheduleStopChs[d.Task().Name()] = stopCh
// 	// go s.runScheduledTask(ctx, d, stopCh)
// }

// // runScheduledTask starts up a go-routine for a given scheduled task/driver.
// // The go-routine will manage the task's schedule and trigger the task on time.
// // If there are dependency changes since the task's last run time, then the task
// // will also apply.
// func (tm *TasksManager) runScheduledTask(ctx context.Context, d driver.Driver, stopCh chan struct{}) error {
// 	task := d.Task()
// 	taskName := task.Name()

// 	cond, ok := task.Condition().(*config.ScheduleConditionConfig)
// 	if !ok {
// 		tm.logger.Error("unexpected condition while running a scheduled "+
// 			"condition", taskNameLogKey, taskName, "condition_type",
// 			fmt.Sprintf("%T", task.Condition()))
// 		return fmt.Errorf("error: expected a schedule condition but got "+
// 			"condition type %T", task.Condition())
// 	}

// 	expr, err := cronexpr.Parse(*cond.Cron)
// 	if err != nil {
// 		tm.logger.Error("error parsing task cron", taskNameLogKey, taskName,
// 			"cron", *cond.Cron, "error", err)
// 		return err
// 	}

// 	nextTime := expr.Next(time.Now())
// 	waitTime := time.Until(nextTime)
// 	tm.logger.Info("scheduled task next run time", taskNameLogKey, taskName,
// 		"wait_time", waitTime, "next_runtime", nextTime)

// 	for {
// 		select {
// 		case <-time.After(waitTime):
// 			if _, ok := tm.drivers.Get(taskName); !ok {
// 				// Should not happen in the typical workflow, but stopping if in this state
// 				tm.logger.Debug("scheduled task no longer exists", taskNameLogKey, taskName)
// 				tm.logger.Info("stopping deleted scheduled task", taskNameLogKey, taskName)
// 				delete(tm.scheduleStopChs, taskName)
// 				return nil
// 			}

// 			if tm.drivers.IsMarkedForDeletion(taskName) {
// 				tm.logger.Trace("task is marked for deletion, skipping", taskNameLogKey, taskName)
// 				return nil
// 			}

// 			tm.logger.Info("time for scheduled task", taskNameLogKey, taskName)
// 			if tm.drivers.IsActive(taskName) {
// 				// The driver is currently active with the task, initiated by an ad-hoc run.
// 				tm.logger.Trace("task is active", taskNameLogKey, taskName)
// 				continue
// 			}

// 			complete, err := tm.checkApply(ctx, d, true, false)
// 			if err != nil {
// 				// print error but continue
// 				tm.logger.Error("error applying task %q: %s",
// 					taskNameLogKey, taskName, "error", err)
// 			}

// 			if tm.taskNotify != nil && complete {
// 				tm.taskNotify <- taskName
// 			}

// 			nextTime := expr.Next(time.Now())
// 			waitTime = time.Until(nextTime)
// 			tm.logger.Info("scheduled task next run time", taskNameLogKey, taskName,
// 				"wait_time", waitTime, "next_runtime", nextTime)
// 		case <-stopCh:
// 			tm.logger.Info("stopping scheduled task", taskNameLogKey, taskName)
// 			return nil
// 		case <-ctx.Done():
// 			tm.logger.Info("stopping scheduled task", taskNameLogKey, taskName)
// 			return ctx.Err()
// 		}
// 	}
// }
