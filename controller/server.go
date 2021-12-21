package controller

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul-terraform-sync/config"
)

func (rw *ReadWrite) Config() config.Config {
	return *rw.baseController.conf
}

func (rw *ReadWrite) Task(ctx context.Context, taskName string) (config.TaskConfig, error) {
	// TODO handle ctx while waiting for driver lock if it is currently active
	_, ok := rw.drivers.Get(taskName)
	if !ok {
		return config.TaskConfig{}, fmt.Errorf("a task with name '%s' does not exist or has not been initialized yet", taskName)
	}

	// TODO fill in return object
	return config.TaskConfig{}, nil
}

func (rw *ReadWrite) TaskCreate(ctx context.Context, taskConfig config.TaskConfig) error {
	_, err := rw.createTask(ctx, taskConfig)
	return err
}

func (rw *ReadWrite) TaskCreateAndRun(ctx context.Context, taskConfig config.TaskConfig) error {
	d, err := rw.createTask(ctx, taskConfig)
	if err != nil {
		return err
	}

	return rw.runTaskOnce(ctx, d)
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
