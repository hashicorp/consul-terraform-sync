package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
)

const taskStatusPath = "status/tasks"

// TaskStatus is the status for a single task
type TaskStatus struct {
	TaskName  string        `json:"task_name"`
	Status    string        `json:"status"`
	Enabled   bool          `json:"enabled"`
	Providers []string      `json:"providers"`
	Services  []string      `json:"services"`
	EventsURL string        `json:"events_url"`
	Events    []event.Event `json:"events,omitempty"`
}

// taskStatusHandler handles the task status endpoint
type taskStatusHandler struct {
	store   *event.Store
	drivers map[string]driver.Driver
	version string
}

// newTaskStatusHandler returns a new TaskStatusHandler
func newTaskStatusHandler(store *event.Store, drivers map[string]driver.Driver, version string) *taskStatusHandler {
	return &taskStatusHandler{
		store:   store,
		drivers: drivers,
		version: version,
	}
}

// ServeHTTP serves the task status endpoint which returns a map of taskname to
// task status
func (h *taskStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[TRACE] (api.taskstatus) requesting task status '%s'", r.URL.Path)

	taskName, err := getTaskName(r.URL.Path, taskStatusPath, h.version)
	if err != nil {
		log.Printf("[TRACE] (api.taskstatus) bad request: %s", err)
		jsonErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	filter, err := statusFilter(r)
	if err != nil {
		log.Printf("[TRACE] (api.taskstatus) bad request: %s", err)
		jsonErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	include, err := include(r)
	if err != nil {
		log.Printf("[TRACE] (api.taskstatus) bad request: %s", err)
		jsonErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	data := h.store.Read(taskName)
	statuses := make(map[string]TaskStatus)
	for taskName, events := range data {
		d, ok := h.drivers[taskName]
		if !ok {
			err := fmt.Errorf("task '%s' does not exist", taskName)
			log.Printf("[TRACE] (api.updatetask) %s", err)
			jsonErrorResponse(w, http.StatusNotFound, err)
			return
		}
		status := makeTaskStatus(taskName, events, d.Task(), h.version)

		if filter != "" && status.Status != filter {
			continue
		}
		if include {
			status.Events = events
		}
		statuses[taskName] = status
	}

	// if user requested a specific task that does not have events, check if
	// a driver exists
	if taskName != "" {
		if _, ok := data[taskName]; !ok {
			if d, ok := h.drivers[taskName]; ok {
				statuses[taskName] = makeTaskStatusUnknown(d.Task())
			}
		}
	}

	// if user requested all tasks and status filter applicable, check driver
	// for tasks without events
	if taskName == "" && (filter == "" || filter == StatusUnknown) {
		for tN, d := range h.drivers {
			if _, ok := data[tN]; !ok {
				statuses[tN] = makeTaskStatusUnknown(d.Task())
			}
		}
	}

	jsonResponse(w, http.StatusOK, statuses)
}

// makeTaskStatus takes event data for a task and returns a task status
func makeTaskStatus(taskName string, events []event.Event, task driver.Task,
	version string) TaskStatus {

	successes := make([]bool, len(events))
	uniqProviders := make(map[string]bool)
	uniqServices := make(map[string]bool)

	for i, event := range events {
		successes[i] = event.Success
		if event.Config == nil {
			continue
		}
		for _, p := range event.Config.Providers {
			uniqProviders[p] = true
		}
		for _, s := range event.Config.Services {
			uniqServices[s] = true
		}
	}

	return TaskStatus{
		TaskName:  taskName,
		Status:    successToStatus(successes),
		Enabled:   task.Enabled,
		Providers: mapKeyToArray(uniqProviders),
		Services:  mapKeyToArray(uniqServices),
		EventsURL: makeEventsURL(events, version, taskName),
	}
}

// makeTaskStatusUnknown returns a task status for tasks that do not have events
// but still exist within CTS. Example: a task that has been disabled from the start
func makeTaskStatusUnknown(task driver.Task) TaskStatus {
	return TaskStatus{
		TaskName:  task.Name,
		Status:    StatusUnknown,
		Enabled:   task.Enabled,
		Providers: task.ProviderNames(),
		Services:  task.ServiceNames(),
		EventsURL: "",
	}
}

// mapKeyToArray returns an array of map keys
func mapKeyToArray(m map[string]bool) []string {
	arr := make([]string, len(m))
	ix := 0
	for k := range m {
		arr[ix] = k
		ix++
	}
	return arr
}

// successToStatus determines a status from an array of success/failures
func successToStatus(successes []bool) string {
	if len(successes) == 0 {
		return StatusUnknown
	}

	latest := successes[0]
	if latest {
		// Last event was successful
		return StatusSuccessful
	}

	// Last event had errored, determine if the task is in critical state.
	var errorsCount int
	for _, success := range successes {
		if !success {
			errorsCount++
		}
	}

	if errorsCount > 1 {
		return StatusCritical
	}
	return StatusErrored
}

// makeEventsURL returns an events URL for a task. Returns an empty string
// if the task has no events i.e. no events to look into
func makeEventsURL(events []event.Event, version, taskName string) string {
	if len(events) == 0 {
		return ""
	}

	return fmt.Sprintf("/%s/%s/%s?include=events",
		version, taskStatusPath, taskName)
}

// include determines whether or not to include events in task status payload
func include(r *http.Request) (bool, error) {
	// `?include=events` parameter
	const includeKey = "include"
	const includeValue = "events"

	keys, ok := r.URL.Query()[includeKey]
	if !ok {
		return false, nil
	}

	if len(keys) != 1 {
		return false, fmt.Errorf("cannot support more than one include "+
			"parameter, got include values: %v", keys)
	}

	if keys[0] != includeValue {
		return false, fmt.Errorf("unsupported ?include parameter value. only "+
			"supporting 'include=events' but got 'include=%s'", keys[0])
	}

	return true, nil
}

// statusFilter returns a status to filter task statuses
func statusFilter(r *http.Request) (string, error) {
	// `?status=<health>` parameter
	const statusKey = "status"

	keys, ok := r.URL.Query()[statusKey]
	if !ok {
		return "", nil
	}

	if len(keys) != 1 {
		return "", fmt.Errorf("cannot support more than one status query "+
			"parameter, got status values: %v", keys)
	}

	value := keys[0]
	value = strings.ToLower(value)
	switch value {
	case StatusSuccessful, StatusErrored, StatusCritical, StatusUnknown:
		return value, nil
	default:
		return "", fmt.Errorf("unsupported status parameter value. only "+
			"supporting status values %s, %s, %s, and %s but got %s",
			StatusSuccessful, StatusErrored, StatusCritical, StatusUnknown,
			value)
	}
}
