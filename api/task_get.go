package api

import (
	"net/http"

	"github.com/hashicorp/consul-terraform-sync/logging"
)

// GetTaskByName retrieves a task's information by the task's name
func (h *TaskLifeCycleHandler) GetTaskByName(w http.ResponseWriter, r *http.Request, name string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ctx := r.Context()
	requestID := requestIDFromContext(ctx)
	logger := logging.FromContext(r.Context()).Named(getTaskSubsystemName).With("task_name", name)
	logger.Trace("get task request")

	// Retrieve task if it exists
	taskConfig, err := h.ctrl.Task(ctx, name)
	if err != nil {
		logger.Trace("task not found", "error", err)
		sendError(w, r, http.StatusNotFound, err)
		return
	}

	resp := taskResponseFromTaskConfig(taskConfig, requestID)
	writeResponse(w, r, http.StatusOK, resp)

	logger.Trace("task retrieved", "get_task_response", resp)
}
