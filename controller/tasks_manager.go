package controller

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/state/event"
	"github.com/pkg/errors"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// TasksManager manages the CRUD operations and execution of tasks
type TasksManager struct {
	// TODO: placeholder. Will convert these ReadWrite methods to TasksManager
}

func (rw *ReadWrite) Config() config.Config {
	return rw.state.GetConfig()
}

func (rw *ReadWrite) Events(ctx context.Context, taskName string) (map[string][]event.Event, error) {
	return rw.state.GetTaskEvents(taskName), nil
}

func (rw *ReadWrite) Task(ctx context.Context, taskName string) (config.TaskConfig, error) {
	// TODO handle ctx while waiting for state lock if it is currently active
	conf, ok := rw.state.GetTask(taskName)
	if ok {
		return conf, nil
	}

	return config.TaskConfig{}, fmt.Errorf("a task with name '%s' does not exist or has not been initialized yet", taskName)
}

func (rw *ReadWrite) Tasks(ctx context.Context) ([]config.TaskConfig, error) {
	// TODO handle ctx while waiting for state lock if it is currently active
	tasks := rw.state.GetAllTasks()

	// convert config.TaskConfigs => []config.TaskConfig
	taskConfs := make([]config.TaskConfig, len(tasks))
	for ix, taskConf := range tasks {
		taskConfs[ix] = *taskConf
	}

	return taskConfs, nil
}

func (rw *ReadWrite) TaskCreate(ctx context.Context, taskConfig config.TaskConfig) (config.TaskConfig, error) {
	d, err := rw.createTask(ctx, taskConfig)
	if err != nil {
		return config.TaskConfig{}, err
	}

	return rw.addTask(ctx, d)
}

func (rw *ReadWrite) TaskCreateAndRun(ctx context.Context, taskConfig config.TaskConfig) (config.TaskConfig, error) {
	d, err := rw.createTask(ctx, taskConfig)
	if err != nil {
		return config.TaskConfig{}, err
	}

	if err := rw.runTask(ctx, d); err != nil {
		return config.TaskConfig{}, err
	}

	return rw.addTask(ctx, d)
}

// TaskDelete marks a task for deletion
func (rw *ReadWrite) TaskDelete(ctx context.Context, name string) error {
	logger := rw.logger.With(taskNameLogKey, name)
	if rw.drivers.IsMarkedForDeletion(name) {
		logger.Debug("task is already marked for deletion")
		return nil
	}
	rw.drivers.MarkForDeletion(name)
	rw.deleteCh <- name
	logger.Debug("task marked for deletion")
	return nil
}

// TaskInspect creates and inspects a temporary task that is not added to the drivers list.
func (rw *ReadWrite) TaskInspect(ctx context.Context, taskConfig config.TaskConfig) (bool, string, string, error) {
	d, err := rw.createTask(ctx, taskConfig)
	if err != nil {
		return false, "", "", err
	}

	plan, err := d.InspectTask(ctx)
	return plan.ChangesPresent, plan.Plan, plan.URL, err
}

func (rw *ReadWrite) TaskUpdate(ctx context.Context, updateConf config.TaskConfig, runOp string) (bool, string, string, error) {
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
	if rw.drivers.IsActive(taskName) {
		return false, "", "", fmt.Errorf("task '%s' is active and cannot be updated at this time", taskName)
	}
	rw.drivers.SetActive(taskName)
	defer rw.drivers.SetInactive(taskName)

	d, ok := rw.drivers.Get(taskName)
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
			if err := rw.state.AddTaskEvent(*ev); err != nil {
				// only log error since update task occurred successfully by now
				logger.Error("error storing event", "event", ev.GoString(), "error", err)
			}
		}()
		ev.Start()
	}

	if runOp != driver.RunOptionInspect {
		// Only update state if the update is not inspect type
		rw.state.SetTask(updateConf)
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
func (rw ReadWrite) addTask(ctx context.Context, d driver.Driver) (config.TaskConfig, error) {
	d.SetBufferPeriod()

	name := d.Task().Name()
	if err := rw.drivers.Add(name, d); err != nil {
		rw.cleanupTask(ctx, name)
		return config.TaskConfig{}, err
	}

	conf, err := configFromDriverTask(d.Task())
	if err != nil {
		rw.cleanupTask(ctx, name)
		return config.TaskConfig{}, err
	}

	rw.state.SetTask(conf)

	if d.Task().IsScheduled() {
		rw.scheduleStartCh <- d
	}

	return conf, nil
}

func (rw ReadWrite) cleanupTask(ctx context.Context, name string) {
	err := rw.TaskDelete(ctx, name)
	if err != nil {
		rw.logger.Error("unable to cleanup task after error", "task_name", name)
	}
}
