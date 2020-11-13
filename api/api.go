package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/consul-terraform-sync/event"
)

const (
	defaultAPIVersion = "v1"

	statusHealthy      = "healthy"
	statusDegraded     = "degraded"
	statusCritical     = "critical"
	statusUndetermined = "undetermined"
)

// API supports api requests to the cts biniary
type API struct {
	store   *event.Store
	port    int
	version string
}

// NewAPI create a new API object
func NewAPI(store *event.Store, port int) *API {
	return &API{
		port:    port,
		store:   store,
		version: defaultAPIVersion,
	}
}

// Serve starts up and handles shutdown for the http server to serve
// API requests
func (api *API) Serve(ctx context.Context) {
	mux := http.NewServeMux()

	// retrieve task status for a task-name
	mux.Handle(fmt.Sprintf("/%s/%s/", defaultAPIVersion, taskStatusPath),
		newTaskStatusHandler(api.store, api.version))
	// retrieve all task statuses
	mux.Handle(fmt.Sprintf("/%s/%s", defaultAPIVersion, taskStatusPath),
		newTaskStatusHandler(api.store, api.version))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", api.port),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// go ahead and stop cts from continuing
			log.Fatalf("error starting api server at '%d': '%s'\n", api.port, err)
		}
	}()

	log.Printf("[INFO] (api) server started at port %d", api.port)

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("[INFO] (api) stopping api server")
				if err := srv.Shutdown(ctxShutDown); err != nil {
					log.Printf("[ERROR] (api) error stopping api server: '%s'", err)
				}
				cancel()
				return
			}
		}
	}()
}

// jsonResponse adds the return response for handlers
func jsonResponse(w http.ResponseWriter, code int, response interface{}) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}
