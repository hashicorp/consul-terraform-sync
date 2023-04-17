// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/state/event"
)

//go:generate mockery --name=Store --filename=store.go  --output=../mocks/state

// Store stores the CTS state
type Store interface {

	// GetConfig returns a copy of the CTS configuration
	GetConfig() config.Config

	// GetAllTasks returns a copy of the configs for all the tasks
	GetAllTasks() config.TaskConfigs

	// GetTask returns a copy of the task configuration. If the task name does
	// not exist, then it returns false
	GetTask(taskName string) (config.TaskConfig, bool)

	// SetTask adds a new task configuration or does a patch update to an
	// existing task configuration with the same name
	SetTask(taskConf config.TaskConfig) error

	// DeleteTask deletes the task config if it exists
	DeleteTask(taskName string) error

	// GetTaskEvents returns all the events for a task. If no task name is
	// specified, then it returns events for all tasks
	GetTaskEvents(taskName string) map[string][]event.Event

	// DeleteTaskEvents deletes all the events for a given task
	DeleteTaskEvents(taskName string) error

	// AddTaskEvent adds an event to the store for the task configured in the
	// event
	AddTaskEvent(event event.Event) error
}
