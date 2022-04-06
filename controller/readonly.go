package controller

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/state"
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

	state        state.Store
	watcher      templates.Watcher
	tasksManager *TasksManager
}

// NewReadOnly configures and initializes a new ReadOnly controller
func NewReadOnly(conf *config.Config) (*ReadOnly, error) {
	logger := logging.Global().Named(ctrlSystemName)

	state := state.NewInMemoryStore(conf)

	watcher, err := newWatcher(conf)
	if err != nil {
		return nil, err
	}

	tm, err := NewTasksManager(conf, state, watcher)
	if err != nil {
		return nil, err
	}

	return &ReadOnly{
		logger:       logger,
		watcher:      watcher,
		state:        state,
		tasksManager: tm,
	}, nil
}

// Init initializes the controller before it can be run
// TODO: potentially remove Init
func (ro *ReadOnly) Init(ctx context.Context) error {
	return nil
}

func (ro *ReadOnly) Run(ctx context.Context) error {
	ro.logger.Info("running tasks once")

	// start watcher in order to render templates for inspected tasks
	ctxWatcher, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		for {
			ro.logger.Trace("starting inspect-mode template dependency monitoring")
			err := <-ro.watcher.WaitCh(ctxWatcher)
			if err == context.Canceled {
				ro.logger.Info("stopping inspect-mode dependency monitoring")
				return
			}
			if err != nil {
				ro.logger.Error("error monitoring inspect-mode template dependencies", "error", err)
				return
			}
		}
	}()

	tasks := ro.state.GetAllTasks()
	for ix, task := range tasks {
		ro.logger.Info("inspecting task once", taskNameLogKey, *task.Name)
		_, plan, _, err := ro.tasksManager.TaskInspect(ctx, *task)
		if err != nil {
			return err
		}

		if !MuteReadOnlyController {
			// output plan to console
			fmt.Println(plan)
		}
		logDepSize(ro.watcher, ro.logger, 50, int64(ix))
	}

	return nil
}

func (ro *ReadOnly) Stop() {
	ro.watcher.Stop()
}
