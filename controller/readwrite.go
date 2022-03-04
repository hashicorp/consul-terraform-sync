package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	"github.com/hashicorp/consul-terraform-sync/retry"
	"github.com/hashicorp/cronexpr"
)

var (
	_ Controller = (*ReadWrite)(nil)

	// Number of times to retry attempts
	defaultRetry uint = 2
)

// ReadWrite is the controller to run in read-write mode
type ReadWrite struct {
	*baseController
	store *event.Store
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
		store:           event.NewStore(),
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

// checkApply runs a task by attempting to render the template and applying the
// task as necessary.
//
// An event is stored:
//  1. whenever a task errors while executing
//  2. when a dynamic task successfully executes
//  3. when a scheduled task successfully executes (regardless of if there
//     were dependency changes)
//
// Note on #2: no event is stored when a dynamic task renders but does not apply.
// This can occur because driver.RenderTemplate() may need to be called multiple
// times before a template is ready to be applied.
func (rw *ReadWrite) checkApply(ctx context.Context, d driver.Driver, retry, once bool) (bool, error) {
	task := d.Task()
	taskName := task.Name()
	if !task.IsEnabled() {
		if task.IsScheduled() {
			// Schedule tasks are specifically triggered and logged at INFO.
			// Accompanying disabled log should be at same level
			rw.logger.Info("skipping disabled scheduled task", taskNameLogKey, taskName)
		} else {
			// Dynamic tasks are all triggered together on any dependency
			// change so logs can be noisy
			rw.logger.Trace("skipping disabled task", taskNameLogKey, taskName)
		}
		return true, nil
	}

	// setup to store event information
	ev, err := event.NewEvent(taskName, &event.Config{
		Providers: task.ProviderNames(),
		Services:  task.ServiceNames(),
		Source:    task.Module(),
	})
	if err != nil {
		return false, fmt.Errorf("error creating event for task %s: %s",
			taskName, err)
	}
	var storedErr error
	storeEvent := func() {
		ev.End(storedErr)
		rw.logger.Trace("adding event", "event", ev.GoString())
		if err := rw.store.Add(*ev); err != nil {
			rw.logger.Error("error storing event", "event", ev.GoString())
		}
	}
	ev.Start()

	var rendered bool
	rendered, storedErr = d.RenderTemplate(ctx)
	if storedErr != nil {
		defer storeEvent()
		return false, fmt.Errorf("error rendering template for task %s: %s",
			taskName, storedErr)
	}

	if !rendered && !once {
		if task.IsScheduled() {
			// We sometimes want to store an event when a scheduled task did not
			// render i.e. the task ran on schedule but there were no
			// dependency changes so the template did not re-render
			//
			// During once-mode though, a task may not have rendered because it
			// may take multiple calls to fully render. check for once-mode to
			// avoid extra logs/events while the template is finishing rendering.
			rw.logger.Info("scheduled task triggered but had no changes",
				taskNameLogKey, taskName)
			defer storeEvent()
		}
		return rendered, nil
	}

	// rendering a template may take several cycles in order to completely fetch
	// new data
	if rendered {
		rw.logger.Info("executing task", taskNameLogKey, taskName)
		rw.drivers.SetActive(taskName)
		defer rw.drivers.SetInactive(taskName)
		defer storeEvent()

		if retry {
			desc := fmt.Sprintf("ApplyTask %s", taskName)
			storedErr = rw.retry.Do(ctx, d.ApplyTask, desc)
		} else {
			storedErr = d.ApplyTask(ctx)
		}
		if storedErr != nil {
			return false, fmt.Errorf("could not apply changes for task %s: %s",
				taskName, storedErr)
		}

		rw.logger.Info("task completed", taskNameLogKey, taskName)
	}

	return rendered, nil
}

// createTask creates and initializes a singular task from configuration
func (rw *ReadWrite) createTask(ctx context.Context, taskConfig config.TaskConfig) (driver.Driver, error) {
	taskConfig.Finalize(rw.conf.BufferPeriod, *rw.conf.WorkingDir)
	if err := taskConfig.Validate(); err != nil {
		rw.logger.Trace("invalid config to create task", "error", err)
		return nil, err
	}

	taskName := *taskConfig.Name
	logger := rw.logger.With(taskNameLogKey, taskName)

	// Check if task exists, if it does, do not create again
	if _, ok := rw.drivers.Get(taskName); ok {
		logger.Trace("task already exists")
		return nil, fmt.Errorf("task with name %s already exists", taskName)
	}

	d, err := rw.createNewTaskDriver(taskConfig)
	if err != nil {
		logger.Error("error creating new task driver", "error", err)
		return nil, err
	}

	// Initialize the new task
	err = d.InitTask(ctx)
	if err != nil {
		logger.Error("error initializing new task", "error", err)
		// Cleanup the task
		d.DestroyTask(ctx)
		logger.Debug("task destroyed", "task_name", *taskConfig.Name)
		return nil, err
	}

	csTimeout := time.After(30 * time.Second)
	timeout := time.After(1 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-csTimeout:
			// Short-term solution to issue when Create Task API with
			// catalog-service condition edge-case would hang
			logger.Debug("catalog-services condition hanging needs to be overriden")
			d.OverrideNotifier()
		case <-timeout:
			logger.Error("timed out rendering template")
			// Cleanup the task
			d.DestroyTask(ctx)
			logger.Debug("task destroyed", "task_name", *taskConfig.Name)
			return nil, fmt.Errorf("error initializing task")
		default:
		}
		ok, err := d.RenderTemplate(ctx)
		if err != nil {
			logger.Error("error rendering task template")
			// Cleanup the task
			d.DestroyTask(ctx)
			logger.Debug("task destroyed", "task_name", *taskConfig.Name)
			return nil, err
		}
		if ok {
			// Short-term solution to issue when Create Task API with edge-case
			// could cause an extra trigger
			logger.Debug("once-mode extra trigger edge-case needs to be prevented")
			d.OverrideNotifier()

			// Once template rendering is finished, return
			return d, nil
		}
		time.Sleep(50 * time.Millisecond) // waiting because cannot block on a dependency change
	}
}

