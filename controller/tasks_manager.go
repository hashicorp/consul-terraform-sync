package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/retry"
	"github.com/hashicorp/consul-terraform-sync/state/event"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/pkg/errors"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// TasksManager manages the CRUD operations and execution of tasks
type TasksManager struct {
	*baseController

	watcher templates.Watcher

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

// NewTasksManager configures a new tasks manager
func NewTasksManager(conf *config.Config, watcher templates.Watcher) (*TasksManager, error) {
	baseCtrl, err := newBaseController(conf, watcher)
	if err != nil {
		return nil, err
	}

	return &TasksManager{
		baseController:  baseCtrl,
		watcher:         watcher,
		retry:           retry.NewRetry(defaultRetry, time.Now().UnixNano()),
		scheduleStartCh: make(chan driver.Driver, 10), // arbitrarily chosen size
		deleteCh:        make(chan string, 10),        // arbitrarily chosen size
		scheduleStopChs: make(map[string](chan struct{})),
	}, nil
}

// Init initializes a tasks manager
func (tm *TasksManager) Init(ctx context.Context) error {
	return tm.init(ctx)
}

func (tm *TasksManager) Config() config.Config {
	return tm.state.GetConfig()
}

func (tm *TasksManager) Events(ctx context.Context, taskName string) (map[string][]event.Event, error) {
	return tm.state.GetTaskEvents(taskName), nil
}

func (tm *TasksManager) Task(ctx context.Context, taskName string) (config.TaskConfig, error) {
	// TODO handle ctx while waiting for state lock if it is currently active
	conf, ok := tm.state.GetTask(taskName)
	if ok {
		return conf, nil
	}

	return config.TaskConfig{}, fmt.Errorf("a task with name '%s' does not exist or has not been initialized yet", taskName)
}

func (tm *TasksManager) Tasks(ctx context.Context) ([]config.TaskConfig, error) {
	// TODO handle ctx while waiting for state lock if it is currently active
	tasks := tm.state.GetAllTasks()

	// convert config.TaskConfigs => []config.TaskConfig
	taskConfs := make([]config.TaskConfig, len(tasks))
	for ix, taskConf := range tasks {
		taskConfs[ix] = *taskConf
	}

	return taskConfs, nil
}

func (tm *TasksManager) TaskCreate(ctx context.Context, taskConfig config.TaskConfig) (config.TaskConfig, error) {
	d, err := tm.createTask(ctx, taskConfig)
	if err != nil {
		return config.TaskConfig{}, err
	}

	return tm.addTask(ctx, d)
}

func (tm *TasksManager) TaskCreateAndRun(ctx context.Context, taskConfig config.TaskConfig) (config.TaskConfig, error) {
	d, err := tm.createTask(ctx, taskConfig)
	if err != nil {
		return config.TaskConfig{}, err
	}

	if err := tm.runTask(ctx, d); err != nil {
		return config.TaskConfig{}, err
	}

	return tm.addTask(ctx, d)
}

// TaskDelete marks a task for deletion
func (tm *TasksManager) TaskDelete(ctx context.Context, name string) error {
	logger := tm.logger.With(taskNameLogKey, name)
	if tm.drivers.IsMarkedForDeletion(name) {
		logger.Debug("task is already marked for deletion")
		return nil
	}
	tm.drivers.MarkForDeletion(name)
	tm.deleteCh <- name
	logger.Debug("task marked for deletion")
	return nil
}

// TaskInspect creates and inspects a temporary task that is not added to the drivers list.
func (tm *TasksManager) TaskInspect(ctx context.Context, taskConfig config.TaskConfig) (bool, string, string, error) {
	d, err := tm.createTask(ctx, taskConfig)
	if err != nil {
		return false, "", "", err
	}

	plan, err := d.InspectTask(ctx)
	return plan.ChangesPresent, plan.Plan, plan.URL, err
}

func (tm *TasksManager) TaskUpdate(ctx context.Context, updateConf config.TaskConfig, runOp string) (bool, string, string, error) {
	// Only enabled changes are supported at this time
	if updateConf.Enabled == nil {
		return false, "", "", nil
	}
	if updateConf.Name == nil || *updateConf.Name == "" {
		return false, "", "", fmt.Errorf("task name is required for updating a task")
	}

	taskName := *updateConf.Name
	logger := tm.logger.With(taskNameLogKey, taskName)
	logger.Trace("updating task")
	if tm.drivers.IsActive(taskName) {
		return false, "", "", fmt.Errorf("task '%s' is active and cannot be updated at this time", taskName)
	}
	tm.drivers.SetActive(taskName)
	defer tm.drivers.SetInactive(taskName)

	d, ok := tm.drivers.Get(taskName)
	if !ok {
		return false, "", "", fmt.Errorf("task %s does not exist to run", taskName)
	}

	var storedErr error
	if runOp == driver.RunOptionNow {
		task := d.Task()
		ev, err := event.NewEvent(taskName, &event.Config{
			Providers: task.ProviderIDs(),
			Services:  task.ServiceNames(),
			Source:    task.Module(),
		})
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("error creating task update"+
				"event for %q", taskName))
			logger.Error("error creating new event", "error", err)
			return false, "", "", err
		}
		defer func() {
			ev.End(storedErr)
			logger.Trace("adding event", "event", ev.GoString())
			if err := tm.state.AddTaskEvent(*ev); err != nil {
				// only log error since update task occurred successfully by now
				logger.Error("error storing event", "event", ev.GoString(), "error", err)
			}
		}()
		ev.Start()
	}

	if runOp != driver.RunOptionInspect {
		// Only update state if the update is not inspect type
		tm.state.SetTask(updateConf)
	}

	patch := driver.PatchTask{
		RunOption: runOp,
		Enabled:   *updateConf.Enabled,
	}
	var plan driver.InspectPlan
	plan, storedErr = d.UpdateTask(ctx, patch)
	if storedErr != nil {
		logger.Trace("error while updating task", "error", storedErr)
		return false, "", "", storedErr
	}

	return plan.ChangesPresent, plan.Plan, "", nil
}

