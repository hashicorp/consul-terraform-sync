package api

import (
	"log"
	"net/http"

	"github.com/hashicorp/consul-terraform-sync/event"
)

const overallStatusPath = "status"

// OverallStatus is the overall status information across all the tasks
type OverallStatus struct {
	Status string `json:"status"`
}

// overallStatusHandler handles the overall status endpoint
type overallStatusHandler struct {
	store   *event.Store
	version string
}

// newOverallStatusHandler returns a new overall status handler
func newOverallStatusHandler(store *event.Store, version string) *overallStatusHandler {
	return &overallStatusHandler{
		store:   store,
		version: version,
	}
}

// ServeHTTP serves the overall status endpoint which returns a struct
// containing overall information across all tasks
func (h *overallStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[TRACE] (api.overallstatus) requesting task status '%s'", r.URL.Path)

	tasks := h.store.Read("")
	statuses := make([]string, len(tasks))
	ix := 0
	for _, events := range tasks {
		successes := make([]bool, len(events))
		for i, event := range events {
			successes[i] = event.Success
		}

		status := successToStatus(successes)
		statuses[ix] = status
		ix++
	}

	jsonResponse(w, http.StatusOK, OverallStatus{
		Status: taskStatusToOverall(statuses),
	})
}

// taskStatusToOverall determines an overall status from the health of all the
// task statuses
func taskStatusToOverall(statuses []string) string {
	total := len(statuses)
	if total == 0 {
		return StatusUndetermined
	}

	statCount := make(map[string]int)
	for _, status := range statuses {
		statCount[status]++
	}

	healthy := statCount[StatusHealthy]
	degraded := statCount[StatusDegraded]
	critical := statCount[StatusCritical]

	switch {
	case healthy == total:
		return StatusHealthy
	case degraded > 0 && critical == 0:
		return StatusDegraded
	default:
		return StatusCritical
	}
}
