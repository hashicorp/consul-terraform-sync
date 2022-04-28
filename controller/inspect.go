package controller

import (
	"context"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/state"
)

var (
	_ Controller = (*Inspect)(nil)

	// MuteInspectController is used to toggle muting the InspectController
	// from forcing Terraform output, useful for benchmarks
	MuteInspectController bool
)

// Inspect is the controller to run in inspect mode
type Inspect struct {
	logger logging.Logger

	state        state.Store
	tasksManager *TasksManager
}

// NewInspect configures and initializes a new inspect controller
func NewInspect(conf *config.Config) (*Inspect, error) {
	logger := logging.Global().Named(ctrlSystemName)
	logger.Info("setting up controller", "type", "inspect")

	s := state.NewInMemoryStore(conf)

	tm, err := NewTasksManager(conf, s)
	if err != nil {
		return nil, err
	}

	return &Inspect{
		logger:       logger,
		state:        s,
		tasksManager: tm,
	}, nil
}

// Init initializes the controller before it can be run
func (ctrl *Inspect) Init(ctx context.Context) error {
	return ctrl.tasksManager.Init(ctx)
}

func (ctrl *Inspect) Run(ctx context.Context) error {
	ctrl.logger.Info("inspecting all tasks")

	// Stop watching dependencies after inspecting tasks ends
	ctxWatch, cancelWatch := context.WithCancel(ctx)

	// Stop inspecting tasks early if WatchDep errors
	ctxInspect, cancelInspect := context.WithCancel(ctx)

	exitBufLen := 2 // watchDep & once-ing tasks
	exitCh := make(chan error, exitBufLen)

	// start watching dependencies in order to render templates to plan tasks
	go func() {
		exitCh <- ctrl.tasksManager.WatchDep(ctxWatch)
		cancelInspect()
	}()

	// always inspect consecutively to keep inspect logs in order
	go func() {
		exitCh <- ctrl.inspectConsecutive(ctxInspect)
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

func (ctrl *Inspect) inspectConsecutive(ctx context.Context) error {
	tasks := ctrl.state.GetAllTasks()
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			taskName := *task.Name
			ctrl.logger.Info("inspecting task", taskNameLogKey, taskName)
			_, plan, url, err := ctrl.tasksManager.TaskInspect(ctx, *task)
			if err != nil {
				return err
			}

			if !MuteInspectController {
				// output plan to console
				if url != "" {
					ctrl.logger.Info("inspection results", taskNameLogKey,
						taskName, "plan", plan, "url", url)
				} else {
					ctrl.logger.Info("inspection results", taskNameLogKey,
						taskName, "plan", plan)
				}
			}

			ctrl.logger.Info("inspected task", taskNameLogKey, taskName)
		}
	}

	ctrl.logger.Info("all tasks inspected")
	return nil
}

func (ctrl *Inspect) Stop() {
	ctrl.tasksManager.Stop()
}
