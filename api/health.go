package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/manager"
)

type HealthHandler struct {
	mu sync.RWMutex
	hm manager.Health
}

func NewHealthHandler(hm manager.Health) *HealthHandler {
	return &HealthHandler{
		hm: hm,
	}
}

func (hh *HealthHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	hh.mu.RLock()
	defer hh.mu.RUnlock()

	logger := logging.FromContext(r.Context()).Named(manager.HealthSubsystemName)
	logger.Trace("get health")

	ts := oapigen.Timestamp(time.Now().UTC().Format(time.RFC3339))
	err := hh.hm.Check()

	if err != nil {
		resp := oapigen.BadHealthCheckResponse{
			Error:     oapigen.Error{Message: err.Error()},
			Timestamp: ts,
		}

		writeResponse(w, r, http.StatusInternalServerError, resp)
	} else {
		writeResponse(w, r, http.StatusOK, oapigen.GoodHealthCheckResponse{Timestamp: ts})
	}

	logger.Trace("health retrieved")
}
