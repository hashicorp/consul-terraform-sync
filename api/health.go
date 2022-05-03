package api

import (
	"errors"
	"net/http"
	"sync"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/health"
	"github.com/hashicorp/consul-terraform-sync/logging"
)

const (
	getHealthSubsystemName = "gethealth"
)

// HealthHandler handles the health endpoints
type HealthHandler struct {
	mu      sync.RWMutex
	checker health.Checker
}

// NewHealthHandler creates a new health handler using the provided health manager
// to determine health
func NewHealthHandler(hc health.Checker) *HealthHandler {
	return &HealthHandler{
		checker: hc,
	}
}

// GetHealth returns the health status
func (hh *HealthHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	hh.mu.RLock()
	defer hh.mu.RUnlock()

	logger := logging.FromContext(r.Context()).Named(getHealthSubsystemName)
	logger.Trace("get health")

	err := hh.checker.Check()

	// use error type to determine if service is considered unhealthy and return
	// a 503: service unavailable response if the system is unhealthy
	if err != nil {
		resp := oapigen.BadHealthCheckResponse{
			Error: oapigen.Error{Message: err.Error()},
		}

		var unhealthyErr *health.UnhealthySystemError
		if errors.As(err, &unhealthyErr) {
			logger.Error("system is unhealthy", "error", err)
			writeResponse(w, r, http.StatusServiceUnavailable, resp)
		} else {
			logger.Error("error checking health", "error", err)
			writeResponse(w, r, http.StatusInternalServerError, resp)
		}
	} else {
		logger.Trace("system is healthy")
		writeResponse(w, r, http.StatusOK, oapigen.GoodHealthCheckResponse{})
	}

	logger.Trace("health retrieved")
}
