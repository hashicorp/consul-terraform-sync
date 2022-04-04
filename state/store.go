package state

import (
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/state/event"
)

// Store stores the CTS state
type Store interface {

	// GetConfig returns the CTS configuration
	GetConfig() config.Config

	// GetTaskEvents retrieves all the events for a task
	GetTaskEvents(taskName string) map[string][]event.Event

	// DeleteTaskEvents deletes all the events for a task
	DeleteTaskEvents(taskName string)

	// AddTaskEvent adds an event for a task
	AddTaskEvent(event event.Event) error
}
