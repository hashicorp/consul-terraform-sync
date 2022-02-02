package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
)

const (
	ClientErrorResponseCategory = 4 // category for http status codes from 400-499
	ServerErrorResponseCategory = 5 // category for http status codes from 500-599
)

// checkStatusCodeCategory checks if a given status code matches
// a particular category. It does this by taking the first digit
// of a category (i.e. 4 for 400, 401, etc.) and checking if the first
// digit of the status code matches.
// E.g. category = 4, statusCode = 401 checkStatusCodeCategory returns true
func checkStatusCodeCategory(category int, statusCode int) bool {
	var i int
	for i = statusCode; i >= 10; i = i / 10 {
	}

	return category == i
}

// plaintextErrorToJsonResponseWriter catches plaintext error responses and
// reprocesses them to use the correct JSON error response
type plaintextErrorToJsonResponseWriter struct {
	http.ResponseWriter
	buf        *bytes.Buffer
	requestID  oapigen.RequestID
	statusCode int
}

// Header handles setting the header, if the content type
// is plaintext, it sets the header to json instead
func (r *plaintextErrorToJsonResponseWriter) Header() http.Header {
	h := r.ResponseWriter.Header()
	if h.Get("content-Type") == "text/plain; charset=utf-8" {
		h.Set("content-Type", "application/json")
	}

	return h
}

// WriteHeader handles writing the header and captures the
// status code
func (r *plaintextErrorToJsonResponseWriter) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Write checks if the status code is a 4xx and if it is plaintext. If it is plaintext,
// it converts the error into the correct JSON error response
func (r *plaintextErrorToJsonResponseWriter) Write(p []byte) (int, error) {
	if checkStatusCodeCategory(ClientErrorResponseCategory, r.statusCode) {
		var errResp oapigen.ErrorResponse
		if err := json.Unmarshal(p, &errResp); err != nil {
			msg := strings.TrimSpace(string(p))
			errResp = oapigen.ErrorResponse{
				Error: oapigen.Error{
					Message: msg,
				},
				RequestId: r.requestID,
			}

			b, err := json.Marshal(errResp)
			if err != nil {
				return 0, err
			}

			return r.ResponseWriter.Write(b)
		}
	}

	return r.ResponseWriter.Write(p)
}

// loggerResponseWriter is a wrapper around the stand http response writer that
// captures the status code for use in logging
type loggerResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader handles writing the header and captures the
// status code
func (r *loggerResponseWriter) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}
