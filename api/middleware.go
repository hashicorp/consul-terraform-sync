// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/go-uuid"
	middleware "github.com/oapi-codegen/nethttp-middleware"
)

//go:generate mockery --name=Interceptor --filename=middleware.go --output=../mocks/api --tags=enterprise --with-expecter

const (
	timeFormat = "2006-01-02T15:04:05.000Z0700"
)

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID, err := uuid.GenerateUUID()
		if err != nil {
			logging.Global().Named(logSystemName).Error("error generating uuid", "error", err)
			return
		}

		r = r.WithContext(requestIDWithContext(r.Context(), reqID))
		next.ServeHTTP(w, r)
	})
}

type loggingMiddleware struct {
	uriExclusions map[string]bool
	logger        logging.Logger
}

func newLoggingMiddleware(uriExclusions []string, logger logging.Logger) loggingMiddleware {
	lm := loggingMiddleware{
		logger: logger,
	}
	lm.uriExclusions = make(map[string]bool)
	for _, v := range uriExclusions {
		lm.uriExclusions[v] = true
	}
	return lm
}

// withLogging creates a request ID and logger and adds them to the context passed to the
// next handler. It also handles logging incoming requests and once a request has finished processing.
func (lm loggingMiddleware) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If we are excluding a URI, return here, this is required to support
		// oapi-codegen which has an all-or-nothing approach to generated endpoint middleware
		if _, ok := lm.uriExclusions[r.RequestURI]; ok {
			next.ServeHTTP(w, r)
			return
		}

		logger := lm.logger

		// Add a UUID from the context if available
		logger = logger.With("request_id", requestIDFromContext(r.Context()))
		ts := time.Now()

		// Log request info before calling the next handler
		logger.Debug("received request",
			"time", ts.Format(timeFormat),
			"remote_ip", r.RemoteAddr,
			"uri", r.RequestURI,
			"method", r.Method,
			"host", r.Host)

		r = r.WithContext(logging.WithContext(r.Context(), logger))

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

// Interceptor is an interface for determining when a request needs to be intercepted
// and how that request handled instead.
type Interceptor interface {
	ShouldIntercept(*http.Request) bool
	Intercept(http.ResponseWriter, *http.Request)
}

type interceptMiddleware struct {
	i Interceptor
}

func newInterceptMiddleware(i Interceptor) *interceptMiddleware {
	return &interceptMiddleware{
		i: i,
	}
}

// withIntercept will intercept the request if the given condition is met
// and apply custom logic to the request instead. It will not serve the request
// to the next handler. If the condition is not met, then it will continue
// to serve the request to the next handler without any modifications.
func (im interceptMiddleware) withIntercept(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if im.i.ShouldIntercept(r) {
			im.i.Intercept(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
