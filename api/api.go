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

	// StatusHealthy is the healthy status. This is determined based on status
	// type.
	//
	// Task Status: Determined by the success of a task updating. The 5 most
	// recent task updates are stored as an ‘event’ in CTS. A task is healthy
	// when all the stored events are successful.
	StatusHealthy = "healthy"

	// StatusDegraded is the degraded status. This is determined based on status
	// type.
	//
	// Task Status: Determined by the success of a task updating. The 5 most
	// recent task updates are stored as an ‘event’ in CTS. A task is degraded
	// when more than half of the stored events are successful _or_ less than
	// half of the stored events are successful but the most recent event is
	// successful.
	StatusDegraded = "degraded"

	// StatusCritical is the critical status. This is determined based on status
	// type.
	//
	// Task Status: Determined by the success of a task updating. The 5 most
	// recent task updates are stored as an ‘event’ in CTS. A task is critical
	// when less than half of the stored events are successful and the most
	// recent event is not successful
	StatusCritical = "critical"

	// StatusUndetermined is when the status is unknown. This is determined
	// based on status type.
	//
	// Task Status: Determined by the success of a task updating. The 5 most
	// recent task updates are stored as an ‘event’ in CTS. A task is
	// undetermined when no event data has been collected yet.
	StatusUndetermined = "undetermined"
)

// API supports api requests to the cts biniary
type API struct {
	store   *event.Store
	port    int
	version string
	srv     *http.Server
}

// NewAPI create a new API object
func NewAPI(store *event.Store, port int) *API {
	mux := http.NewServeMux()

	// retrieve task status for a task-name
	mux.Handle(fmt.Sprintf("/%s/%s/", defaultAPIVersion, taskStatusPath),
		newTaskStatusHandler(store, defaultAPIVersion))
	// retrieve all task statuses
	mux.Handle(fmt.Sprintf("/%s/%s", defaultAPIVersion, taskStatusPath),
		newTaskStatusHandler(store, defaultAPIVersion))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      mux,
	}

	return &API{
		port:    port,
		store:   store,
		version: defaultAPIVersion,
		srv:     srv,
	}
}

// Serve starts up and handles shutdown for the http server to serve
// API requests
func (api *API) Serve(ctx context.Context) {
	go func() {
		if err := api.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
				if err := api.srv.Shutdown(ctxShutDown); err != nil {
					log.Printf("[ERROR] (api) error stopping api server: '%s'", err)
				}
				return
			}
		}
	}()
}

// jsonResponse adds the return response for handlers. Returns if json encode
// errored. Option to check error or add responses to jsonResponse test to
// test json encoding
func jsonResponse(w http.ResponseWriter, code int, response interface{}) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(response)
}
