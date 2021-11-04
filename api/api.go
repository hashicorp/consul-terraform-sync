package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	"github.com/hashicorp/consul-terraform-sync/logging"
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
	// stored event is all not successful.
	StatusCritical = "critical"

	// StatusUnknown is when the status is unknown. This is determined
	// based on status type.
	//
	// Task Status: Determined by the success of a task updating. The 5 most
	// recent task updates are stored as an ‘event’ in CTS. A task is
	// unknown when no event data has been collected yet.
	StatusUnknown = "unknown"

	logSystemName = "api"
)

// API supports api requests to the cts binary
type API struct {
	store   *event.Store
	drivers *driver.Drivers
	port    int
	version string
	srv     *http.Server
	tls     *config.TLSConfig
}

// NewAPI create a new API object
func NewAPI(store *event.Store, drivers *driver.Drivers, port int, tls *config.TLSConfig) *API {
	mux := http.NewServeMux()

	// retrieve overall status
	mux.Handle(fmt.Sprintf("/%s/%s", defaultAPIVersion, overallStatusPath),
		withLogging(newOverallStatusHandler(store, drivers, defaultAPIVersion)))
	// retrieve task status for a task-name
	mux.Handle(fmt.Sprintf("/%s/%s/", defaultAPIVersion, taskStatusPath),
		withLogging(newTaskStatusHandler(store, drivers, defaultAPIVersion)))
	// retrieve all task statuses
	mux.Handle(fmt.Sprintf("/%s/%s", defaultAPIVersion, taskStatusPath),
		withLogging(newTaskStatusHandler(store, drivers, defaultAPIVersion)))

	// crud task
	mux.Handle(fmt.Sprintf("/%s/%s/", defaultAPIVersion, taskPath),
		withLogging(newTaskHandler(store, drivers, defaultAPIVersion)))

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
		tls:     tls,
	}
}

// Serve starts up and handles shutdown for the http server to serve
// API requests
func (api *API) Serve(ctx context.Context) error {
	var wg sync.WaitGroup
	wg.Add(1)

	logger := logging.Global().Named(logSystemName)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if err := api.srv.Shutdown(ctxShutDown); err != nil {
					logger.Error("error stopping api server", "error", err)
				} else {
					logger.Info("shutdown api server")
				}
				return
			}
		}
	}()

	logger.Info("starting server", "port", api.port)
	var err error
	if config.BoolVal(api.tls.Enabled) {
		err = api.srv.ListenAndServeTLS(*api.tls.Cert, *api.tls.Key)
	} else {
		err = api.srv.ListenAndServe()
	}
	if err != nil && err != http.ErrServerClosed {
		logger.Error("error serving api", "port", api.port, "error", err)
		return err
	}

	// wait for shutdown
	wg.Wait()
	return ctx.Err()
}

// jsonResponse adds the return response for handlers. Returns if json encode
// errored. Option to check error or add responses to jsonResponse test to
// test json encoding
func jsonResponse(w http.ResponseWriter, code int, response interface{}) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(response)
}

func jsonErrorResponse(ctx context.Context, w http.ResponseWriter, code int, err error) {
	err = jsonResponse(w, code, NewErrorResponse(err))
	if err != nil {
		logging.FromContext(ctx).Named(logSystemName).Error("error, could not generate json error response",
			"error", err)
	}
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
