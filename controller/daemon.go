package controller

import (
	"context"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/client"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/health"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/registration"
	"github.com/hashicorp/consul-terraform-sync/state"
	"github.com/hashicorp/consul-terraform-sync/templates"
)

var (
	_ Controller = (*Daemon)(nil)

	// Number of times to retry attempts
	defaultRetry = 2
)

// Daemon is the controller to run CTS as a daemon. It executes the tasks once
// (once-mode) and then runs the task in long-running mode. It also starts
// daemon-only features such as the API server
type Daemon struct {
	logger logging.Logger

	state        state.Store
	tasksManager *TasksManager
	watcher      templates.Watcher
	monitor      *ConditionMonitor

	consulClient client.ConsulClientInterface

	// indicates whether the tasks have gone through once-mode or not
	once bool
}

// NewDaemon configures and initializes a new Daemon controller
func NewDaemon(conf *config.Config) (*Daemon, error) {
	logger := logging.Global().Named(ctrlSystemName)
	logger.Info("setting up controller", "type", "daemon")

	s := state.NewInMemoryStore(conf)

	logger.Info("initializing Consul client and testing connection")
	watcher, err := newWatcher(conf, client.ConsulDefaultMaxRetry)
	if err != nil {
		return nil, err
	}

	tm, err := NewTasksManager(conf, s, watcher)
	if err != nil {
		return nil, err
	}

	return &Daemon{
		logger:       logger,
		state:        s,
		tasksManager: tm,
		watcher:      watcher,
		monitor:      NewConditionMonitor(tm, watcher),
	}, nil
}

// Init initializes the controller before it can be run. Ensures that
// driver is initializes, works are created for each task.
func (ctrl *Daemon) Init(ctx context.Context) error {
	return ctrl.tasksManager.Init(ctx)
}

func (ctrl *Daemon) Run(ctx context.Context) error {
	exitBufLen := 2 // api & run tasks exit
	exitCh := make(chan error, exitBufLen)

	// Configure API
	conf := ctrl.tasksManager.state.GetConfig()
	s, err := api.NewAPI(ctx, api.Config{
		Controller: ctrl.tasksManager,
		Health:     &health.BasicChecker{},
		Port:       config.IntVal(conf.Port),
		TLS:        conf.TLS,
	})
	if err != nil {
		return err
	}

	// Serve API
	go func() {
		err := s.Serve(ctx)
		exitCh <- err
	}()

	var rm *registration.ServiceRegistrationManager
	if *conf.Consul.ServiceRegistration.Enabled {
		// Expect one more long-running goroutine
		exitBufLen++
		exitCh = make(chan error, exitBufLen)

		// Configure Consul client if not already
		if ctrl.consulClient == nil {
			c, err := client.NewConsulClient(conf.Consul, client.ConsulDefaultMaxRetry)
			if err != nil {
				ctrl.logger.Error("error setting up Consul client", "error", err)
				return err
			}
			ctrl.consulClient = c
		}

		// Configure and start service registration manager
		rm = registration.NewServiceRegistrationManager(
			&registration.ServiceRegistrationManagerConfig{
				ID:                  *conf.ID,
				Port:                *conf.Port,
				TLSEnabled:          conf.TLS != nil && *conf.TLS.Enabled,
				ServiceRegistration: conf.Consul.ServiceRegistration,
			},
			ctrl.consulClient)

		go func() {
			rm.Start(ctx)
			exitCh <- nil // registration errors are logged only
		}()
	}

	// Run tasks once through once-mode
	if !ctrl.once {
		if err := ctrl.Once(ctx); err != nil {
			return err
		}
	}

	// Run long-running mode and monitor existing
	// and created tasks
	go func() {
		ctrl.logger.Info("start task monitoring")
		err := ctrl.monitor.Run(ctx)
		exitCh <- err
	}()

	counter := 0
	for {
		err := <-exitCh
		counter++
		if err != nil && err != context.Canceled {
			// Exit if an error is returned
			return err
		}
		if counter >= exitBufLen {
			// Wait for all contexts to cancel
			return ctx.Err()
		}
	}
}

// Once runs the tasks once. Intended to only be called by Run()
func (ctrl *Daemon) Once(ctx context.Context) error {
	once := Once{
		logger:       ctrl.logger,
		state:        ctrl.state,
		tasksManager: ctrl.tasksManager,
		monitor:      ctrl.monitor,
	}

	// no need to init or stop Once controller since it shares tasksManager
	// with Daemon controller
	if err := once.Run(ctx); err != nil {
		return err
	}

	ctrl.once = true
	return nil
}

func (ctrl *Daemon) Stop() {
	ctrl.watcher.Stop()
}

func (ctrl *Daemon) EnableTaskRanNotify() <-chan string {
	return ctrl.tasksManager.EnableTaskRanNotify()
}