func configFromDriverTask(t *driver.Task) (config.TaskConfig, error) {
	vars := make(map[string]string)

	// value can be anything so marshal it to equivalent json
	// and store json as the string value in the map
	for k, v := range t.Variables() {
		b, err := ctyjson.Marshal(v, v.Type())
		if err != nil {
			return config.TaskConfig{}, err
		}
		vars[k] = string(b)
	}

	var bpConf config.BufferPeriodConfig
	bp, ok := t.BufferPeriod()
	if ok {
		bpConf = config.BufferPeriodConfig{
			Enabled: config.Bool(true),
			Max:     config.TimeDuration(bp.Max),
			Min:     config.TimeDuration(bp.Min),
		}
	} else {
		bpConf = config.BufferPeriodConfig{
			Enabled: config.Bool(false),
			Max:     config.TimeDuration(0),
			Min:     config.TimeDuration(0),
		}
	}

	inputs := t.ModuleInputs()
	tfcWs := t.TFCWorkspace()

	return config.TaskConfig{
		Description:        config.String(t.Description()),
		Name:               config.String(t.Name()),
		Enabled:            config.Bool(t.IsEnabled()),
		Providers:          t.ProviderIDs(),
		DeprecatedServices: t.ServiceNames(),
		Module:             config.String(t.Module()),
		Variables:          vars, // TODO: omit or safe to return?
		Version:            config.String(t.Version()),
		BufferPeriod:       &bpConf,
		Condition:          t.Condition(),
		ModuleInputs:       &inputs,
		WorkingDir:         config.String(t.WorkingDir()),

		// Enterprise
		DeprecatedTFVersion: config.String(t.DeprecatedTFVersion()),
		TFCWorkspace:        &tfcWs,
	}, nil
}

// addTask handles the necessary steps to add a task for CTS to monitor and run.
// For example: setting buffer period, updating driver list and state, etc
//
// Assumes that the task driver has already been successfully created. On any
// error, the task will be cleaned up. Returns a copy of the added task's
// config
func (tm TasksManager) addTask(ctx context.Context, d driver.Driver) (config.TaskConfig, error) {
	d.SetBufferPeriod()

	name := d.Task().Name()
	if err := tm.drivers.Add(name, d); err != nil {
		tm.cleanupTask(ctx, name)
		return config.TaskConfig{}, err
	}

	conf, err := configFromDriverTask(d.Task())
	if err != nil {
		tm.cleanupTask(ctx, name)
		return config.TaskConfig{}, err
	}

	tm.state.SetTask(conf)

	if d.Task().IsScheduled() {
		tm.scheduleStartCh <- d
	}

	return conf, nil
}

