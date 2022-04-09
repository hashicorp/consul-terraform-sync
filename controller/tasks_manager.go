package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/retry"
	"github.com/hashicorp/consul-terraform-sync/state"
	"github.com/hashicorp/consul-terraform-sync/state/event"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/pkg/errors"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// TasksManager manages CRUD operations and running tasks
type TasksManager struct {
	state  state.Store
	logger logging.Logger
	retry  retry.Retry

	driverFactory *driver.Factory
	drivers       *driver.Drivers

	// scheduleStartCh sends the task name of newly created scheduled tasks
	// that will need to be monitored
	scheduleStartCh chan string
}

// NewTasksManager configures a new tasks manager
func NewTasksManager(conf *config.Config, state state.Store, watcher templates.Watcher) (*TasksManager, error) {
	logger := logging.Global().Named(ctrlSystemName)

	// tm.drivers.Reset() TODO: where? here?
	driverFactory, err := driver.NewFactory(conf, watcher)
	if err != nil {
		return nil, err
	}

	return &TasksManager{
		logger:        logger,
		state:         state,
		driverFactory: driverFactory,
		drivers:       driver.NewDrivers(),
		retry:         retry.NewRetry(defaultRetry, time.Now().UnixNano()),

		// Size of channel is an arbitrarily chosen value.
		scheduleStartCh: make(chan string, 10),
	}, nil
}

func (tm *TasksManager) Init(ctx context.Context) error {
	return tm.driverFactory.Init(ctx)
}

func (tm *TasksManager) Config() config.Config {
	return tm.state.GetConfig()
}

func (tm *TasksManager) Events(ctx context.Context, taskName string) (map[string][]event.Event, error) {
	return tm.state.GetTaskEvents(taskName), nil
}

func (tm *TasksManager) Task(ctx context.Context, taskName string) (config.TaskConfig, error) {
	conf, ok := tm.state.GetTask(taskName)
	if ok {
		return conf, nil
	}

	return config.TaskConfig{}, fmt.Errorf("a task with name '%s' does not exist or has not been initialized yet", taskName)
}

func (tm *TasksManager) Tasks(ctx context.Context) ([]config.TaskConfig, error) {
	conf := tm.state.GetConfig()
	taskConfs := make([]config.TaskConfig, len(*conf.Tasks))

	for ix, taskConf := range *conf.Tasks {
		taskConfs[ix] = *taskConf
	}

	return taskConfs, nil
}

func (tm *TasksManager) TaskCreate(ctx context.Context, taskConfig config.TaskConfig) (config.TaskConfig, error) {
	d, err := tm.createTask(ctx, taskConfig)
	if err != nil {
		return config.TaskConfig{}, err
	}

	// Set the buffer period
	d.SetBufferPeriod()

	// Add the driver to task runner only after it is successully created
	return tm.addTask(ctx, d)
}

func (tm *TasksManager) TaskCreateAndRun(ctx context.Context, taskConfig config.TaskConfig) (config.TaskConfig, error) {
	d, err := tm.createTask(ctx, taskConfig)
	if err != nil {
		return config.TaskConfig{}, err
	}

	if err := tm.runNewTask(ctx, d); err != nil {
		return config.TaskConfig{}, err
	}

	d.SetBufferPeriod()

	// Add the driver to task runner only after it is successully created
	return tm.addTask(ctx, d)
}

// TaskDelete deletes a task from the state store
func (tm *TasksManager) TaskDelete(ctx context.Context, name string) error {
	return tm.deleteTask(ctx, name)
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

func (rw *TasksManager) TaskUpdate(ctx context.Context, updateConf config.TaskConfig, runOp string) (bool, string, string, error) {
	// Only enabled changes are supported at this time
	if updateConf.Enabled == nil {
		return false, "", "", nil
	}
	if updateConf.Name == nil || *updateConf.Name == "" {
		return false, "", "", fmt.Errorf("task name is required for updating a task")
	}

	taskName := *updateConf.Name
	logger := rw.logger.With(taskNameLogKey, taskName)
	logger.Trace("updating task")

	d, ok := rw.drivers.Get(taskName)
	if !ok {
		return false, "", "", fmt.Errorf("task %s does not exist to run", taskName)
	}

	if rw.drivers.IsActive(taskName) {
		return false, "", "", fmt.Errorf("task '%s' is active and cannot be updated at this time", taskName)
	}
	rw.drivers.SetActive(taskName)
	defer rw.drivers.SetInactive(taskName)

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
			if err := rw.state.AddTaskEvent(*ev); err != nil {
				// only log error since update task occurred successfully by now
				logger.Error("error storing event", "event", ev.GoString(), "error", err)
			}
		}()
		ev.Start()
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

	// Add to state
	rw.state.SetTask(updateConf)

	return plan.ChangesPresent, plan.Plan, "", nil
}

