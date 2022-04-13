package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/retry"
	"github.com/hashicorp/cronexpr"
)

var (
	_ Controller = (*ReadWrite)(nil)

	// Number of times to retry attempts
	defaultRetry = 2
)

// ReadWrite is the controller to run in read-write mode
type ReadWrite struct {
	*baseController
	retry retry.Retry

	watcherCh chan string

	// scheduleStartCh is used to coordinate scheduled tasks created via the API
	scheduleStartCh chan driver.Driver
	// scheduleStopChs is a map of channels used to stop scheduled tasks
	scheduleStopChs map[string](chan struct{})

	// deleteCh is used to coordinate task deletion via the API
	deleteCh chan string

	// taskNotify is only initialized if EnableTestMode() is used. It provides
	// tests insight into which tasks were triggered and had completed
	taskNotify chan string
}

// NewReadWrite configures and initializes a new ReadWrite controller
func NewReadWrite(conf *config.Config) (*ReadWrite, error) {
	baseCtrl, err := newBaseController(conf)
	if err != nil {
		return nil, err
	}

	return &ReadWrite{
		baseController:  baseCtrl,
		retry:           retry.NewRetry(defaultRetry, time.Now().UnixNano()),
		scheduleStartCh: make(chan driver.Driver, 10), // arbitrarily chosen size
		deleteCh:        make(chan string, 10),        // arbitrarily chosen size
		scheduleStopChs: make(map[string](chan struct{})),
	}, nil
}

// Init initializes the controller before it can be run. Ensures that
// driver is initializes, works are created for each task.
func (rw *ReadWrite) Init(ctx context.Context) error {
	return rw.init(ctx)
}

// Run runs the controller in read-write mode by continuously monitoring Consul
// catalog and using the driver to apply network infrastructure changes for
// any work that have been updated.
//
// Blocking call runs the main consul monitoring loop, which identifies triggers
// for dynamic tasks. Scheduled tasks use their own go routine to trigger on
// schedule.
func (rw *ReadWrite) Run(ctx context.Context) error {
	// Only initialize buffer periods for running the full loop and not for Once
	// mode so it can immediately render the first time.
	rw.drivers.SetBufferPeriod()

	for _, d := range rw.drivers.Map() {
		if d.Task().IsScheduled() {
			stopCh := make(chan struct{}, 1)
			rw.scheduleStopChs[d.Task().Name()] = stopCh
			go rw.runScheduledTask(ctx, d, stopCh)
		}
	}

	errCh := make(chan error)
	if rw.watcherCh == nil {
		// Size of channel is larger than just current number of drivers
		// to account for additional tasks created via the API. Adding 10
		// is an arbitrarily chosen value.
		rw.watcherCh = make(chan string, rw.drivers.Len()+10)
	}
	if rw.scheduleStartCh == nil {
		// Size of channel is an arbitrarily chosen value.
		rw.scheduleStartCh = make(chan driver.Driver, 10)
	}
	if rw.deleteCh == nil {
		// Size of channel is an arbitrarily chosen value.
		rw.deleteCh = make(chan string, 10)
	}
	if rw.scheduleStopChs == nil {
		rw.scheduleStopChs = make(map[string](chan struct{}))
	}
	go func() {
		for {
			rw.logger.Trace("starting template dependency monitoring")
			err := rw.watcher.Watch(ctx, rw.watcherCh)
			if err == nil || err == context.Canceled {
				rw.logger.Info("stopping dependency monitoring")
				return
			}
			rw.logger.Error("error monitoring template dependencies", "error", err)
		}
	}()

	for i := int64(1); ; i++ {
		select {
		case tmplID := <-rw.watcherCh:
			d, ok := rw.drivers.GetTaskByTemplate(tmplID)
			if !ok {
				rw.logger.Debug("template was notified for update but the template ID does not match any task", "template_id", tmplID)
				continue
			}

			go rw.runDynamicTask(ctx, d) // errors are logged for now

		case d := <-rw.scheduleStartCh:
			// Run newly created scheduled tasks
			stopCh := make(chan struct{}, 1)
			rw.scheduleStopChs[d.Task().Name()] = stopCh
			go rw.runScheduledTask(ctx, d, stopCh)

		case n := <-rw.deleteCh:
			go rw.deleteTask(ctx, n)

		case err := <-errCh:
			return err

		case <-ctx.Done():
			rw.logger.Info("stopping controller")
			return ctx.Err()
		}

		rw.logDepSize(50, i)
	}
}

// runDynamicTask will try to render the template and apply the task if necessary.
func (rw *ReadWrite) runDynamicTask(ctx context.Context, d driver.Driver) error {
	task := d.Task()
	taskName := task.Name()
	if task.IsScheduled() {
		// Schedule tasks are not dynamic and run in a different process
		return nil
	}
	if rw.drivers.IsMarkedForDeletion(taskName) {
		rw.logger.Trace("task is marked for deletion, skipping", taskNameLogKey, taskName)
		return nil
	}

	err := rw.waitForTaskInactive(ctx, taskName)
	if err != nil {
		return err
	}
	complete, err := rw.checkApply(ctx, d, true, false)
	if err != nil {
		return err
	}

	if rw.taskNotify != nil && complete {
		rw.taskNotify <- taskName
	}
	return nil
}

