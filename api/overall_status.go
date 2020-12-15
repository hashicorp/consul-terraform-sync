package api

import (
	"log"
	"net/http"

	"github.com/hashicorp/consul-terraform-sync/event"
)

const overallStatusPath = "status"

// OverallStatus is the overall status information for cts and across all the tasks
type OverallStatus struct {
	TaskSummary TaskSummary `json:"task_summary"`
}

// TaskSummary is the count of how many tasks have which status
type TaskSummary struct {
	Successful int `json:"successful"`
	Errored    int `json:"errored"`
	Critical   int `json:"critical"`
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
	taskSummary := TaskSummary{}
	for _, events := range tasks {
		successes := make([]bool, len(events))
		for i, event := range events {
			successes[i] = event.Success
		}
		status := successToStatus(successes)
		switch status {
		case StatusSuccessful:
			taskSummary.Successful++
		case StatusErrored:
			taskSummary.Errored++
		case StatusCritical:
			taskSummary.Critical++
		}
	}

	jsonResponse(w, http.StatusOK, OverallStatus{
		TaskSummary: taskSummary,
	})
}
