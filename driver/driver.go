package driver

import "context"

//go:generate mockery --name=Driver --filename=driver.go  --output=../mocks/driver

// Driver describes the interface for using an Sync driver to carry out changes
// downstream to update network infrastructure.
type Driver interface {
	// InitTask initializes the task that the driver executes
	InitTask(ctx context.Context, force bool) error

	// InspectTask inspects for any differences pertaining to the task between
	// the state of Consul and network infrastructure
	InspectTask(ctx context.Context) error

	// ApplyTask applies change for the task managed by the driver
	ApplyTask(ctx context.Context) error

	// Version returns the version of the driver.
	Version() string
}
