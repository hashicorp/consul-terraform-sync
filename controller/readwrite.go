package controller

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	"github.com/hashicorp/consul-terraform-sync/retry"
)

var (
	_ Controller = (*ReadWrite)(nil)

	// Number of times to retry attempts
	defaultRetry uint = 2
)

// ReadWrite is the controller to run in read-write mode
type ReadWrite struct {
	*baseController
	store *event.Store
	retry retry.Retry

	testMode   bool
	taskNotify chan string
}

// NewReadWrite configures and initializes a new ReadWrite controller
func NewReadWrite(conf *config.Config) (Controller, error) {
	baseCtrl, err := newBaseController(conf)
	if err != nil {
		return nil, err
	}

	return &ReadWrite{
		baseController: baseCtrl,
		store:          event.NewStore(),
		retry:          retry.NewRetry(defaultRetry, time.Now().UnixNano()),
		taskNotify:     make(chan string, len(*conf.Tasks)),
	}, nil
}

// Init initializes the controller before it can be run. Ensures that
// driver is initializes, works are created for each task.
func (rw *ReadWrite) Init(ctx context.Context) error {
	return rw.init(ctx)
}

// Run runs the controller in read-write mode by continuously monitoring Consul
// catalog and using the driver to apply network infrastructure changes for
// any work that have been updated.
// Blocking call that runs main consul monitoring loop
func (rw *ReadWrite) Run(ctx context.Context) error {
	// Only initialize buffer periods for running the full loop and not for Once
	// mode so it can immediately render the first time.
	rw.drivers.SetBufferPeriod()

	for i := int64(1); ; i++ {
		// Blocking on Wait is first as we just ran in Once mode so we want
		// to wait for updates before re-running. Doing it the other way is
		// basically a noop as it checks if templates have been changed but
		// the logs read weird. Revisit after async work is done.
		select {
		case err := <-rw.watcher.WaitCh(ctx):
			if err != nil {
				log.Printf("[ERR] (ctrl) error watching template dependencies: %s", err)
				return err
			}

		case <-ctx.Done():
			log.Printf("[INFO] (ctrl) stopping controller")
			return ctx.Err()
		}

		for err := range rw.runTasks(ctx) {
			// aggregate error collector for runTasks, just logs everything for now
			log.Printf("[ERR] (ctrl) %s", err)
		}

		rw.logDepSize(50, i)
	}
}

// A single run through of all the drivers/tasks/templates
// Returned error channel closes when done with all units
func (rw *ReadWrite) runTasks(ctx context.Context) chan error {
	// keep error chan and waitgroup here to keep runTask simple (on task)
	errCh := make(chan error, 1)
	wg := sync.WaitGroup{}
	for taskName, d := range rw.drivers.Map() {
		if rw.drivers.IsActive(taskName) {
			// The driver is currently active with the task, initiated by an ad-hoc run.
			// There may be updates for other tasks, so we'll continue checking
			log.Printf("[TRACE] (ctrl) task %q is active", taskName)
			continue
		}
		wg.Add(1)
		go func(taskName string, d driver.Driver) {
			complete, err := rw.checkApply(ctx, d, true)
			if err != nil {
				errCh <- err
			}

			if rw.testMode && complete {
				rw.taskNotify <- taskName
			}
			wg.Done()
		}(taskName, d)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	return errCh
}

// Once runs the controller in read-write mode making sure each template has
// been fully rendered and the task run, then it returns.
func (rw *ReadWrite) Once(ctx context.Context) error {
	log.Println("[INFO] (ctrl) executing all tasks once through")

	driversCopy := rw.drivers.Map()
	completed := make(map[string]bool, len(driversCopy))
	for i := int64(0); ; i++ {
		done := true
		for taskName, d := range driversCopy {
			if !completed[taskName] {
				complete, err := rw.checkApply(ctx, d, false)
				if err != nil {
					return err
				}
				completed[taskName] = complete
				if !complete && done {
					done = false
				}
			}
		}
		rw.logDepSize(50, i)
		if done {
			log.Println("[INFO] (ctrl) all tasks completed once")
			return nil
		}

		select {
		case err := <-rw.watcher.WaitCh(ctx):
			if err != nil {
				log.Printf("[ERR] (ctrl) error watching template dependencies: %s", err)
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// ServeAPI runs the API server for the controller
func (rw *ReadWrite) ServeAPI(ctx context.Context) error {
	return api.NewAPI(rw.store, rw.drivers, config.IntVal(rw.conf.Port)).Serve(ctx)
}

// Single run, render, apply of a unit (task).
//
// Stores event data on error or on successful _full_ execution of task.
// Note: We do not store event when a template has not completely rendered since
// driver.RenderTemplate() may be called many times per full task execution
// since there could be many per full task execution i.e. when driver.RenderTemplate()
// returns false, no event is stored.
func (rw *ReadWrite) checkApply(ctx context.Context, d driver.Driver, retry bool) (bool, error) {
	task := d.Task()
	taskName := task.Name()
	if !task.IsEnabled() {
		log.Printf("[TRACE] (ctrl) skipping disabled task '%s'", taskName)
		return true, nil
	}

	// setup to store event information
	ev, err := event.NewEvent(taskName, &event.Config{
		Providers: task.ProviderNames(),
		Services:  task.ServiceNames(),
		Source:    task.Source(),
	})
	if err != nil {
		return false, fmt.Errorf("error creating event for task %s: %s",
			taskName, err)
	}
	var storedErr error
	storeEvent := func() {
		ev.End(storedErr)
		log.Printf("[TRACE] (ctrl) adding event %s", ev.GoString())
		if err := rw.store.Add(*ev); err != nil {
			log.Printf("[ERROR] (ctrl) error storing event %s", ev.GoString())
		}
	}
	ev.Start()

	var rendered bool
	rendered, storedErr = d.RenderTemplate(ctx)
	if storedErr != nil {
		defer storeEvent()
		return false, fmt.Errorf("error rendering template for task %s: %s",
			taskName, storedErr)
	}

	// rendering a template may take several cycles in order to completely fetch
	// new data
	if rendered {
		log.Printf("[INFO] (ctrl) executing task %s", taskName)
		defer storeEvent()

		if retry {
			desc := fmt.Sprintf("ApplyTask %s", taskName)
			storedErr = rw.retry.Do(ctx, d.ApplyTask, desc)
		} else {
			storedErr = d.ApplyTask(ctx)
		}
		if storedErr != nil {
			return false, fmt.Errorf("could not apply changes for task %s: %s",
				taskName, storedErr)
		}

		log.Printf("[INFO] (ctrl) task completed %s", taskName)
	}

	return rendered, nil
}

// EnableTestMode is a helper for testing which tasks were triggered and
// executed. Callers of this method must consume from TaskNotifyChannel to
// prevent the buffered channel from filling and causing a dead lock.
func (rw *ReadWrite) EnableTestMode() {
	rw.testMode = true
}

// TaskNotifyChannel returns a read only channel. The channel should only be
// consumed by one reader.
func (rw *ReadWrite) TaskNotifyChannel() (<-chan string, error) {
	if !rw.testMode {
		// This may change in the future, but for it errors
		return nil, fmt.Errorf("TaskNotifyChannel is only for testing purposes")
	}
	return rw.taskNotify, nil
}
