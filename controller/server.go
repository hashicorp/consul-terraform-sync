package controller

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
)

func (rw *ReadWrite) Config() config.Config {
	return *rw.baseController.conf
}

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
	return configFromDriverTask(d.Task()), nil
}

func (rw *ReadWrite) TaskCreateAndRun(ctx context.Context, taskConfig config.TaskConfig) (config.TaskConfig, error) {
	d, err := rw.createTask(ctx, taskConfig)
	if err != nil {
		return config.TaskConfig{}, err
	}

	if err := rw.runTaskOnce(ctx, d); err != nil {
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
		Source:       config.String(t.Source()),
		Variables:    vars, // TODO: omit or safe to return?
		Version:      config.String(t.Version()),
		BufferPeriod: &bpConf,
		Condition:    t.Condition(),
		SourceInput:  t.SourceInput(),
		WorkingDir:   config.String(t.WorkingDir()),
	}
}
