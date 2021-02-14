package driver

import (
	"context"

	"github.com/hashicorp/consul-terraform-sync/templates"
)

//go:generate mockery --name=Driver --filename=driver.go  --output=../mocks/driver

// Driver describes the interface for using an Sync driver to carry out changes
// downstream to update network infrastructure.
type Driver interface {
	// InitTask initializes the task that the driver executes
	InitTask(force bool) error

	// SetBufferPeriod sets the task's buffer period on the watcher
	SetBufferPeriod(watcher templates.Watcher)

	// RenderTemplate renders a template. Returns if template rendering
	// completed or not
	RenderTemplate(ctx context.Context, watcher templates.Watcher) (bool, error)

	// InspectTask inspects for any differences pertaining to the task between
	// the state of Consul and network infrastructure
	InspectTask(ctx context.Context) error

	// ApplyTask applies change for the task managed by the driver
	ApplyTask(ctx context.Context) error

	// UpdateTask supports updating certain fields of a task
	UpdateTask(task PatchTask) error

	Task() Task

	// Version returns the version of the driver.
	Version() string
}
