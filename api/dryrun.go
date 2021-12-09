package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/logging"
)

const (
	dryRunTaskSubsystemName = "dryruntasks"
)

type DryRunTasksHandlerConfig struct {
	drivers             *driver.Drivers
	workingDir          string
	createNewTaskDriver func(taskConfig config.TaskConfig, variables map[string]string) (driver.Driver, error)
}

type DryRunTasksHandler struct {
	mu sync.RWMutex
	DryRunTasksHandlerConfig
}

func NewDryRunTasksHandler(c DryRunTasksHandlerConfig) *DryRunTasksHandler {
	return &DryRunTasksHandler{
		DryRunTasksHandlerConfig: c,
	}
}

func (h *DryRunTasksHandler) CreateDryRunTask(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context()).Named(dryRunTaskSubsystemName)
	h.mu.Lock()
	defer h.mu.Unlock()

	// Read and parse the request
	var req taskRequest
	requestID := requestIDFromContext(r.Context())
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("bad request", "error", err, "dryrun_task_request", r.Body)
		sendError(w, r, http.StatusBadRequest, fmt.Sprintf("error decoding the request: %v", err))
		return
	}
	logger.Trace("create dry run task request", "dryrun_task_request", req)

	// Check if task exists
	if _, ok := h.drivers.Get(req.Name); ok {
		logger.Trace("task already exists", "task_name", req.Name)
		sendError(w, r, http.StatusBadRequest, fmt.Sprintf("task with name %s already exists", req.Name))
		return
	}

	// Convert task request to task configuration
	taskConf, err := req.ToTaskRequestConfig(config.DefaultBufferPeriodConfig(), h.workingDir)
	if err != nil {
		err = fmt.Errorf("error converting dry run task request to task config, %s", err)
		logger.Error("error creating dry run task", "error", err)
		sendError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// Create a temporary driver that will be discarded at the end of request
	d, err := h.createNewTaskDriver(taskConf.TaskConfig, taskConf.variables)
	if err != nil {
		err = fmt.Errorf("error creating new task driver: %v", err)
		logger.Error("error creating dry run task", "error", err)
		sendError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// Initialize the new task
	err = initNewTask(r.Context(), d, "")
	if err != nil {
		err = fmt.Errorf("error initializing new task: %s", err)
		logger.Error("error creating task", "error", err)
		sendError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// Inspect task
	plan, err := d.InspectTask(r.Context())
	if err != nil {
		err = fmt.Errorf("error inspecting task: %s", err)
		logger.Error("error creating dry run task", "error", err)
		sendError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := taskResponseFromTaskRequestConfig(taskConf, requestID)
	resp.Run = &oapigen.Run{
		Plan: &plan.Plan,
	}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		logger.Error("error encoding json", "error", err, "dryrun_task_response", resp)
	}
	logger.Trace("dry run task completed", "dryrun_task_response", resp)
}
