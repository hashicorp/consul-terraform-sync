package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	"github.com/hashicorp/consul-terraform-sync/logging"
)

// CreateTask creates a task
// TODO: handle inclusion of variables map[string]string
// TODO: handle setting the bufferPeriod of the driver
func (h *TaskLifeCycleHandler) CreateTask(w http.ResponseWriter, r *http.Request, params oapigen.CreateTaskParams) {
	h.mu.Lock()
	defer h.mu.Unlock()
	logger := logging.FromContext(r.Context()).Named(createTaskSubsystemName)
	logger.Trace("create task request received, reading request")

	// Decode the task request
	var req taskRequest
	requestID := requestIDFromContext(r.Context())
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("bad request", "error", err, "create_task_request", r.Body)
		sendError(w, r, http.StatusBadRequest, fmt.Sprintf("error decoding the request: %v", err))
		return
	}
	logger.Trace("create task request", "create_task_request", req)

	// Check if task exists, if it does, do not create again
	if _, ok := h.drivers.Get(req.Name); ok {
		logger.Trace("task already exists", "task_name", req.Name)
		sendError(w, r, http.StatusBadRequest, fmt.Sprintf("task with name %s already exists", req.Name))
		return
	}

	// Convert task request to config task config
	trc, err := req.ToTaskRequestConfig(h.bufferPeriod, h.workingDir)
	if err != nil {
		err = fmt.Errorf("error with task configuration: %s", err)
		logger.Error("error creating task", "error", err)
		sendError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// Create new driver
	d, err := h.createNewTaskDriver(trc.TaskConfig, trc.variables)
	if err != nil {
		err = fmt.Errorf("error creating new task driver: %v", err)
		logger.Error("error creating task", "error", err)
		sendError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	// Create a dry run task if run parameter is set to inspect
	var run string
	if params.Run != nil {
		run = string(*params.Run)
	}
	if run == driver.RunOptionInspect {
		logger.Trace("run inspect option", "task_name", d.Task().Name())
		h.createDryRunTask(w, r, d, trc)
		return
	}

	// Create a new event for tracking infrastructure change required actions
	var storedErr error
	task := d.Task()
	ev, err := event.NewEvent(task.Name(), &event.Config{
		Providers: task.ProviderNames(),
		Services:  task.ServiceNames(),
		Source:    task.Source(),
	})
	if err != nil {
		err = fmt.Errorf("error creating new event: %s", err)
		logger.Error("error creating task", "error", err)
		sendError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() {
		ev.End(storedErr)
		logger.Trace("adding event", "event", ev.GoString())
		if err := h.store.Add(*ev); err != nil {
			// only log error since creating a task occurred successfully by now
			logger.Error("error storing event", "event", ev.GoString(), "error", err)
		}
	}()
	ev.Start()

	// Initialize the new task
	storedErr = initNewTask(r.Context(), d)
	if storedErr != nil {
		err = fmt.Errorf("error initializing new task: %s", storedErr)
		logger.Error("error creating task", "error", err)
		sendError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	// Apply task if run option is now
	if run == driver.RunOptionNow {
		logger.Trace("run now option", "task_name", d.Task().Name())
		err = d.ApplyTask(r.Context())
		if err != nil {
			err = fmt.Errorf("error applying new task: %s", err)
			logger.Error("error applying task", "error", err)
			sendError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	}

	// Add the task driver to the driver list
	err = h.drivers.Add(req.Name, d)
	if err != nil {
		err = fmt.Errorf("error initializing new task: %s", err)
		logger.Error("error creating task", "error", err)
		sendError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// Return the task response
	resp := taskResponseFromTaskRequestConfig(trc, requestID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		logger.Error("error encoding json", "error", err, "create_task_response", resp)
	}
	logger.Trace("task created", "create_task_response", resp)
}

func (h *TaskLifeCycleHandler) createDryRunTask(w http.ResponseWriter, r *http.Request,
	d driver.Driver, taskConf taskRequestConfig) {
	logger := logging.FromContext(r.Context()).Named(createTaskSubsystemName)

	// Initialize the dry run task
	err := initNewTask(r.Context(), d)
	if err != nil {
		err = fmt.Errorf("error initializing new task: %s", err)
		logger.Error("error creating task", "error", err)
		sendError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	// Inspect task
	plan, err := d.InspectTask(r.Context())
	if err != nil {
		err = fmt.Errorf("error inspecting task: %s", err)
		logger.Error("error creating task", "error", err, "task_name", d.Task().Name())
		sendError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	requestID := requestIDFromContext(r.Context())
	resp := taskResponseFromTaskRequestConfig(taskConf, requestID)
	resp.Run = &oapigen.Run{
		Plan: &plan.Plan,
	}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		logger.Error("error encoding json", "error", err, "create_task_response", resp)
	}
	logger.Trace("task inspection complete", "create_task_response", resp)
}
