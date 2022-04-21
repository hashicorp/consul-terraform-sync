package controller

import (
	"context"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/state"
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
	tasksManager *TasksManager
}

// NewReadOnly configures and initializes a new ReadOnly controller
func NewReadOnly(conf *config.Config) (*ReadOnly, error) {
	logger := logging.Global().Named(ctrlSystemName)

	state := state.NewInMemoryStore(conf)

	tm, err := NewTasksManager(conf, state)
	if err != nil {
		return nil, err
	}

	return &ReadOnly{
		logger:       logger,
		state:        state,
		tasksManager: tm,
	}, nil
}

// Init initializes the controller before it can be run
func (ro *ReadOnly) Init(ctx context.Context) error {
	return ro.tasksManager.Init(ctx)
}

func (ro *ReadOnly) Run(ctx context.Context) error {
	ro.logger.Info("inspecting all tasks")

	// Stop watching dependencies after inspecting tasks ends
	ctxWatch, cancelWatch := context.WithCancel(ctx)

	// Stop inspecting tasks early if WatchDep errors
	ctxInspect, cancelInspect := context.WithCancel(ctx)

	exitBufLen := 2 // watchDep & once-ing tasks
	exitCh := make(chan error, exitBufLen)

	// start watching dependencies in order to render templates to plan tasks
	go func() {
		exitCh <- ro.tasksManager.WatchDep(ctxWatch)
		cancelInspect()
	}()

	// always inspect consecutively to keep inspect logs in order
	go func() {
		exitCh <- ro.inspectConsecutive(ctxInspect)
		cancelWatch()
	}()

	counter := 0
	for {
		err := <-exitCh
		counter++
		if err != nil && err != context.Canceled {
			// Exit if an error is returned
			// inspectConsecutive sends a nil error on completion
			return err
		}
		if counter >= exitBufLen {
			// Wait for all contexts to cancel
			return ctx.Err()
		}
	}
}

func (ro *ReadOnly) inspectConsecutive(ctx context.Context) error {
	tasks := ro.state.GetAllTasks()
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			taskName := *task.Name
			ro.logger.Info("inspecting task", taskNameLogKey, taskName)
			_, plan, url, err := ro.tasksManager.TaskInspect(ctx, *task)
			if err != nil {
				return err
			}

			if !MuteReadOnlyController {
				// output plan to console
				if url != "" {
					ro.logger.Info("inspection results", taskNameLogKey,
						taskName, "plan", plan, "url", url)
				} else {
					ro.logger.Info("inspection results", taskNameLogKey,
						taskName, "plan", plan)
				}
			}

			ro.logger.Info("inspected task", taskNameLogKey, taskName)
		}
	}

	ro.logger.Info("all tasks inspected")
	return nil
}

func (ro *ReadOnly) Stop() {
	ro.tasksManager.Stop()
}
