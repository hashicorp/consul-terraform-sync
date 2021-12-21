package controller

import (
	"context"

	"github.com/hashicorp/consul-terraform-sync/config"
)

func (rw *ReadWrite) Config() config.Config {
	return *rw.baseController.conf
}

func (rw *ReadWrite) Task(ctx context.Context, taskName string) (config.TaskConfig, error) {
	return config.TaskConfig{}, nil
}

func (rw *ReadWrite) TaskCreate(ctx context.Context, taskConfig config.TaskConfig) error {
	return nil
}

func (rw *ReadWrite) TaskCreateAndRun(ctx context.Context, taskConfig config.TaskConfig) error {
	return nil
}

func (rw *ReadWrite) TaskDelete(ctx context.Context, name string) error {
	return nil
}
