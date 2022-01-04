package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
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
	ctx := r.Context()
	requestID := requestIDFromContext(ctx) // TODO: log with request ID
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("bad request", "error", err, "create_task_request", r.Body)
		sendError(w, r, http.StatusBadRequest, fmt.Errorf("error decoding the request: %v", err))
		return
	}
	logger = logger.With("task_name", req.Name)
	logger.Trace("create task request", "create_task_request", req)

	// Check if task exists, if it does, do not create again
	if _, err := h.ctrl.Task(ctx, req.Name); err == nil {
		logger.Trace("task already exists")
		sendError(w, r, http.StatusBadRequest, fmt.Errorf("task with name %s already exists", req.Name))
		return
	}

	// Convert task request to config task config
	trc, err := req.ToTaskRequestConfig()
	if err != nil {
		err = fmt.Errorf("error converting create task request to task config, %s", err)
		logger.Error("error creating task", "error", err)
		sendError(w, r, http.StatusBadRequest, err)
		return
	}

	var tc config.TaskConfig
	if params.Run != nil && *params.Run == driver.RunOptionNow {
		// TODO rebase to add inspect handling
		// if *params.Run == driver.RunOptionInspect {
		// 	logger.Trace("run inspect option")
		// 	_, _, err = h.ctrl.TaskInspect(ctx, trc)
		// }
		logger.Trace("run now option")
		tc, err = h.ctrl.TaskCreateAndRun(ctx, trc)
	} else {
		tc, err = h.ctrl.TaskCreate(ctx, trc)
	}
	if err != nil {
		sendError(w, r, http.StatusInternalServerError, err)
		return
	}

	// Return the task response
	resp := taskResponseFromTaskConfig(tc, requestID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		logger.Error("error encoding json", "error", err, "create_task_response", resp)
	}
	logger.Trace("task created", "create_task_response", resp)
}