// TaskRunNow forces an existing task to run now with a retry. Assumes the task
// has already been created through TaskCreate or TaskCreateAndRun
func (rw *TasksManager) TaskRunNow(ctx context.Context, taskName string) (bool, error) {
	logger := rw.logger.With(taskNameLogKey, taskName)

	d, ok := rw.drivers.Get(taskName)
	if !ok {
		return false, fmt.Errorf("task %s does not exist to run", taskName)
	}

	task := d.Task()
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

	logger.Trace("running task")

	// for scheduled tasks, do not wait if task is active
	if rw.drivers.IsActive(taskName) && task.IsScheduled() {
		return false, fmt.Errorf("task '%s' is active and cannot be run at this time", taskName)
	}

	// for dynamic tasks, wait to see if the task will become inactive
	if err := rw.drivers.WaitForInactive(ctx, taskName); err != nil {
		return false, err
	}

	rw.drivers.SetActive(taskName)
	defer rw.drivers.SetInactive(taskName)

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
		logger.Trace("adding event", "event", ev.GoString())
		if err := rw.state.AddTaskEvent(*ev); err != nil {
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

	if !rendered {
		if task.IsScheduled() {
			// We sometimes want to store an event when a scheduled task did not
			// render i.e. the task ran on schedule but there were no
			// dependency changes so the template did not re-render
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
		defer storeEvent()

		desc := fmt.Sprintf("ApplyTask %s", taskName)
		storedErr = rw.retry.Do(ctx, d.ApplyTask, desc)
		if storedErr != nil {
			return false, fmt.Errorf("could not apply changes for task %s: %s",
				taskName, storedErr)
		}

		rw.logger.Info("task completed", taskNameLogKey, taskName)
	}

	return rendered, nil
}

func (tm TasksManager) TaskByTemplate(tmplID string) (string, bool) {
	driver, ok := tm.drivers.GetTaskByTemplate(tmplID)
	if !ok {
		return "", false
	}
	return driver.Task().Name(), true
}

func (tm TasksManager) WatchNewScheduleTaskCh() <-chan string {
	return tm.scheduleStartCh
}

func (tm TasksManager) cleanupTask(ctx context.Context, name string) {
	if err := tm.deleteTask(ctx, name); err != nil {
		tm.logger.Error("unable to cleanup task after error", "task_name", name)
	}
}

// Add task to task manager. We want to keep this task
func (tm *TasksManager) addTask(ctx context.Context, d driver.Driver) (config.TaskConfig, error) {
	taskName := d.Task().Name()
	err := tm.drivers.Add(taskName, d)
	if err != nil {
		tm.cleanupTask(ctx, taskName)
		return config.TaskConfig{}, err
	}

	conf, err := configFromDriverTask(d.Task())
	if err != nil {
		tm.cleanupTask(ctx, taskName)
		return config.TaskConfig{}, err
	}

	// Add task to state
	tm.state.SetTask(conf)

	if d.Task().IsScheduled() {
		tm.scheduleStartCh <- taskName
	}

	return conf, nil
}

// createTask creates and initializes a singular task from configuration
// also renders template
func (tm *TasksManager) createTask(ctx context.Context, taskConfig config.TaskConfig) (driver.Driver, error) {
	conf := tm.state.GetConfig()
	taskConfig.Finalize(conf.BufferPeriod, *conf.WorkingDir)
	if err := taskConfig.Validate(); err != nil {
		tm.logger.Trace("invalid config to create task", "error", err)
		return nil, err
	}

	taskName := *taskConfig.Name
	logger := tm.logger.With(taskNameLogKey, taskName)

	d, err := tm.driverFactory.Make(ctx, &conf, &taskConfig)
	if err != nil {
		logger.Error("error creating new task driver", "error", err)
		return nil, err
	}

	// and render template
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

// runNewTask force runs a new task that has not been added to the list of drivers
func (tm *TasksManager) runNewTask(ctx context.Context, d driver.Driver) error {
	task := d.Task()
	taskName := task.Name()
	logger := tm.logger.With(taskNameLogKey, taskName)
	if !task.IsEnabled() {
		logger.Trace("skipping disabled task")
		return nil
	}

	logger.Info("executing new task")

	ev, err := event.NewEvent(taskName, &event.Config{
		Providers: task.ProviderIDs(),
		Services:  task.ServiceNames(),
		Source:    task.Module(),
	})
	if err != nil {
		return fmt.Errorf("error creating event for task %s: %s",
			taskName, err)
	}
	var storedErr error
	storeEvent := func() {
		ev.End(storedErr)
		logger.Trace("adding event", "event", ev.GoString())
		if err := tm.state.AddTaskEvent(*ev); err != nil {
			tm.logger.Error("error storing event", "event", ev.GoString())
		}
	}
	defer storeEvent()
	ev.Start()

	// Apply task
	if storedErr = d.ApplyTask(ctx); storedErr != nil {
		logger.Error("error applying task", "error", storedErr)
		return err
	}

	logger.Info("task completed")
	return nil
}

// deleteTask deletes the task from the state. The task runner will handle deleting
// the task when the task is not running any more
func (tm *TasksManager) deleteTask(ctx context.Context, name string) error {
	logger := tm.logger.With(taskNameLogKey, name)

	// Check if task exists
	_, ok := tm.state.GetTask(name)
	if !ok {
		logger.Debug("task does not exist. no need to delete")
		return nil
	}

	tm.state.DeleteTask(name)
	tm.state.DeleteTaskEvents(name)

	logger.Debug("task deleted")
	return nil
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
		TFVersion:    config.String(t.TFVersion()),
		TFCWorkspace: &tfcWs,
	}, nil
}
