package api

import (
	"fmt"
	"net/http"

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
	ctrl    Server
	version string
}

// newOverallStatusHandler returns a new overall status handler
func newOverallStatusHandler(ctrl Server, version string) *overallStatusHandler {
	return &overallStatusHandler{
		ctrl:    ctrl,
		version: version,
	}
}

// ServeHTTP serves the overall status endpoint which returns a struct
// containing overall information across all tasks
func (h *overallStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.FromContext(ctx).Named(overallStatusSubsystemName)
	logger.Trace("requesting task status", "url_path", r.URL.Path)
	switch r.Method {
	case http.MethodGet:

		data, err := h.ctrl.Events(ctx, "")
		if err != nil {
			jsonErrorResponse(ctx, w, http.StatusInternalServerError, err)
			return
		}

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

		tasks, err := h.ctrl.Tasks(ctx)
		if err != nil {
			jsonErrorResponse(ctx, w, http.StatusInternalServerError, err)
			return
		}
		for _, task := range tasks {
			// look for any tasks that have a driver but no events
			if _, ok := data[*task.Name]; !ok {
				taskSummary.Status.Unknown++
			}

			if *task.Enabled {
				taskSummary.Enabled.True++
			} else {
				taskSummary.Enabled.False++
			}
		}

		err = jsonResponse(w, http.StatusOK, OverallStatus{
			TaskSummary: taskSummary,
		})
		if err != nil {
			logger.Error("error, could not generate json error response", "error", err)
		}
	default:
		err := fmt.Errorf("'%s' in an unsupported method. The overallStatus API "+
			"currently supports the method(s): '%s'", r.Method, http.MethodGet)
		logger.Trace("unsupported method: %s", err)
		jsonErrorResponse(ctx, w, http.StatusMethodNotAllowed, err)
	}
}
