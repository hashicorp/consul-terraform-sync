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
}

//go:generate oapi-codegen  -package oapigen -generate types -o oapigen/types.go openapi.yaml
//go:generate oapi-codegen  -package oapigen -generate chi-server,spec -o oapigen/server.go openapi.yaml

// writeResponse sets headers and JSON encodes the response body in the response writer
func writeResponse(w http.ResponseWriter, r *http.Request, code int, resp interface{}) {
	logger := logging.FromContext(r.Context()).Named(handlerSubsystemName)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("error encoding json", "error", err, "response", resp)
	}
}

// sendError wraps sending of an error in the Error format
func sendError(w http.ResponseWriter, r *http.Request, code int, err error) {
	writeResponse(w, r, code, oapigen.ErrorResponse{
		Error: oapigen.Error{
			Message: err.Error(),
		},
		RequestId: requestIDFromContext(r.Context()),
	})
}
