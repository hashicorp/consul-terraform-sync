package api

import (
	"encoding/json"
	"net/http"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/logging"
)

// Make sure we conform to ServerInterface
var _ oapigen.ServerInterface = (*Handlers)(nil)

const (
	handlerSubsystemName = "handlers"
)

// Handlers is composed of CTS server handlers which allows
// the handler to adhere to the generated server interface
type Handlers struct {
	*TaskLifeCycleHandler
	*DryRunTasksHandler
}

//go:generate oapi-codegen  -package oapigen -generate types -o oapigen/types.go openapi.yaml
//go:generate oapi-codegen  -package oapigen -generate chi-server,spec -o oapigen/server.go openapi.yaml

// sendError wraps sending of an error in the Error format
func sendError(w http.ResponseWriter, r *http.Request, code int, message string) {
	logger := logging.FromContext(r.Context()).Named(handlerSubsystemName)
	taskErr := oapigen.ErrorResponse{
		Error: oapigen.Error{
			Message: message,
		},
		RequestId: requestIDFromContext(r.Context()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(taskErr)
	if err != nil {
		logger.Error("error encoding json", "error", err, "error_response", taskErr)
	}
}
