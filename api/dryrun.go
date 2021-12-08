package api

import (
	"net/http"
	"sync"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
)

const (
	dryRunTaskSubsystemName = "dryruntasks"
)

type DryRunTasksHandlerConfig struct {
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
	// logger := logging.FromContext(r.Context()).Named(dryRunTaskSubsystemName)
	// h.mu.Lock()
	// defer h.mu.Unlock()

	// // Read the request
	// requestID := requestIDFromContext(r.Context())
	// var req oapigen.DryRunRequest
	// if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
	// 	sendError(w, r, http.StatusBadRequest, "invalid format for execute DryRun request")
	// 	return
	// }
	// logger.Trace("execute DryRun request", "execute_DryRun_request", r.Body)

	// // Don't do anything with DryRun since this is just a test
	// var resp oapigen.DryRunResponse
	// task := oapigen.Task(req)
	// resp.Task = &task
	// resp.RequestId = requestID

	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusOK)
	// err := json.NewEncoder(w).Encode(resp)
	// if err != nil {
	// 	logger.Error("error encoding json", "error", err, "execute_DryRun_response", resp)
	// }
	// logger.Trace("DryRun executed", "execute_DryRun_response", resp)
}
