package state

import "github.com/hashicorp/consul-terraform-sync/event"

// Store stores the CTS state
type Store interface {

	// GetTaskEvents retrieves all the events for a task
	GetTaskEvents(taskName string) map[string][]event.Event

	// DeleteTaskEvents deletes all the events for a task
	DeleteTaskEvents(taskName string)

	// AddTaskEvent adds an event for a task
	AddTaskEvent(event event.Event) error
}
