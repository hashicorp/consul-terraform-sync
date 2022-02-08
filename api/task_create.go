package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
)

// CreateTask creates a task
func (h *TaskLifeCycleHandler) CreateTask(w http.ResponseWriter, r *http.Request, params oapigen.CreateTaskParams) {
	h.mu.Lock()
	defer h.mu.Unlock()
	logger := logging.FromContext(r.Context()).Named(createTaskSubsystemName)
	logger.Trace("create task request received, reading request")

	// Decode the task request
	var req TaskRequest
	ctx := r.Context()
	requestID := requestIDFromContext(ctx)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("bad request", "error", err, "create_task_request", r.Body)
		sendError(w, r, http.StatusBadRequest,
			fmt.Errorf("error decoding the request: %v", err))
		return
	}
	logger = logger.With("task_name", req.Task.Name)
	logger.Trace("create task request", "create_task_request", req)

	// Check if task exists, if it does, do not create again
	if _, err := h.ctrl.Task(ctx, req.Task.Name); err == nil {
		logger.Trace("task already exists")
		sendError(w, r, http.StatusBadRequest,
			fmt.Errorf("task with name %s already exists", req.Task.Name))
		return
	}

	// Convert task request to config task config
	trc, err := req.ToTaskConfig()
	if err != nil {
		err = fmt.Errorf("error with task configuration: %s", err)
		logger.Error("error creating task", "error", err)
		sendError(w, r, http.StatusBadRequest, err)
		return
	}

	var tc config.TaskConfig
	if params.Run == nil || *params.Run == "" {
		tc, err = h.ctrl.TaskCreate(ctx, trc)
	} else if *params.Run == RunOptionNow {
		logger.Trace("run now option")
		tc, err = h.ctrl.TaskCreateAndRun(ctx, trc)
	} else if *params.Run == RunOptionInspect {
		logger.Trace("run inspect option")
		h.createDryRunTask(w, r, trc)
		return
	}
	if err != nil {
		sendError(w, r, http.StatusInternalServerError, err)
		return
	}

	// Return the task response
	resp := taskResponseFromTaskConfig(tc, requestID)
	writeResponse(w, r, http.StatusCreated, resp)

	logger.Trace("task created", "create_task_response", resp)
}

func (h *TaskLifeCycleHandler) createDryRunTask(w http.ResponseWriter, r *http.Request,
	taskConf config.TaskConfig) {
	ctx := r.Context()
	logger := logging.FromContext(ctx).Named(createTaskSubsystemName).With("task_name", *taskConf.Name)

	// Inspect task
	changes, plan, runUrl, err := h.ctrl.TaskInspect(ctx, taskConf)
	if err != nil {
		logger.Error("error inspecting new task", "error", err)
		sendError(w, r, http.StatusBadRequest, err)
		return
	}

	requestID := requestIDFromContext(ctx)
	resp := taskResponseFromTaskConfig(taskConf, requestID)
	resp.Run = &oapigen.Run{
		Plan:           &plan,
		ChangesPresent: &changes,
	}
	if runUrl != "" {
		resp.Run.TfcRunUrl = &runUrl
	}
	writeResponse(w, r, http.StatusOK, resp)
	logger.Trace("task inspection complete", "create_task_response", resp)
}
