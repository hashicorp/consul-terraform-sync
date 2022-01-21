package event

import (
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/go-uuid"
)

const (
	logSystemName = "event"
)

// Event captures the series of actions that needs to happen to update network
// infrastructure for a given task when it receives a service change from Consul.
// An event should encompass: rendering the task’s templates, creating/updating
// resources, and executing any handlers.
type Event struct {
	ID         string    `json:"id"`
	Success    bool      `json:"success"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	TaskName   string    `json:"task_name"`
	EventError *Error    `json:"error"`
	Config     *Config   `json:"config"`
}

// Error captures an event's error information
type Error struct {
	// Code    string `json:"code"` TODO: future work
	Message string `json:"message"`
}

// Config provides details on an event's task configuration
type Config struct {
	Providers []string `json:"providers"`
	Services  []string `json:"services"`

	// Source was deprecated in v0.5. Use Module instead. External packages
	// should use Module except for tests
	Source string `json:"source"`
	// Module introduced in 0.5
	Module string `json:"module"`
}

// NewEvent configures a new event with a task name and any relevant information
// that the task is configured with
func NewEvent(taskName string, config *Config) (*Event, error) {
	if taskName == "" {
		return nil, errors.New("error creating new event: taskname cannot be empty")
	}
	uuid, err := uuid.GenerateUUID()
	if err != nil {
		return nil, err
	}

	// Set deprecated 'source' field if not set
	if config != nil && config.Source == "" {
		config.Source = config.Module
	}

	return &Event{
		ID:       uuid,
		TaskName: taskName,
		Config:   config,
	}, nil
}

// Start sets the start time on an event. Can only be called once.
func (e *Event) Start() {
	if !e.StartTime.IsZero() {
		logging.Global().Named(logSystemName).Warn("event already started. unable to restart")
		return
	}
	e.StartTime = time.Now()
}

// End sets the end time and captures any end results e.g. error, success status.
// Can only be called once
func (e *Event) End(err error) {
	if !e.EndTime.IsZero() {
		logging.Global().Named(logSystemName).Warn("event already ended. unable to re-end")
		return
	}

	e.EndTime = time.Now()

	if err == nil {
		e.Success = true
		return
	}

	e.Success = false
	e.EventError = &Error{
		Message: err.Error(),
	}
}

// GoString defines the printable version of this struct.
func (c *Config) GoString() string {
	if c == nil {
		return "(*Config)(nil)"
	}

	return fmt.Sprintf("&Config{"+
		"Providers:%s, "+
		"Services:%s, "+
		"Module:%s"+
		"}",
		c.Providers,
		c.Services,
		c.Module,
	)
}

// GoString defines the printable version of this struct.
func (e *Event) GoString() string {
	if e == nil {
		return "(*Event)(nil)"
	}

	return fmt.Sprintf("&Event{"+
		"ID:%s, "+
		"TaskName:%s, "+
		"Success:%t, "+
		"StartTime:%s, "+
		"EndTime:%s, "+
		"EventError:%s, "+
		"Config:%s"+
		"}",
		e.ID,
		e.TaskName,
		e.Success,
		e.StartTime,
		e.EndTime,
		e.EventError,
		e.Config.GoString(),
	)
}