// runScheduledTask starts up a go-routine for a given scheduled task/driver.
// The go-routine will manage the task's schedule and trigger the task on time.
// If there are dependency changes since the task's last run time, then the task
// will also apply.
func (rw *ReadWrite) runScheduledTask(ctx context.Context, d driver.Driver, stopCh chan struct{}) error {
	task := d.Task()
	taskName := task.Name()

	cond, ok := task.Condition().(*config.ScheduleConditionConfig)
	if !ok {
		rw.logger.Error("unexpected condition while running a scheduled "+
			"condition", taskNameLogKey, taskName, "condition_type",
			fmt.Sprintf("%T", task.Condition()))
		return fmt.Errorf("error: expected a schedule condition but got "+
			"condition type %T", task.Condition())
	}

	expr, err := cronexpr.Parse(*cond.Cron)
	if err != nil {
		rw.logger.Error("error parsing task cron", taskNameLogKey, taskName,
			"cron", *cond.Cron, "error", err)
		return err
	}

	nextTime := expr.Next(time.Now())
	waitTime := time.Until(nextTime)
	rw.logger.Info("scheduled task next run time", taskNameLogKey, taskName,
		"wait_time", waitTime, "next_runtime", nextTime)

	for {
		select {
		case <-time.After(waitTime):
			if _, ok := rw.drivers.Get(taskName); !ok {
				// Should not happen in the typical workflow, but stopping if in this state
				rw.logger.Debug("scheduled task no longer exists", taskNameLogKey, taskName)
				rw.logger.Info("stopping deleted scheduled task", taskNameLogKey, taskName)
				delete(rw.scheduleStopChs, taskName)
				return nil
			}

			if rw.drivers.IsMarkedForDeletion(taskName) {
				rw.logger.Trace("task is marked for deletion, skipping", taskNameLogKey, taskName)
				return nil
			}

			rw.logger.Info("time for scheduled task", taskNameLogKey, taskName)
			if rw.drivers.IsActive(taskName) {
				// The driver is currently active with the task, initiated by an ad-hoc run.
				rw.logger.Trace("task is active", taskNameLogKey, taskName)
				continue
			}

			complete, err := rw.checkApply(ctx, d, true, false)
			if err != nil {
				// print error but continue
				rw.logger.Error("error applying task %q: %s",
					taskNameLogKey, taskName, "error", err)
			}

			if rw.taskNotify != nil && complete {
				rw.taskNotify <- taskName
			}

			nextTime := expr.Next(time.Now())
			waitTime = time.Until(nextTime)
			rw.logger.Info("scheduled task next run time", taskNameLogKey, taskName,
				"wait_time", waitTime, "next_runtime", nextTime)
		case <-stopCh:
			rw.logger.Info("stopping scheduled task", taskNameLogKey, taskName)
			return nil
		case <-ctx.Done():
			rw.logger.Info("stopping scheduled task", taskNameLogKey, taskName)
			return ctx.Err()
		}
	}
}

// Once runs the controller in read-write mode making sure each template has
// been fully rendered and the task run, then it returns.
func (rw *ReadWrite) Once(ctx context.Context) error {
	rw.logger.Info("executing all tasks once through")

	// run consecutively to keep logs in order
	return rw.onceConsecutive(ctx)
}

// onceConsecutive runs all tasks consecutively until each task has completed once
func (rw *ReadWrite) onceConsecutive(ctx context.Context) error {
	driversCopy := rw.drivers.Map()
	completed := make(map[string]bool, len(driversCopy))
	for i := int64(0); ; i++ {
		done := true
		for taskName, d := range driversCopy {
			if !completed[taskName] {
				complete, err := rw.checkApply(ctx, d, false, true)
				if err != nil {
					return err
				}
				completed[taskName] = complete
				if !complete && done {
					done = false
				}
			}
		}
		rw.logDepSize(50, i)
		if done {
			rw.logger.Info("all tasks completed once")
			return nil
		}

		select {
		case err := <-rw.watcher.WaitCh(ctx):
			if err != nil {
				rw.logger.Error("error watching template dependencies", "error", err)
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// EnableTestMode is a helper for testing which tasks were triggered and
// executed. Callers of this method must consume from TaskNotify channel to
// prevent the buffered channel from filling and causing a dead lock.
func (rw *ReadWrite) EnableTestMode() <-chan string {
	tasks := rw.state.GetAllTasks()
	rw.taskNotify = make(chan string, tasks.Len())
	return rw.taskNotify
}
