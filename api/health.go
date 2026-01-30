// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"errors"
	"net/http"
	"sync"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/health"
)

const (
	healthPath = "/v1/health"
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
// Logging is explicitly left out of this method to avoid flooding the logs
// as this endpoint is expected to be hit often by external entities
func (hh *HealthHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	hh.mu.RLock()
	defer hh.mu.RUnlock()

	err := hh.checker.Check()

	// use error type to determine if service is considered unhealthy and return
	// a 503: service unavailable response if the system is unhealthy
	status := http.StatusOK
	resp := oapigen.HealthCheckResponse{}

	if err != nil {
		resp.Error = &oapigen.Error{Message: err.Error()}
		var unhealthyErr *health.UnhealthySystemError
		if errors.As(err, &unhealthyErr) {
			status = http.StatusServiceUnavailable
		} else {
			status = http.StatusInternalServerError
		}
	}

	writeResponse(w, r, status, resp)
}
