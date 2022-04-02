package controller

import (
	"context"

	"github.com/hashicorp/consul-terraform-sync/config"
)

var (
	_ Controller = (*ReadWrite)(nil)

	// Number of times to retry attempts
	defaultRetry uint = 2
)

// ReadWrite is the controller to run in read-write mode
type ReadWrite struct {
	tasksManager *TasksManager
}

// NewReadWrite configures and initializes a new ReadWrite controller
func NewReadWrite(conf *config.Config) (*ReadWrite, error) {
	tm, err := NewTasksManager(conf)
	if err != nil {
		return nil, err
	}

	return &ReadWrite{
		tasksManager: tm,
	}, nil
}

// Init initializes the controller before it can be run. Ensures that
// driver is initializes, works are created for each task.
func (rw *ReadWrite) Init(ctx context.Context) error {
	return rw.tasksManager.Init(ctx)
}

func (rw *ReadWrite) Run(ctx context.Context) error {
	return rw.tasksManager.Run(ctx)
}

func (rw *ReadWrite) Once(ctx context.Context) error {
	return rw.tasksManager.RunOnce(ctx)
}

func (rw *ReadWrite) Stop() {
	rw.tasksManager.Stop()
}

// EnableTestMode is a helper for testing which tasks were triggered and
// executed. Callers of this method must consume from TaskNotify channel to
// prevent the buffered channel from filling and causing a dead lock.
func (rw *ReadWrite) EnableTestMode() <-chan string {
	return rw.tasksManager.EnableTestMode()
}
