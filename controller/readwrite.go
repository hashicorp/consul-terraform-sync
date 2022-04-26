package controller

import (
	"context"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/state"
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

	// whether or not the tasks have gone through once-mode. intended to be used
	// by benchmarks to run once-mode separately
	once bool
}

// NewReadWrite configures and initializes a new ReadWrite controller
func NewReadWrite(conf *config.Config) (*ReadWrite, error) {
	logger := logging.Global().Named(ctrlSystemName)

	state := state.NewInMemoryStore(conf)

	tm, err := NewTasksManager(conf, state)
	if err != nil {
		return nil, err
	}

	return &ReadWrite{
		logger:       logger,
		state:        state,
		tasksManager: tm,
	}, nil
}

// Init initializes the controller before it can be run. Ensures that
// driver is initializes, works are created for each task.
func (rw *ReadWrite) Init(ctx context.Context) error {
	return rw.tasksManager.Init(ctx)
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

	// Run tasks once through once-mode
	if !rw.once {
		if err := rw.Once(ctx); err != nil {
			return err
		}
	}

	// Run tasks in long-running mode
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

// Once runs the tasks once. Intended to only be called by Run() or outside of
// Run() for the case of benchmarks
func (rw *ReadWrite) Once(ctx context.Context) error {
	once := Once{
		logger:       rw.logger,
		state:        rw.state,
		tasksManager: rw.tasksManager,
	}

	// no need to init or stop Once controller since it shares tasksManager
	// with ReadWrite controller
	if err := once.Run(ctx); err != nil {
		return err
	}

	rw.once = true
	return nil
}

func (rw *ReadWrite) Stop() {
	rw.tasksManager.Stop()
}

func (rw *ReadWrite) EnableTestMode() <-chan string {
	return rw.tasksManager.EnableTestMode()
}
