package controller

import (
	"context"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/client"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/state"
	"github.com/hashicorp/consul-terraform-sync/templates"
)

var (
	_ Controller = (*ReadWrite)(nil)

	// Number of times to retry attempts
	defaultRetry = 2
)

// ReadWrite is the controller to run in read-write mode
type ReadWrite struct {
	logger logging.Logger

	state        state.Store
	tasksManager *TasksManager
	watcher      templates.Watcher
}

// NewReadWrite configures and initializes a new ReadWrite controller
func NewReadWrite(conf *config.Config) (*ReadWrite, error) {
	logger := logging.Global().Named(ctrlSystemName)

	logger.Info("initializing Consul client and testing connection")
	watcher, err := newWatcher(conf, client.ConsulDefaultMaxRetry)
	if err != nil {
		return nil, err
	}

	state := state.NewInMemoryStore(conf)

	tm, err := NewTasksManager(conf, watcher, state)
	if err != nil {
		return nil, err
	}

	return &ReadWrite{
		logger:       logger,
		state:        state,
		tasksManager: tm,
		watcher:      watcher,
	}, nil
}

// Init initializes the controller before it can be run. Ensures that
// driver is initializes, works are created for each task.
func (rw *ReadWrite) Init(ctx context.Context) error {
	return rw.tasksManager.init(ctx)
}

func (rw *ReadWrite) Run(ctx context.Context) error {
	// Serve API
	conf := rw.tasksManager.state.GetConfig()
	s, err := api.NewAPI(api.Config{
		Controller: rw.tasksManager,
		Port:       config.IntVal(conf.Port),
		TLS:        conf.TLS,
	})
	if err != nil {
		return err
	}

	exitBufLen := 2 // api & run tasks exit
	exitCh := make(chan error, exitBufLen)
	go func() {
		err := s.Serve(ctx)
		exitCh <- err
	}()

	// Run tasks
	go func() {
		err := rw.tasksManager.Run(ctx)
		exitCh <- err
	}()

	counter := 0
	for {
		err := <-exitCh
		counter++
		if err != nil && err != context.Canceled {
			// Exit if an error is returned
			// Not expecting any routines to send a nil error because they run
			// until canceled. Nil check is just to be safe
			return err
		}
		if counter >= exitBufLen {
			// Wait for all contexts to cancel
			return ctx.Err()
		}
	}
}

func (rw *ReadWrite) Once(ctx context.Context) error {
	rw.logger.Info("executing all tasks once through")

	// Stop watching dependencies after once-ing tasks ends
	ctxWatch, cancelWatch := context.WithCancel(ctx)

	// Stop once-ing tasks early if WatchDep errors
	ctxOnce, cancelOnce := context.WithCancel(ctx)

	exitBufLen := 2 // watchDep & once-ing tasks
	exitCh := make(chan error, exitBufLen)

	// start watching dependencies in order to render templates to apply tasks
	go func() {
		exitCh <- rw.tasksManager.WatchDep(ctxWatch)
		cancelOnce()
	}()

	// run consecutively to keep logs in order
	go func() {
		exitCh <- rw.onceConsecutive(ctxOnce)
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

func (rw *ReadWrite) onceConsecutive(ctx context.Context) error {
	tasks := rw.state.GetAllTasks()
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			taskName := *task.Name
			rw.logger.Info("running task once", taskNameLogKey, taskName)
			if _, err := rw.tasksManager.TaskCreateAndRun(ctx, *task); err != nil {
				return err
			}
			rw.logger.Info("task completed", taskNameLogKey, taskName)
		}
	}

	rw.logger.Info("all tasks completed once")
	return nil
}

func (rw *ReadWrite) Stop() {
	rw.tasksManager.Stop()
}

func (rw *ReadWrite) EnableTestMode() <-chan string {
	return rw.tasksManager.EnableTestMode()
}
