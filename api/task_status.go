package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/event"
)

const taskStatusPath = "status/tasks"

// TaskStatus is the status for a single task
type TaskStatus struct {
	TaskName  string        `json:"task_name"`
	Status    string        `json:"status"`
	Providers []string      `json:"providers"`
	Services  []string      `json:"services"`
	EventsURL string        `json:"events_url"`
	Events    []event.Event `json:"events,omitempty"`
}

// taskStatusHandler handles the task status endpoint
type taskStatusHandler struct {
	store   *event.Store
	version string
}

// newTaskStatusHandler returns a new TaskStatusHandler
func newTaskStatusHandler(store *event.Store, version string) *taskStatusHandler {
	return &taskStatusHandler{
		store:   store,
		version: version,
	}
}

// ServeHTTP serves the task status endpoint which returns a map of taskname to
// task status
func (h *taskStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[TRACE] (api.taskstatus) requesting task status '%s'", r.URL.Path)

	taskName, err := getTaskName(r.URL.Path, h.version)
	if err != nil {
		log.Printf("[TRACE] (api.taskstatus) bad request: %s", err)
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	include, err := include(r)
	if err != nil {
		log.Printf("[TRACE] (api.taskstatus) bad request: %s", err)
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	data := h.store.Read(taskName)
	statuses := make(map[string]TaskStatus)
	for taskName, events := range data {
		status := makeTaskStatus(taskName, events, h.version)
		if include {
			status.Events = events
		}
		statuses[taskName] = status
	}

	jsonResponse(w, http.StatusOK, statuses)
}

// getTaskName retrieves the taskname from the url. Returns empty string if no
// taskname is specified
func getTaskName(path, version string) (string, error) {
	taskPathNoID := fmt.Sprintf("/%s/%s", version, taskStatusPath)
	if path == taskPathNoID {
		return "", nil
	}

	taskPathWithID := taskPathNoID + "/"
	taskName := strings.TrimPrefix(path, taskPathWithID)
	if invalid := strings.ContainsRune(taskName, '/'); invalid {
		return "", fmt.Errorf("unsupported path '%s'. request must be format "+
			"'/status/tasks/{task-name}'. task name cannot have '/ ' and api "+
			"does not support further resources", path)
	}

	return taskName, nil
}

// makeTaskStatus takes event data for a task and returns an overall task status
func makeTaskStatus(taskName string, events []event.Event, version string) TaskStatus {
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
		Providers: mapKeyToArray(uniqProviders),
		Services:  mapKeyToArray(uniqServices),
		EventsURL: makeEventsURL(events, version, taskName),
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
		return StatusUndetermined
	}

	total := len(successes)
	mostRecentSuccess := successes[0]
	successCount := 0
	for _, s := range successes {
		if s {
			successCount++
		}
	}

	percentSuccess := 100 * successCount / total
	switch {
	case percentSuccess == 100:
		return StatusHealthy
	case percentSuccess > 50:
		return StatusDegraded
	case mostRecentSuccess == true:
		return StatusDegraded
	default:
		return StatusCritical
	}
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
