package controller

import (
	"context"

	"github.com/hashicorp/consul-terraform-sync/config"
)

var (
	_ Controller = (*ReadOnly)(nil)

	// MuteReadOnlyController is used to toggle muting the ReadOnlyController
	// from forcing Terraform output, useful for benchmarks
	MuteReadOnlyController bool
)

// ReadOnly is the controller to run in read-only mode
type ReadOnly struct {
	tasksManager *TasksManager
}

// NewReadOnly configures and initializes a new ReadOnly controller
func NewReadOnly(conf *config.Config) (*ReadOnly, error) {
	// Run the driver with logging to output the Terraform plan to stdout
	if tfConfig := conf.Driver.Terraform; tfConfig != nil && !MuteReadOnlyController {
		tfConfig.Log = config.Bool(true)
	}

	tm, err := NewTasksManager(conf)
	if err != nil {
		return nil, err
	}

	return &ReadOnly{
		tasksManager: tm,
	}, nil
}

// Init initializes the controller before it can be run
func (ro *ReadOnly) Init(ctx context.Context) error {
	return ro.tasksManager.Init(ctx)
}

func (ro *ReadOnly) Run(ctx context.Context) error {
	return ro.tasksManager.InspectOnce(ctx)
}

func (ro *ReadOnly) Stop() {
	ro.tasksManager.Stop()
}
