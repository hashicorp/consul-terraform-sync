package state

import (
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/state/event"
)

// Store stores the CTS state
type Store interface {

	// GetConfig returns a copy of the CTS configuration
	GetConfig() config.Config

	// GetTaskEvents returns all the events for a task. If no task name is
	// specified, then it returns events for all tasks
	GetTaskEvents(taskName string) map[string][]event.Event

	// DeleteTaskEvents deletes all the events for a given task
	DeleteTaskEvents(taskName string)

	// AddTaskEvent adds an event to the store for the task configured in the
	// event
	AddTaskEvent(event event.Event) error
}
