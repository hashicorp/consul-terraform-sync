package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

const (
	updateTaskSubsystemName = "updatetask"
	taskPath                = "tasks"
)

// taskHandler handles the tasks endpoint
type taskHandler struct {
	store   *event.Store
	drivers *driver.Drivers
	version string
}

// newTaskHandler returns a new taskHandler
func newTaskHandler(store *event.Store, drivers *driver.Drivers,
	version string) *taskHandler {

	return &taskHandler{
		store:   store,
		drivers: drivers,
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
	Inspect *driver.InspectPlan `json:"inspect,omitempty"`
}

// updateTask does a patch update to an existing task
func (h *taskHandler) updateTask(w http.ResponseWriter, r *http.Request) {
	taskName, err := getTaskName(r.URL.Path, taskPath, h.version)
	logger := logging.FromContext(r.Context()).Named(updateTaskSubsystemName)
	if err != nil {
		logger.Trace("bad request", "error", err)

		jsonErrorResponse(r.Context(), w, http.StatusBadRequest, err)
		return
	}

	if taskName == "" {
		err := fmt.Errorf("no task name was included in the api request. " +
			"Updating a task requires the task name: '/v1/tasks/:task_name'")
		logger.Trace("bad request", "error", err)
		jsonErrorResponse(r.Context(), w, http.StatusBadRequest, err)
		return
	}

	d, ok := h.drivers.Get(taskName)
	if !ok {
		err := fmt.Errorf("a task with the name '%s' does not exist or has not "+
			"been initialized yet", taskName)
		logger.Trace("task not found", "error", err)
		jsonErrorResponse(r.Context(), w, http.StatusNotFound, err)
		return
	}
	h.drivers.SetActive(taskName)
	defer h.drivers.SetInactive(taskName)

	runOp, err := runOption(r)
	if err != nil {
		logger.Trace("unsupported run option", "error", err)
		jsonErrorResponse(r.Context(), w, http.StatusBadRequest, err)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Trace("unable to read request body from update", "task_name", taskName, "error", err)
		jsonErrorResponse(r.Context(), w, http.StatusInternalServerError, err)
		return
	}

	conf, err := decodeBody(body)
	if err != nil {
		logger.Trace("problem decoding body from update request "+
			"for task", "task_name", taskName, "error", err)
		jsonErrorResponse(r.Context(), w, http.StatusBadRequest, err)
		return
	}

	patch := driver.PatchTask{
		RunOption: runOp,
	}
	if conf.Enabled != nil {
		patch.Enabled = config.BoolVal(conf.Enabled)

		if runOp == driver.RunOptionInspect {
			logger.Info("generating inspect plan if task becomes enabled",
				"task_name", taskName)
		} else {
			logger.Info("enabling task", "task_name", taskName)
		}
	}

	var storedErr error
	if runOp == driver.RunOptionNow {
		task := d.Task()
		ev, err := event.NewEvent(taskName, &event.Config{
			Providers: task.ProviderNames(),
			Services:  task.ServiceNames(),
			Source:    task.Source(),
		})
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("error creating task update"+
				"event for %q", taskName))
			logger.Error("error creating new event", "error", err)
			jsonErrorResponse(r.Context(), w, http.StatusInternalServerError, err)
			return
		}
		defer func() {
			ev.End(storedErr)
			logger.Trace("adding event", "event", ev.GoString())
			if err := h.store.Add(*ev); err != nil {
				// only log error since update task occurred successfully by now
				logger.Error("error storing event", "event", ev.GoString(), "error", err)
			}
		}()
		ev.Start()
	}
	var plan driver.InspectPlan
	plan, storedErr = d.UpdateTask(r.Context(), patch)
	if storedErr != nil {
		logger.Trace("error while updating task", "task_name", taskName, "error", err)
		jsonErrorResponse(r.Context(), w, http.StatusInternalServerError, storedErr)
		return
	}

	if runOp != driver.RunOptionInspect {
		if err = jsonResponse(w, http.StatusOK, UpdateTaskResponse{}); err != nil {
			logger.Error("error, could not generate json error response", "error", err)
		}
		return
	}

	if err = jsonResponse(w, http.StatusOK, UpdateTaskResponse{&plan}); err != nil {
		logger.Error("error, could not generate json response", "error", err)
	}
}

func decodeBody(body []byte) (UpdateTaskConfig, error) {
	var raw map[string]interface{}

	err := json.Unmarshal(body, &raw)
	if err != nil {
		return UpdateTaskConfig{}, err
	}

	var config UpdateTaskConfig
	var md mapstructure.Metadata
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		ErrorUnused:      false,
		Metadata:         &md,
		Result:           &config,
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

	return config, nil
}

// runOption returns a run option for updating the task
func runOption(r *http.Request) (string, error) {
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
	case driver.RunOptionNow, driver.RunOptionInspect:
		return value, nil
	default:
		return "", fmt.Errorf("unsupported run parameter value. only "+
			"supporting run values %s and %s but got %s",
			driver.RunOptionNow, driver.RunOptionInspect, value)
	}
}
