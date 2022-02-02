package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-rootcerts"
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
	ctrl    Server
	port    int
	version string
	srv     *http.Server
	tls     *config.CTSTLSConfig
}

type Config struct {
	Port       int
	TLS        *config.CTSTLSConfig
	Controller Server
}

// NewAPI create a new API object
func NewAPI(conf Config) (*API, error) {
	logger := logging.Global().Named(logSystemName)
	api := &API{
		ctrl:    conf.Controller,
		port:    conf.Port,
		version: defaultAPIVersion,
		tls:     conf.TLS,
	}

	if conf.TLS == nil {
		api.tls = config.DefaultCTSTLSConfig()
	}

	r := chi.NewRouter()

	// add the middleware for all endpoints
	r.Use(withLogging)
	r.Use(withCORS)

	// add the base path route then mount the endpoints
	r.Route(fmt.Sprintf("/%s", defaultAPIVersion), func(r chi.Router) {
		// Legacy Endpoints
		// retrieve overall status
		r.Mount(fmt.Sprintf("/%s", overallStatusPath),
			newOverallStatusHandler(api.ctrl, defaultAPIVersion))

		// retrieve all task statuses
		r.Mount(fmt.Sprintf("/%s", taskStatusPath),
			newTaskStatusHandler(api.ctrl, defaultAPIVersion))

		// crud task
		r.Mount(fmt.Sprintf("/%s", taskPath),
			newTaskHandler(api.ctrl, defaultAPIVersion))
	})

	r.Group(func(r chi.Router) {
		// Use our validation middleware to check all requests against the
		// OpenAPI schema.
		r.Use(withPlaintextErrorToJson)
		r.Use(withSwaggerValidate)

		// Generated Endpoints
		server := Handlers{
			TaskLifeCycleHandler: NewTaskLifeCycleHandler(api.ctrl),
		}
		oapigen.HandlerFromMux(server, r)
	})

	t := &tls.Config{}
	if config.BoolVal(api.tls.Enabled) && config.BoolVal(api.tls.VerifyIncoming) {
		certPool, err := rootcerts.LoadCACerts(&rootcerts.Config{
			CAFile: config.StringVal(api.tls.CACert),
			CAPath: config.StringVal(api.tls.CAPath),
		})
		if err != nil {
			logger.Error("error loading TLS configs for api server", "error", err)
			return nil, err
		}
		t.ClientCAs = certPool
		t.ClientAuth = tls.RequireAndVerifyClientCert
	}

	api.srv = &http.Server{
		Addr:        fmt.Sprintf(":%d", api.port),
		ReadTimeout: time.Second * 15,
		IdleTimeout: time.Second * 60,
		Handler:     r,
		TLSConfig:   t,
		ErrorLog: logger.StandardLogger(&hclog.StandardLoggerOptions{
			InferLevels: false,
			ForceLevel:  hclog.Warn,
		}),
	}

	return api, nil
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
