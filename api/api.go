package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
)

const (
	defaultAPIVersion = "v1"

	// StatusSuccessful is the successful status. This is determined based on status
	// type.
	//
	// Task Status: Determined by the success of a task updating. The 5 most
	// recent task updates are stored as an ‘event’ in CTS. A task is successful
	// when the most recent stored event is successful.
	StatusSuccessful = "successful"

	// StatusErrored is the errored status. This is determined based on status
	// type.
	//
	// Task Status: Determined by the success of a task updating. The 5 most
	// recent task updates are stored as an ‘event’ in CTS. A task is errored
	// when the most recent stored event is not successful but all prior stored
	// events are successful.
	StatusErrored = "errored"

	// StatusCritical is the critical status. This is determined based on status
	// type.
	//
	// Task Status: Determined by the success of a task updating. The 5 most
	// recent task updates are stored as an ‘event’ in CTS. A task is critical
	// when the most recent stored event is not successful and at least one prior
	// stored event is all not succesful.
	StatusCritical = "critical"

	// StatusUnknown is when the status is unknown. This is determined
	// based on status type.
	//
	// Task Status: Determined by the success of a task updating. The 5 most
	// recent task updates are stored as an ‘event’ in CTS. A task is
	// unknown when no event data has been collected yet.
	StatusUnknown = "unknown"
)

// API supports api requests to the cts biniary
type API struct {
	store   *event.Store
	drivers map[string]driver.Driver
	port    int
	version string
	srv     *http.Server
}

// NewAPI create a new API object
func NewAPI(store *event.Store, drivers map[string]driver.Driver, port int) *API {
	mux := http.NewServeMux()

	// retrieve overall status
	mux.Handle(fmt.Sprintf("/%s/%s", defaultAPIVersion, overallStatusPath),
		newOverallStatusHandler(store, defaultAPIVersion))
	// retrieve task status for a task-name
	mux.Handle(fmt.Sprintf("/%s/%s/", defaultAPIVersion, taskStatusPath),
		newTaskStatusHandler(store, defaultAPIVersion))
	// retrieve all task statuses
	mux.Handle(fmt.Sprintf("/%s/%s", defaultAPIVersion, taskStatusPath),
		newTaskStatusHandler(store, defaultAPIVersion))

	// crud task
	mux.Handle(fmt.Sprintf("/%s/%s/", defaultAPIVersion, taskPath),
		newTaskHandler(store, drivers, defaultAPIVersion))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      mux,
	}

	return &API{
		port:    port,
		drivers: drivers,
		store:   store,
		version: defaultAPIVersion,
		srv:     srv,
	}
}

// Serve starts up and handles shutdown for the http server to serve
// API requests
func (api *API) Serve(ctx context.Context) error {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if err := api.srv.Shutdown(ctxShutDown); err != nil {
					log.Printf("[ERROR] (api) error stopping api server: '%s'", err)
				} else {
					log.Printf("[INFO] (api) shutdown api server")
				}
				return
			}
		}
	}()

	log.Printf("[INFO] (api) starting server at '%d'", api.port)
	if err := api.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("[ERROR] (api) serving api at '%d': %s", api.port, err)
		return err
	}

	// wait for shutdown
	wg.Wait()
	return ctx.Err()
}

// FreePort finds the next free port incrementing upwards. Use for testing.
func FreePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err = listener.Close(); err != nil {
		return 0, err
	}
	return port, nil
}

// jsonResponse adds the return response for handlers. Returns if json encode
// errored. Option to check error or add responses to jsonResponse test to
// test json encoding
func jsonResponse(w http.ResponseWriter, code int, response interface{}) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(response)
}

func jsonErrorResponse(w http.ResponseWriter, code int, err error) error {
	return jsonResponse(w, code, NewErrorResponse(err))
}

// getTaskName retrieves the taskname from the url. Returns empty string if no
// taskname is specified
func getTaskName(reqPath, apiPath, version string) (string, error) {
	taskPathNoID := fmt.Sprintf("/%s/%s", version, apiPath)
	if reqPath == taskPathNoID {
		return "", nil
	}

	taskPathWithID := taskPathNoID + "/"
	taskName := strings.TrimPrefix(reqPath, taskPathWithID)
	if invalid := strings.ContainsRune(taskName, '/'); invalid {
		return "", fmt.Errorf("unsupported path '%s'. request must be format "+
			"'path/{task-name}'. task name cannot have '/ ' and api "+
			"does not support further resources", reqPath)
	}

	return taskName, nil
}
