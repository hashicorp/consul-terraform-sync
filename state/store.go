package state

import (
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/state/event"
)

// Store stores the CTS state
type Store interface {

	// GetConfig returns a copy of the CTS configuration
	GetConfig() config.Config

	// GetAllTasks returns a copy of all the tasks
	GetAllTasks() config.TaskConfigs

	// SetTask creates a new task or does a patch update with an existing task
	// with the same name
	SetTask(taskConf config.TaskConfig)

	// GetTask returns a copy of the task. Returns false if the task does not exist
	GetTask(taskName string) (config.TaskConfig, bool)

	// DeleteTask deletes a task if it exists
	DeleteTask(taskName string)

	// GetTaskEvents retrieves all the events for a task. If no task name is
	// specified, then it returns events for all tasks
	GetTaskEvents(taskName string) map[string][]event.Event

	// DeleteTaskEvents deletes all the events for a task
	DeleteTaskEvents(taskName string)

	// AddTaskEvent adds an event for a task
	AddTaskEvent(event event.Event) error
}
