package controller

import (
	"context"

	"github.com/hashicorp/consul-terraform-sync/client"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/templates"
)

var (
	_ Controller = (*ReadOnly)(nil)

	// MuteReadOnlyController is used to toggle muting the ReadOnlyController
	// from forcing Terraform output, useful for benchmarks
	MuteReadOnlyController bool
)

// ReadOnly is the controller to run in read-only mode
type ReadOnly struct {
	logger logging.Logger

	tasksManager *TasksManager
	watcher      templates.Watcher
}

// NewReadOnly configures and initializes a new ReadOnly controller
func NewReadOnly(conf *config.Config) (*ReadOnly, error) {
	// Run the driver with logging to output the Terraform plan to stdout
	if tfConfig := conf.Driver.Terraform; tfConfig != nil && !MuteReadOnlyController {
		tfConfig.Log = config.Bool(true)
	}

	logger := logging.Global().Named(ctrlSystemName)

	logger.Info("initializing Consul client and testing connection")
	watcher, err := newWatcher(conf, client.ConsulDefaultMaxRetry)
	if err != nil {
		return nil, err
	}

	tm, err := NewTasksManager(conf, watcher)
	if err != nil {
		return nil, err
	}

	return &ReadOnly{
		logger:       logger,
		tasksManager: tm,
		watcher:      watcher,
	}, nil
}

// Init initializes the controller before it can be run
func (ro *ReadOnly) Init(ctx context.Context) error {
	return ro.tasksManager.init(ctx)
}

func (ro *ReadOnly) Run(ctx context.Context) error {
	return ro.tasksManager.RunInspect(ctx)
}

func (ro *ReadOnly) Stop() {
	ro.tasksManager.Stop()
}
