package api

import (
	"net/http"
)

const (
	executDryrunSubsystemName = "executedryrun"
)

type DryrunHandler struct {
	// TODO: replace below when implementing handler
	//lock sync.Mutex
}

func NewDryrunHandler() *DryrunHandler {
	return &DryrunHandler{}
}

func (h *DryrunHandler) ExecuteTaskDryrun(w http.ResponseWriter, r *http.Request) {
	// TODO: replace below when implementing handler
	//logger := logging.FromContext(r.Context()).Named(executDryrunSubsystemName)
	//h.lock.Lock()
	//defer h.lock.Unlock()
	//
	//// Read the dryrun request
	//requestID := requestIDFromContext(r.Context())
	//var req oapigen.DryrunRequest
	//if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
	//	sendError(w, r, http.StatusBadRequest, "invalid format for execute dryrun request")
	//	return
	//}
	//logger.Trace("execute dryrun request", "execute_dryrun_request", r.Body)
	//
	//// Don't do anything with dryrun since this is just a test
	//var resp oapigen.DryrunResponse
	//task := oapigen.Task(req)
	//resp.Task = &task
	//resp.RequestId = requestID
	//
	//w.Header().Set("Content-Type", "application/json")
	//w.WriteHeader(http.StatusOK)
	//err := json.NewEncoder(w).Encode(resp)
	//if err != nil {
	//	logger.Error("error encoding json", "error", err, "execute_dryrun_response", resp)
	//}
	//logger.Trace("dryrun executed", "execute_dryrun_response", resp)
}
