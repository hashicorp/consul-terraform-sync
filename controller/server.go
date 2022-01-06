package controller

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
)

func (rw *ReadWrite) Config() config.Config {
	return *rw.baseController.conf
}

// TODO: remove getter functions when status and task API handlers are refactored
func (rw *ReadWrite) Drivers() *driver.Drivers {
	return rw.drivers
}

func (rw *ReadWrite) Store() *event.Store {
	return rw.store
}

// End TODO

func (rw *ReadWrite) Task(ctx context.Context, taskName string) (config.TaskConfig, error) {
	// TODO handle ctx while waiting for driver lock if it is currently active
	d, ok := rw.drivers.Get(taskName)
	if !ok {
		return config.TaskConfig{}, fmt.Errorf("a task with name '%s' does not exist or has not been initialized yet", taskName)
	}

	return configFromDriverTask(d.Task()), nil
}

func (rw *ReadWrite) TaskCreate(ctx context.Context, taskConfig config.TaskConfig) (config.TaskConfig, error) {
	d, err := rw.createTask(ctx, taskConfig)
	if err != nil {
		return config.TaskConfig{}, err
	}

	// TODO: Set the buffer period
	// d.SetBufferPeriod()

	// Add the task driver to the driver list only after successful create
	err = rw.drivers.Add(*taskConfig.Name, d)
	if err != nil {
		return config.TaskConfig{}, err
	}
	return configFromDriverTask(d.Task()), nil
}

func (rw *ReadWrite) TaskCreateAndRun(ctx context.Context, taskConfig config.TaskConfig) (config.TaskConfig, error) {
	d, err := rw.createTask(ctx, taskConfig)
	if err != nil {
		return config.TaskConfig{}, err
	}

	if err := rw.runTask(ctx, d); err != nil {
		return config.TaskConfig{}, err
	}

	// TODO: Set the buffer period
	// d.SetBufferPeriod()

	// Add the task driver to the driver list only after successful create and run
	err = rw.drivers.Add(*taskConfig.Name, d)
	if err != nil {
		return config.TaskConfig{}, err
	}
	return configFromDriverTask(d.Task()), nil
}

func (rw *ReadWrite) TaskDelete(ctx context.Context, name string) error {
	logger := rw.logger.With(taskNameLogKey, name)
	logger.Trace("deleting task")

	// Check if task is active
	if rw.drivers.IsActive(name) {
		return fmt.Errorf("task '%s' is currently running and cannot be deleted "+
			"at this time", name)
	}

	// Delete task driver and events
	err := rw.drivers.Delete(name)
	if err != nil {
		logger.Trace("unable to delete task", "error", err)
		return err
	}

	rw.store.Delete(name)
	logger.Trace("task deleted")
	return nil
}

// TaskInspect creates and inspects a temporary task that is not added to the drivers list.
func (rw *ReadWrite) TaskInspect(ctx context.Context, taskConfig config.TaskConfig) (bool, string, string, error) {
	d, err := rw.createTask(ctx, taskConfig)
	if err != nil {
		return false, "", "", err
	}

	plan, err := d.InspectTask(ctx)
	return plan.ChangesPresent, plan.Plan, "", err
}

func configFromDriverTask(t *driver.Task) config.TaskConfig {
	vars := make(map[string]string)
	for k, v := range t.Variables() {
		vars[k] = v.AsString()
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

	return config.TaskConfig{
		Description:  config.String(t.Description()),
		Name:         config.String(t.Name()),
		Enabled:      config.Bool(t.IsEnabled()),
		Providers:    t.ProviderNames(),
		Services:     t.ServiceNames(),
		Module:       config.String(t.Source()),
		Variables:    vars, // TODO: omit or safe to return?
		Version:      config.String(t.Version()),
		BufferPeriod: &bpConf,
		Condition:    t.Condition(),
		SourceInput:  t.SourceInput(),
		WorkingDir:   config.String(t.WorkingDir()),
	}
}