func (tm TasksManager) cleanupTask(ctx context.Context, name string) {
	err := tm.TaskDelete(ctx, name)
	if err != nil {
		tm.logger.Error("unable to cleanup task after error", "task_name", name)
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
func (tm *TasksManager) checkApply(ctx context.Context, d driver.Driver, retry, once bool) (bool, error) {
	task := d.Task()
	taskName := task.Name()
	if !task.IsEnabled() {
		if task.IsScheduled() {
			// Schedule tasks are specifically triggered and logged at INFO.
			// Accompanying disabled log should be at same level
			tm.logger.Info("skipping disabled scheduled task", taskNameLogKey, taskName)
		} else {
			// Dynamic tasks are all triggered together on any dependency
			// change so logs can be noisy
			tm.logger.Trace("skipping disabled task", taskNameLogKey, taskName)
		}
		return true, nil
	}

	// setup to store event information
	ev, err := event.NewEvent(taskName, &event.Config{
		Providers: task.ProviderIDs(),
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
		tm.logger.Trace("adding event", "event", ev.GoString())
		if err := tm.state.AddTaskEvent(*ev); err != nil {
			tm.logger.Error("error storing event", "event", ev.GoString())
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
			tm.logger.Info("scheduled task triggered but had no changes",
				taskNameLogKey, taskName)
			defer storeEvent()
		}
		return rendered, nil
	}

	// rendering a template may take several cycles in order to completely fetch
	// new data
	if rendered {
		tm.logger.Info("executing task", taskNameLogKey, taskName)
		tm.drivers.SetActive(taskName)
		defer tm.drivers.SetInactive(taskName)
		defer storeEvent()

		if retry {
			desc := fmt.Sprintf("ApplyTask %s", taskName)
			storedErr = tm.retry.Do(ctx, d.ApplyTask, desc)
		} else {
			storedErr = d.ApplyTask(ctx)
		}
		if storedErr != nil {
			return false, fmt.Errorf("could not apply changes for task %s: %s",
				taskName, storedErr)
		}

		tm.logger.Info("task completed", taskNameLogKey, taskName)
	}

	return rendered, nil
}

// createTask creates and initializes a singular task from configuration
func (tm *TasksManager) createTask(ctx context.Context, taskConfig config.TaskConfig) (driver.Driver, error) {
	conf := tm.state.GetConfig()
	taskConfig.Finalize(conf.BufferPeriod, *conf.WorkingDir)
	if err := taskConfig.Validate(); err != nil {
		tm.logger.Trace("invalid config to create task", "error", err)
		return nil, err
	}

	taskName := *taskConfig.Name
	logger := tm.logger.With(taskNameLogKey, taskName)

	// Check if task exists, if it does, do not create again
	if _, ok := tm.drivers.Get(taskName); ok {
		logger.Trace("task already exists")
		return nil, fmt.Errorf("task with name %s already exists", taskName)
	}

	d, err := tm.createNewTaskDriver(taskConfig)
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
func (tm *TasksManager) runTask(ctx context.Context, d driver.Driver) error {
	task := d.Task()
	taskName := task.Name()
	logger := tm.logger.With(taskNameLogKey, taskName)
	if !task.IsEnabled() {
		logger.Trace("skipping disabled task")
		return nil
	}

	if tm.drivers.IsMarkedForDeletion(taskName) {
		logger.Trace("task is marked for deletion, skipping")
		return nil
	}

	err := tm.waitForTaskInactive(ctx, taskName)
	if err != nil {
		return err
	}

	tm.drivers.SetActive(taskName)
	defer tm.drivers.SetInactive(taskName)

	// Create new event for task run
	ev, err := event.NewEvent(taskName, &event.Config{
		Providers: task.ProviderIDs(),
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
	if err := tm.state.AddTaskEvent(*ev); err != nil {
		// only log error since creating a task occurred successfully by now
		logger.Error("error storing event", "event", ev.GoString(), "error", err)
	}

	if tm.taskNotify != nil {
		tm.taskNotify <- taskName
	}

	return err
}

// deleteTask deletes a task from the drivers map and deletes the task's events.
// If a task is active and running, it will wait until the task has completed before
// proceeding with the deletion.
func (tm *TasksManager) deleteTask(ctx context.Context, name string) error {
	logger := tm.logger.With(taskNameLogKey, name)

	// Check if task exists
	d, ok := tm.drivers.Get(name)
	if !ok {
		logger.Debug("task does not exist")
		return nil
	}

	err := tm.waitForTaskInactive(ctx, name)
	if err != nil {
		return err
	}

	if d.Task().IsScheduled() {
		// Notify the scheduled task to stop
		stopCh := tm.scheduleStopChs[name]
		if stopCh != nil {
			stopCh <- struct{}{}
		}
		delete(tm.scheduleStopChs, name)
	}

	// Delete task from drivers and event store
	err = tm.drivers.Delete(name)
	if err != nil {
		logger.Error("unable to delete task", "error", err)
		return err
	}

	// Delete task from state only after driver successfully deleted
	tm.state.DeleteTask(name)
	tm.state.DeleteTaskEvents(name)

	logger.Debug("task deleted")
	return nil
}

func (tm *TasksManager) waitForTaskInactive(ctx context.Context, name string) error {
	// Check first if inactive, return early and don't log
	if !tm.drivers.IsActive(name) {
		return nil
	}
	// Check continuously in a loop until inactive
	tm.logger.Debug("waiting for task to become inactive", taskNameLogKey, name)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if !tm.drivers.IsActive(name) {
				return nil
			}
			time.Sleep(100 * time.Microsecond)
		}
	}
}
