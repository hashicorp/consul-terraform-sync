package api

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/consul-terraform-sync/logging"
)

// deleteTask deletes an existing task and its events. Does not delete
// if the task is active.
func (h *taskHandler) deleteTask(w http.ResponseWriter, r *http.Request) {
	taskName, err := getTaskName(r.URL.Path, taskPath, h.version)
	logger := logging.FromContext(r.Context()).Named(deleteTaskSubsystemName)
	if err != nil {
		logger.Trace("bad request", "error", err)
		jsonErrorResponse(r.Context(), w, http.StatusBadRequest, err)
		return
	}

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
		logger.Trace("task not found", "task", taskName, "error", err)
		jsonErrorResponse(r.Context(), w, http.StatusNotFound, err)
		return
	}

	// Check if task is active
	if h.drivers.IsActive(taskName) {
		err := fmt.Errorf("task '%s' is currently running and cannot be deleted "+
			"at this time", taskName)
		logger.Trace("task active", "task", taskName, "error", err)
		jsonErrorResponse(r.Context(), w, http.StatusConflict, err)
		return
	}

	// Delete task driver and events
	err = h.drivers.Delete(taskName)
	if err != nil {
		logger.Trace("unable to delete task", "task", taskName, "error", err)
		jsonErrorResponse(r.Context(), w, http.StatusInternalServerError, err)
		return
	}
	h.store.Delete(taskName)
	logger.Trace("task deleted", "task", taskName)

	w.WriteHeader(http.StatusOK)
}