// runTask will set the driver to active, apply it, and store a run event.
// This method will run the task as-is with current values of templates that
// have already been resolved and rendered. This does not handle any templating.
func (rw *ReadWrite) runTask(ctx context.Context, d driver.Driver) error {
	task := d.Task()
	taskName := task.Name()
	logger := rw.logger.With(taskNameLogKey, taskName)
	if !task.IsEnabled() {
		logger.Trace("skipping disabled task")
		return nil
	}

	if rw.drivers.IsMarkedForDeletion(taskName) {
		logger.Trace("task is marked for deletion, skipping")
		return nil
	}

	err := rw.waitForTaskInactive(ctx, taskName)
	if err != nil {
		return err
	}

	rw.drivers.SetActive(taskName)
	defer rw.drivers.SetInactive(taskName)

	// Create new event for task run
	ev, err := event.NewEvent(taskName, &event.Config{
		Providers: task.ProviderNames(),
		Services:  task.ServiceNames(),
		Source:    task.Module(),
	})
	if err != nil {
		logger.Error("error initializing run task event", "error", err)
		return err
	}
	ev.Start()

	// Apply task
	err = d.ApplyTask(ctx)
	if err != nil {
		logger.Error("error applying task", "error", err)
		return err
	}

	// Store event if apply was successful and task will be created
	ev.End(err)
	logger.Trace("adding event", "event", ev.GoString())
	if err := rw.store.Add(*ev); err != nil {
		// only log error since creating a task occurred successfully by now
		logger.Error("error storing event", "event", ev.GoString(), "error", err)
	}

	if rw.taskNotify != nil {
		rw.taskNotify <- taskName
	}

	return err
}

// deleteTask deletes a task from the drivers map and deletes the task's events.
// If a task is active and running, it will wait until the task has completed before
// proceeding with the deletion.
func (rw *ReadWrite) deleteTask(ctx context.Context, name string) error {
	logger := rw.logger.With(taskNameLogKey, name)

	// Check if task exists
	driver, ok := rw.drivers.Get(name)
	if !ok {
		logger.Debug("task does not exist")
		return nil
	}

	err := rw.waitForTaskInactive(ctx, name)
	if err != nil {
		return err
	}

	if driver.Task().IsScheduled() {
		// Notify the scheduled task to stop
		stopCh := rw.scheduleStopChs[name]
		if stopCh != nil {
			stopCh <- struct{}{}
		}
		delete(rw.scheduleStopChs, name)
	}

	// Delete task from drivers and event store
	err = rw.drivers.Delete(name)
	if err != nil {
		logger.Error("unable to delete task", "error", err)
		return err
	}
	rw.store.Delete(name)
	logger.Debug("task deleted")
	return nil
}

func (rw *ReadWrite) waitForTaskInactive(ctx context.Context, name string) error {
	// Check first if inactive, return early and don't log
	if !rw.drivers.IsActive(name) {
		return nil
	}
	// Check continuously in a loop until inactive
	rw.logger.Debug("waiting for task to become inactive", taskNameLogKey, name)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if !rw.drivers.IsActive(name) {
				return nil
			}
			time.Sleep(100 * time.Microsecond)
		}
	}
}

// EnableTestMode is a helper for testing which tasks were triggered and
// executed. Callers of this method must consume from TaskNotify channel to
// prevent the buffered channel from filling and causing a dead lock.
func (rw *ReadWrite) EnableTestMode() <-chan string {
	rw.taskNotify = make(chan string, rw.drivers.Len())
	return rw.taskNotify
}
