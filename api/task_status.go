package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/event"
	"github.com/hashicorp/consul-terraform-sync/logging"
)

const (
	taskStatusPath          = "status/tasks"
	taskStatusSubsystemName = "taskstatus"
)

// TaskStatus is the status for a single task
type TaskStatus struct {
	TaskName  string        `json:"task_name"`
	Status    string        `json:"status"`
	Enabled   bool          `json:"enabled"`
	EventsURL string        `json:"events_url"`
	Events    []event.Event `json:"events,omitempty"`

	// Providers and Services are deprecated in v0.5. These are configuration
	// details about the task rather than status information. Users should
	// switch to using the Get Task API to request the task's provider and
	// services (and more!) information.
	//  - Providers should be removed in 0.8
	//  - Services should be removed in a future major release after 0.8 to align
	//  with the removal of CTS config task.services
	Providers []string `json:"providers"`
	Services  []string `json:"services"`
}

// taskStatusHandler handles the task status endpoint
type taskStatusHandler struct {
	ctrl    Server
	version string
}

// newTaskStatusHandler returns a new TaskStatusHandler
func newTaskStatusHandler(ctrl Server, version string) *taskStatusHandler {
	return &taskStatusHandler{
		ctrl:    ctrl,
		version: version,
	}
}

// ServeHTTP serves the task status endpoint
func (h *taskStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context()).Named(taskStatusSubsystemName)
	logger.Trace("request task status", "url_path", r.URL.Path)

	switch r.Method {
	case http.MethodGet:
		h.getTaskStatus(w, r)
	default:
		err := fmt.Errorf("'%s' in an unsupported method. The task status API "+
			"currently supports the method(s): '%s'", r.Method, http.MethodGet)
		logger.Trace("unsupported method: %s", err)
		jsonErrorResponse(r.Context(), w, http.StatusMethodNotAllowed, err)
	}
}

// getTaskStatus returns a map of taskname to task status
func (h *taskStatusHandler) getTaskStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.FromContext(ctx).Named(taskStatusSubsystemName)

	taskName, err := getTaskName(r.URL.Path, taskStatusPath, h.version)
	if err != nil {
		logger.Trace("bad request", "error", err)
		jsonErrorResponse(ctx, w, http.StatusBadRequest, err)
		return
	}

	filter, err := statusFilter(r)
	if err != nil {
		logger.Trace("bad request", "error", err)
		jsonErrorResponse(ctx, w, http.StatusBadRequest, err)
		return
	}

	include, err := include(r)
	if err != nil {
		logger.Trace("bad request", "error", err)
		jsonErrorResponse(ctx, w, http.StatusBadRequest, err)
		return
	}

	data, err := h.ctrl.Events(ctx, taskName)
	statuses := make(map[string]TaskStatus)
	for taskName, events := range data {
		task, err := h.ctrl.Task(ctx, taskName)
		if err != nil {
			logger.Trace("error getting task", "error", err)
			jsonErrorResponse(ctx, w, http.StatusNotFound, err)
			return
		}
		status := makeTaskStatus(events, task, h.version)

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
			task, err := h.ctrl.Task(ctx, taskName)
			if err != nil {
				logger.Trace("error getting task", "error", err)
				jsonErrorResponse(ctx, w, http.StatusNotFound, err)
				return
			}
			statuses[taskName] = makeTaskStatusUnknown(task)
		}
	}

	// if user requested all tasks and status filter applicable, check driver
	// for tasks without events
	if taskName == "" && (filter == "" || filter == StatusUnknown) {
		tasks, err := h.ctrl.Tasks(ctx)
		if err != nil {
			logger.Trace("error getting tasks", "error", err)
			jsonErrorResponse(ctx, w, http.StatusInternalServerError, err)
			return
		}
		for _, task := range tasks {
			if _, ok := data[*task.Name]; !ok {
				statuses[*task.Name] = makeTaskStatusUnknown(task)
			}
		}
	}

	if err = jsonResponse(w, http.StatusOK, statuses); err != nil {
		logger.Error("error, could not generate json response", "error", err)
	}
}

// makeTaskStatus takes event data for a task and returns a task status
func makeTaskStatus(events []event.Event, task config.TaskConfig,
	version string) TaskStatus {

	successes := make([]bool, len(events))
	uniqProviders := make(map[string]bool)
	uniqServices := make(map[string]bool)

	for i, e := range events {
		successes[i] = e.Success
		if e.Config == nil {
			continue
		}
		for _, p := range e.Config.Providers {
			uniqProviders[p] = true
		}
		for _, s := range e.Config.Services {
			uniqServices[s] = true
		}
	}

	taskName := *task.Name
	return TaskStatus{
		TaskName:  taskName,
		Status:    successToStatus(successes),
		Enabled:   *task.Enabled,
		Providers: mapKeyToArray(uniqProviders),
		Services:  mapKeyToArray(uniqServices),
		EventsURL: makeEventsURL(events, version, taskName),
	}
}

// makeTaskStatusUnknown returns a task status for tasks that do not have events
// but still exist within CTS. Example: a task that has been disabled from the start
func makeTaskStatusUnknown(task config.TaskConfig) TaskStatus {
	return TaskStatus{
		TaskName:  *task.Name,
		Status:    StatusUnknown,
		Enabled:   *task.Enabled,
		Providers: task.Providers,
		Services:  task.DeprecatedServices,
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

// include determines whether to include events in task status payload
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
