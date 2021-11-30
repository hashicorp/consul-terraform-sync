package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/logging"
)

// deleteTask deletes an existing task and its events. Does not delete
// if the task is active.
func (h *TaskLifeCycleHandler) deleteTask(w http.ResponseWriter, r *http.Request) {
	taskName, err := getTaskName(r.URL.Path, taskPath, h.version)
	logger := logging.FromContext(r.Context()).Named(deleteTaskSubsystemName)
	if err != nil {
		logger.Trace("bad request", "error", err)
		jsonErrorResponse(r.Context(), w, http.StatusBadRequest, err)
		return
	}
	logger.Trace("deleting task", "task_name", taskName)

	if taskName == "" {
		err := fmt.Errorf("no task name was included in the api request. " +
			"Deleting a task requires the task name: '/v1/tasks/:name'")
		logger.Trace("bad request", "error", err)
		jsonErrorResponse(r.Context(), w, http.StatusBadRequest, err)
		return
	}
	logger.Trace("deleting task", "task", taskName)

	// Check if task exists
	_, ok := h.drivers.Get(taskName)
	if !ok {
		err := fmt.Errorf("a task with the name '%s' does not exist or has not "+
			"been initialized yet", taskName)
		logger.Trace("task not found", "task_name", taskName, "error", err)
		sendError(w, r, http.StatusNotFound, fmt.Sprint(err))
		return
	}

	// Check if task is active
	if h.drivers.IsActive(taskName) {
		err := fmt.Errorf("task '%s' is currently running and cannot be deleted "+
			"at this time", taskName)
		logger.Trace("task active", "task_name", taskName, "error", err)
		sendError(w, r, http.StatusConflict, fmt.Sprint(err))
		return
	}

	// Delete task driver and events
	err = h.drivers.Delete(taskName)
	if err != nil {
		logger.Trace("unable to delete task", "task_name", taskName, "error", err)
		sendError(w, r, http.StatusInternalServerError, fmt.Sprint(err))
		return
	}
	h.store.Delete(taskName)
	logger.Trace("task deleted", "task_name", taskName)

	var resp oapigen.TaskDeleteResponse
	requestID := requestIDFromContext(r.Context())
	resp.RequestId = requestID

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		logger.Error("error encoding json response", "error", err, "response", resp)
	}
}
