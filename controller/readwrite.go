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
		baseController: baseCtrl,
		store:          event.NewStore(),
		retry:          retry.NewRetry(defaultRetry, time.Now().UnixNano()),
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
			go rw.runScheduledTask(ctx, d)
		}
	}

	errCh := make(chan error)
	if rw.watcherCh == nil {
		rw.watcherCh = make(chan string, rw.drivers.Len()+2)
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

	if rw.drivers.IsActive(taskName) {
		// The driver is currently active with the task, initiated by an ad-hoc run.
		// There may be updates for other tasks, so we'll continue checking
		rw.logger.Trace("task is active", taskNameLogKey, taskName)
		return nil
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
func (rw *ReadWrite) runScheduledTask(ctx context.Context, d driver.Driver) error {
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

	switch cond := task.Condition().(type) {
	// Services condition (with regex) and catalog services condition have
	// multiple API calls it depends on for updates. This adds an additional
	// delay for hcat to process any new updates in the background that may be
	// related to this task. 1 second is  used to account for Consul cluster
	// propagation of the change at scale.
	// https://www.hashicorp.com/blog/hashicorp-consul-global-scale-benchmark
	case *config.ServicesConditionConfig:
		if len(cond.Names) == 0 {
			<-time.After(time.Second)
		}
	case *config.CatalogServicesConditionConfig:
		<-time.After(time.Second)
	}

	// setup to store event information
	ev, err := event.NewEvent(taskName, &event.Config{
		Providers: task.ProviderNames(),
		Services:  task.ServiceNames(),
		Source:    task.Source(),
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
		return nil, fmt.Errorf("error creating new task driver: %v", err)
	}

	// Initialize the new task
	err = d.InitTask(ctx)
	if err != nil {
		return nil, fmt.Errorf("error initializing new task, %s", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		ok, err := d.RenderTemplate(ctx)
		if err != nil {
			logger.Error("error rendering task template")
			return nil, fmt.Errorf("error rendering template for task '%s': %s", taskName, err)
		}
		if ok {
			// Once template rendering is finished, return
			return d, nil
		}
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

	rw.drivers.SetActive(taskName)
	defer rw.drivers.SetInactive(taskName)

	switch cond := task.Condition().(type) {
	// Services condition (with regex) and catalog services condition have
	// multiple API calls it depends on for updates. This adds an additional
	// delay for hcat to process any new updates in the background that may be
	// related to this task. 1 second is  used to account for Consul cluster
	// propagation of the change at scale.
	// https://www.hashicorp.com/blog/hashicorp-consul-global-scale-benchmark
	case *config.ServicesConditionConfig:
		if len(cond.Names) == 0 {
			<-time.After(time.Second)
		}
	case *config.CatalogServicesConditionConfig:
		<-time.After(time.Second)
	}

	// Create new event for task run
	ev, err := event.NewEvent(taskName, &event.Config{
		Providers: task.ProviderNames(),
		Services:  task.ServiceNames(),
		Source:    task.Source(),
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

// EnableTestMode is a helper for testing which tasks were triggered and
// executed. Callers of this method must consume from TaskNotify channel to
// prevent the buffered channel from filling and causing a dead lock.
func (rw *ReadWrite) EnableTestMode() <-chan string {
	rw.taskNotify = make(chan string, rw.drivers.Len())
	return rw.taskNotify
}
