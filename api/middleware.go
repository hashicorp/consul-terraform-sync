package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/go-uuid"
)

const (
	timeFormat = "2006-01-02T15:04:05.000Z0700"
)

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
		logger = logger.With("reqID", reqID)
		ts := time.Now()

		// Log info before calling the next handler
		logger.Info("received request",
			"time", ts.Format(timeFormat),
			"remote_ip", r.RemoteAddr,
			"uri", r.RequestURI,
			"method", r.Method,
			"host", r.Host)

		r = r.WithContext(logging.WithContext(r.Context(), logger))
		r = r.WithContext(requestIDWithContext(r.Context(), reqID))
		next.ServeHTTP(w, r)

		// Log info on exit
		logger.Info("request complete",
			"duration", fmt.Sprintf("%dus", time.Since(ts).Microseconds()))
	})
}

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
