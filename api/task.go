package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/mitchellh/mapstructure"
)

const (
	updateTaskSubsystemName = "updatetask"
	createTaskSubsystemName = "createtask"
	deleteTaskSubsystemName = "deletetask"
	getTaskSubsystemName    = "gettask"

	taskPath = "tasks"

	RunOptionInspect = "inspect"
	RunOptionNow     = "now"
)

// taskHandler handles the tasks endpoint
type taskHandler struct {
	ctrl    Server
	version string
}

// newTaskHandler returns a new taskHandler
func newTaskHandler(ctrl Server, version string) *taskHandler {
	return &taskHandler{
		ctrl:    ctrl,
		version: version,
	}
}

// ServeHTTP serves the tasks endpoint
func (h *taskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	logger.Trace("requesting tasks", "url_path", r.URL.Path)

	switch r.Method {
	case http.MethodPatch:
		h.updateTask(w, r)
	default:
		err := fmt.Errorf("'%s' in an unsupported method. The task API "+
			"currently supports the method(s): '%s'", r.Method, http.MethodPatch)
		logger.Trace("unsupported method", "error", err)
		jsonErrorResponse(r.Context(), w, http.StatusMethodNotAllowed, err)
	}
}

// UpdateTaskConfig contains the fields available for patch updating a task.
// Not all task configuration is available for update
type UpdateTaskConfig struct {
	Enabled *bool `mapstructure:"enabled"`
}

type UpdateTaskResponse struct {
	Inspect *InspectPlan `json:"inspect,omitempty"`
}

type InspectPlan struct {
	ChangesPresent bool   `json:"changes_present"`
	Plan           string `json:"plan"`
	URL            string `json:"url,omitempty"`
}

// updateTask does a patch update to an existing task
func (h *taskHandler) updateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskName, err := getTaskName(r.URL.Path, taskPath, h.version)
	logger := logging.FromContext(ctx).Named(updateTaskSubsystemName)
	if err != nil {
		logger.Trace("bad request", "error", err)
		jsonErrorResponse(ctx, w, http.StatusBadRequest, err)
		return
	}

	if taskName == "" {
		err := fmt.Errorf("no task name was included in the api request. " +
			"Updating a task requires the task name: '/v1/tasks/:task_name'")
		logger.Trace("bad request", "error", err)
		jsonErrorResponse(ctx, w, http.StatusBadRequest, err)
		return
	}
	logger = logger.With("task_name", taskName)

	runOp, err := parseRunOption(r)
	if err != nil {
		logger.Trace("unsupported run option", "error", err)
		jsonErrorResponse(ctx, w, http.StatusBadRequest, err)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Trace("unable to read request body from update", "error", err)
		jsonErrorResponse(ctx, w, http.StatusInternalServerError, err)
		return
	}

	conf, err := decodeBody(body)
	if err != nil {
		logger.Trace("problem decoding body from update request for task", "error", err)
		jsonErrorResponse(ctx, w, http.StatusBadRequest, err)
		return
	}

	if conf.Enabled == nil {
		err = fmt.Errorf("/v1/tasks/:task_name currently only supports the 'enabled' field. Missing 'enabled' from the request body")
		jsonErrorResponse(ctx, w, http.StatusBadRequest, err)
		return
	}

	// Check if task exists
	tc, err := h.ctrl.Task(ctx, taskName)
	if err != nil {
		logger.Trace("task not found", "error", err)
		sendError(w, r, http.StatusNotFound, err)
		return
	}

	tc.Enabled = conf.Enabled
	if runOp == RunOptionInspect {
		logger.Info("generating inspect plan if task becomes enabled")
	} else {
		if *conf.Enabled {
			logger.Info("enabling task")
		} else {
			logger.Info("disabling task")
		}
	}

	// Update the task
	changes, plan, url, err := h.ctrl.TaskUpdate(ctx, tc, runOp)
	if err != nil {
		sendError(w, r, http.StatusInternalServerError, err)
		return
	}

	switch runOp {
	case RunOptionInspect:
		resp := UpdateTaskResponse{Inspect: &InspectPlan{
			ChangesPresent: changes,
			Plan:           plan,
			URL:            url,
		}}
		if err = jsonResponse(w, http.StatusOK, &resp); err != nil {
			logger.Error("error, could not generate json response", "error", err)
		}
	case RunOptionNow, "":
		if err = jsonResponse(w, http.StatusOK, UpdateTaskResponse{}); err != nil {
			logger.Error("error, could not generate json response", "error", err)
		}
	}
}

type TaskLifeCycleHandler struct {
	mu   sync.RWMutex
	ctrl Server
}

func NewTaskLifeCycleHandler(ctrl Server) *TaskLifeCycleHandler {
	return &TaskLifeCycleHandler{
		ctrl: ctrl,
	}
}

func decodeBody(body []byte) (UpdateTaskConfig, error) {
	var raw map[string]interface{}

	err := json.Unmarshal(body, &raw)
	if err != nil {
		return UpdateTaskConfig{}, err
	}

	var conf UpdateTaskConfig
	var md mapstructure.Metadata
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		ErrorUnused:      false,
		Metadata:         &md,
		Result:           &conf,
	})
	if err != nil {
		return UpdateTaskConfig{}, err
	}

	if err = decoder.Decode(raw); err != nil {
		return UpdateTaskConfig{}, err
	}

	if len(md.Unused) > 0 {
		sort.Strings(md.Unused)
		err := fmt.Errorf("request body's JSON contains unsupported keys: %s",
			strings.Join(md.Unused, ", "))
		return UpdateTaskConfig{}, err
	}

	return conf, nil
}

// parseRunOption returns a run option for updating the task
func parseRunOption(r *http.Request) (string, error) {
	// `?run=<option>` parameter
	const runKey = "run"

	keys, ok := r.URL.Query()[runKey]
	if !ok {
		return "", nil
	}

	if len(keys) != 1 {
		return "", fmt.Errorf("cannot support more than one run query "+
			"parameter, got run values: %v", keys)
	}

	value := keys[0]
	value = strings.ToLower(value)
	switch value {
	case RunOptionNow, RunOptionInspect:
		return value, nil
	default:
		return "", fmt.Errorf("unsupported run parameter value. only "+
			"supporting run values %s and %s but got %s",
			RunOptionNow, RunOptionInspect, value)
	}
}
