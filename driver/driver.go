package driver

import (
	"context"
)

//go:generate mockery --name=Driver --filename=driver.go  --output=../mocks/driver

// Driver describes the interface for using an Sync driver to carry out changes
// downstream to update network infrastructure.
type Driver interface {
	// InitTask initializes the task that the driver executes
	InitTask(ctx context.Context) error

	// SetBufferPeriod sets the task's buffer period on the watcher
	SetBufferPeriod()

	// TemplateIDs returns the list of template IDs for the driver
	TemplateIDs() []string

	// RenderTemplate renders a template. Returns if template rendering
	// completed or not
	RenderTemplate(ctx context.Context) (bool, error)

	// InspectTask inspects for any differences pertaining to the task between
	// the state of Consul and network infrastructure
	InspectTask(ctx context.Context) (InspectPlan, error)

	// ApplyTask applies change for the task managed by the driver
	ApplyTask(ctx context.Context) error

	// UpdateTask supports updating certain fields of a task
	UpdateTask(ctx context.Context, task PatchTask) (InspectPlan, error)

	// DestroyTask destroys task dependencies so that it can be safely deleted
	DestroyTask(ctx context.Context)

	// Task returns the task information of the driver
	Task() *Task

	// Version returns the version of the driver.
	Version() string
}
