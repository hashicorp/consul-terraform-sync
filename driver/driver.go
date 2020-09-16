package driver

import "context"

//go:generate mockery --name=Driver --filename=driver.go  --output=../mocks/driver

// Driver describes the interface for using an NIA driver to carry out changes
// downstream to update network infrastructure.
type Driver interface {
	// Init initializes the driver and environment
	Init(ctx context.Context) error

	// InitTask initializes the task that the driver executes
	InitTask(task Task, force bool) error

	// ApplyTask applies change for the task managed by the driver
	ApplyTask(ctx context.Context) error

	// Version returns the version of the driver.
	Version() string
}
