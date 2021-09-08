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
		next.ServeHTTP(w, r)

		// Log info on exit
		logger.Info("request complete",
			"duration", fmt.Sprintf("%dus", time.Since(ts).Microseconds()))
	})
}
