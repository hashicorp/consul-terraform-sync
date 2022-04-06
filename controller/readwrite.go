package controller

import (
	"context"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/state"
	"github.com/hashicorp/consul-terraform-sync/templates"
)

var (
	_ Controller = (*ReadWrite)(nil)

	// Number of times to retry attempts
	defaultRetry uint = 2
)

// ReadWrite is the controller to run in read-write mode
type ReadWrite struct {
	logger logging.Logger

	state        state.Store
	watcher      templates.Watcher
	tasksManager *TasksManager
	monitor      *ConditionMonitor
}

// NewReadWrite configures and initializes a new ReadWrite controller
func NewReadWrite(conf *config.Config) (*ReadWrite, error) {
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

	tr, err := NewConditionMonitor(state, tm, watcher)
	if err != nil {
		return nil, err
	}

	return &ReadWrite{
		logger:       logger,
		watcher:      watcher,
		state:        state,
		tasksManager: tm,
		monitor:      tr,
	}, nil
}

// Init initializes the controller before it can be run. Ensures that
// driver is initializes, works are created for each task.
func (rw *ReadWrite) Init(ctx context.Context) error {
	return nil
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
		// exit if server is unsuccessful
		exitCh <- err
	}()

	// Run tasks
	go func() {
		if err := rw.monitor.Start(ctx); err != nil {
			if err == context.Canceled {
				exitCh <- err
				// only log, don't exit TODO:
			}
		}
	}()

	counter := 0
	for {
		err := <-exitCh
		counter++
		if err != context.Canceled {
			// Exit if error is returned
			return err
		}
		if counter >= exitBufLen {
			// Wait for all contexts to cancel
			return ctx.Err()
		}
	}
}

func (rw *ReadWrite) Once(ctx context.Context) error {
	rw.logger.Info("running tasks once")
	// TODO: split out consecutive

	// start watcher in order to render templates to apply tasks once
	ctxWatcher, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		for {
			rw.logger.Trace("starting once-mode template dependency monitoring")
			err := <-rw.watcher.WaitCh(ctxWatcher)
			if err == context.Canceled {
				rw.logger.Info("stopping once-mode dependency monitoring")
				return
			}
			if err != nil {
				rw.logger.Error("error monitoring once-mode template dependencies", "error", err)
				return
			}
		}
	}()

	return rw.onceConsecutive(ctx)
}

func (rw *ReadWrite) onceConsecutive(ctx context.Context) error {
	tasks := rw.state.GetAllTasks()
	for ix, task := range tasks {
		rw.logger.Info("running task once", taskNameLogKey, *task.Name)
		if _, err := rw.tasksManager.TaskCreateAndRun(ctx, *task); err != nil {
			return err
		}
		logDepSize(rw.watcher, rw.logger, 50, int64(ix))
	}
	return nil
}

func (rw *ReadWrite) Stop() {
	rw.watcher.Stop()
}

// EnableTestMode is a helper for testing which tasks were triggered and
// executed. Callers of this method must consume from TaskNotify channel to
// prevent the buffered channel from filling and causing a dead lock.
func (rw *ReadWrite) EnableTestMode() <-chan string {
	return rw.monitor.EnableTestMode()
}
