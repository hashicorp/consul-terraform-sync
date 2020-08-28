package driver

import "context"

//go:generate mockery --name=Driver --filename=driver.go  --output=../mocks/driver

// Driver describes the interface for using an NIA driver to carry out changes
// downstream to update network infrastructure.
type Driver interface {
	// Init initializes the driver and environment
	Init() error

	// InitTask initializes a task that the driver will execute
	InitTask(task Task, force bool) error

	// InitWorker initializes a worker for a task
	InitWorker(task Task) error

	// InitWork initalizes all work that is managed by driver
	InitTaskWork(taskName string, ctx context.Context) error

	// ApplyWork applies changes for all the work managed by driver
	ApplyTaskWork(taskName string, ctx context.Context) error

	// InitWork initalizes all work that is managed by driver
	InitWork(ctx context.Context) error

	// ApplyWork applies changes for all the work managed by driver
	ApplyWork(ctx context.Context) error

	// Version returns the version of the driver.
	Version() string
}
