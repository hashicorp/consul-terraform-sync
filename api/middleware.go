package api

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	middleware "github.com/deepmap/oapi-codegen/pkg/chi-middleware"
	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/go-uuid"
)

const (
	timeFormat = "2006-01-02T15:04:05.000Z0700"
)

// withLogging creates a request ID and logger and adds them to the context passed to the
// next handler. It also handles logging incoming requests and once a request has finished processing.
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate a UUID to be included with all log messages for an incoming request
		reqID, err := uuid.GenerateUUID()

		logger := logging.Global().Named(logSystemName)
		if err != nil {
			logging.Global().Named(logSystemName).Error("error generating uuid", "error", err)
			return
		}

		// UUID was successfully created, so add it now
		logger = logger.With("request_id", reqID)
		ts := time.Now()

		// Log request info before calling the next handler
		logger.Debug("received request",
			"time", ts.Format(timeFormat),
			"remote_ip", r.RemoteAddr,
			"uri", r.RequestURI,
			"method", r.Method,
			"host", r.Host)

		r = r.WithContext(logging.WithContext(r.Context(), logger))
		r = r.WithContext(requestIDWithContext(r.Context(), reqID))

		// Use logger response writer so that the status code can be captured for logging
		rw := &loggerResponseWriter{
			ResponseWriter: w,
		}

		next.ServeHTTP(rw, r)

		// Log request info on exit
		logger.Debug("request complete",
			"duration", fmt.Sprintf("%dus", time.Since(ts).Microseconds()),
			"status_code", rw.statusCode)
	})
}

// withCORS adds the required CORS headers for interacting with web pages
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, PATCH, POST, DELETE")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// withPlaintextErrorToJson processes any plain text errors and converts them
// to the CTS JSON error response
func withPlaintextErrorToJson(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &plaintextErrorToJsonResponseWriter{
			ResponseWriter: w,
			buf:            &bytes.Buffer{},
			requestID:      requestIDFromContext(r.Context()),
		}

		next.ServeHTTP(rw, r)
	})
}

// withSwaggerValidate validates incoming requests against the openAPI schema.
// This schema is generated as part of the openAPI generated code.
func withSwaggerValidate(next http.Handler) http.Handler {
	swagger, err := oapigen.GetSwagger()
	if err != nil {
		// This should never error
		panic("there was an error getting the swagger")
	}

	// Clear out the servers array in the swagger spec. It is recommended to do this so that it skips validating
	// that server names match.
	swagger.Servers = nil

	f := middleware.OapiRequestValidator(swagger)
	return f(next)
}
