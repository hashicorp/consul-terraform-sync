package api

import (
	"net/http"

	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	"github.com/hashicorp/consul-terraform-sync/logging"
)

const (
	overallStatusPath          = "status"
	overallStatusSubsystemName = "overallstatus"
)

// OverallStatus is the overall status information for cts and across all the tasks
type OverallStatus struct {
	TaskSummary TaskSummary `json:"task_summary"`
}

// TaskSummary holds data that summarizes the tasks configured with CTS
type TaskSummary struct {
	Status  StatusSummary  `json:"status"`
	Enabled EnabledSummary `json:"enabled"`
}

// StatusSummary is the count of how many tasks have which status
type StatusSummary struct {
	Successful int `json:"successful"`
	Errored    int `json:"errored"`
	Critical   int `json:"critical"`
	Unknown    int `json:"unknown"`
}

// EnabledSummary is the count of how many tasks are enabled vs. disabled
type EnabledSummary struct {
	True  int `json:"true"`
	False int `json:"false"`
}

// overallStatusHandler handles the overall status endpoint
type overallStatusHandler struct {
	store   *event.Store
	drivers *driver.Drivers
	version string
}

// newOverallStatusHandler returns a new overall status handler
func newOverallStatusHandler(store *event.Store, drivers *driver.Drivers, version string) *overallStatusHandler {
	return &overallStatusHandler{
		store:   store,
		drivers: drivers,
		version: version,
	}
}

// ServeHTTP serves the overall status endpoint which returns a struct
// containing overall information across all tasks
func (h *overallStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context()).Named(overallStatusSubsystemName)
	logger.Trace("requesting task status", "url_path", r.URL.Path)

	data := h.store.Read("")
	taskSummary := TaskSummary{}
	for _, events := range data {
		successes := make([]bool, len(events))
		for i, event := range events {
			successes[i] = event.Success
		}
		status := successToStatus(successes)
		switch status {
		case StatusSuccessful:
			taskSummary.Status.Successful++
		case StatusErrored:
			taskSummary.Status.Errored++
		case StatusCritical:
			taskSummary.Status.Critical++
		}
	}

	for taskName, d := range h.drivers.Map() {
		// look for any tasks that have a driver but no events
		if _, ok := data[taskName]; !ok {
			taskSummary.Status.Unknown++
		}

		if d.Task().IsEnabled() {
			taskSummary.Enabled.True++
		} else {
			taskSummary.Enabled.False++
		}
	}

	err := jsonResponse(w, http.StatusOK, OverallStatus{
		TaskSummary: taskSummary,
	})
	if err != nil {
		logger.Error("error, could not generate json error response", "error", err)
	}
}
