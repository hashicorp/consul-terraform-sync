package controller

import (
	"context"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/state"
)

var (
	_ Controller = (*Once)(nil)
)

// Once is the controller to run in once mode
type Once struct {
	logger logging.Logger

	state        state.Store
	tasksManager *TasksManager
}

// NewOnce configures and initializes a new Once controller
func NewOnce(conf *config.Config) (*Once, error) {
	logger := logging.Global().Named(ctrlSystemName)
	logger.Info("setting up controller", "type", "once")

	state := state.NewInMemoryStore(conf)

	tm, err := NewTasksManager(conf, state)
	if err != nil {
		return nil, err
	}

	return &Once{
		logger:       logger,
		state:        state,
		tasksManager: tm,
	}, nil
}

// Init initializes the controller before it can be run.
func (ctrl *Once) Init(ctx context.Context) error {
	return ctrl.tasksManager.Init(ctx)
}

func (ctrl *Once) Run(ctx context.Context) error {
	ctrl.logger.Info("executing all tasks once through")

	// Stop watching dependencies after once-ing tasks ends
	ctxWatch, cancelWatch := context.WithCancel(ctx)

	// Stop once-ing tasks early if WatchDep errors
	ctxOnce, cancelOnce := context.WithCancel(ctx)

	exitBufLen := 2 // watchDep & once-ing tasks
	exitCh := make(chan error, exitBufLen)

	// start watching dependencies in order to render templates to apply tasks
	go func() {
		exitCh <- ctrl.tasksManager.WatchDep(ctxWatch)
		cancelOnce()
	}()

	// run consecutively to keep logs in order
	go func() {
		exitCh <- ctrl.onceConsecutive(ctxOnce)
		cancelWatch()
	}()

	counter := 0
	for {
		err := <-exitCh
		counter++
		if err != nil && err != context.Canceled {
			// Exit if an error is returned
			// Once method sends a nil error on completion
			return err
		}
		if counter >= exitBufLen {
			// Wait for all contexts to cancel
			return ctx.Err()
		}
	}
}

func (ctrl *Once) onceConsecutive(ctx context.Context) error {
	tasks := ctrl.state.GetAllTasks()
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			taskName := *task.Name
			ctrl.logger.Info("running task once", taskNameLogKey, taskName)
			if _, err := ctrl.tasksManager.TaskCreateAndRun(ctx, *task); err != nil {
				return err
			}
			ctrl.logger.Info("task completed", taskNameLogKey, taskName)
		}
	}

	ctrl.logger.Info("all tasks completed once")
	return nil
}

func (ctrl *Once) Stop() {
	ctrl.tasksManager.Stop()
}
