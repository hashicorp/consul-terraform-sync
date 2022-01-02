package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/logging"
)

// DeleteTaskByName deletes an existing task and its events. Does not delete
// if the task is active.
func (h *TaskLifeCycleHandler) DeleteTaskByName(w http.ResponseWriter, r *http.Request, name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	logger := logging.FromContext(r.Context()).Named(deleteTaskSubsystemName)
	logger.Trace("deleting task", "task_name", name)

	// Check if task exists
	_, ok := h.drivers.Get(name)
	if !ok {
		err := fmt.Errorf("a task with the name '%s' does not exist or has not "+
			"been initialized yet", name)
		logger.Trace("task not found", "task_name", name, "error", err)
		sendError(w, r, http.StatusNotFound, fmt.Sprint(err))
		return
	}

	// Check if task is active
	if h.drivers.IsActive(name) {
		err := fmt.Errorf("task '%s' is currently running and cannot be deleted "+
			"at this time", name)
		logger.Trace("task active", "task_name", name, "error", err)
		sendError(w, r, http.StatusConflict, fmt.Sprint(err))
		return
	}

	// Delete task driver and events
	err := h.drivers.Delete(name)
	if err != nil {
		logger.Trace("unable to delete task", "task_name", name, "error", err)
		sendError(w, r, http.StatusInternalServerError, fmt.Sprint(err))
		return
	}
	h.store.Delete(name)
	logger.Trace("task deleted", "task_name", name)

	var resp oapigen.TaskResponse
	requestID := requestIDFromContext(r.Context())
	resp.RequestId = requestID

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		logger.Error("error encoding json response", "error", err, "response", resp)
	}
}
