package controller

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/event"
	"github.com/hashicorp/consul-terraform-sync/retry"
	"github.com/hashicorp/hcat"
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
}

// NewReadWrite configures and initializes a new ReadWrite controller
func NewReadWrite(conf *config.Config, store *event.Store) (Controller, error) {
	baseCtrl, err := newBaseController(conf)
	if err != nil {
		return nil, err
	}

	return &ReadWrite{
		baseController: baseCtrl,
		store:          store,
		retry:          retry.NewRetry(defaultRetry, time.Now().UnixNano()),
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
	rw.setTemplateBufferPeriods()

	for i := int64(1); ; i++ {
		// Blocking on Wait is first as we just ran in Once mode so we want
		// to wait for updates before re-running. Doing it the other way is
		// basically a noop as it checks if templates have been changed but
		// the logs read weird. Revisit after async work is done.
		select {
		case err := <-rw.watcher.WaitCh(ctx):
			if err != nil {
				log.Printf("[ERR] (ctrl) error watching template dependencies: %s", err)
			}

		case <-ctx.Done():
			log.Printf("[INFO] (ctrl) stopping controller")
			return ctx.Err()
		}

		for err := range rw.runUnits(ctx) {
			// aggregate error collector for runUnits, just logs everything for now
			log.Printf("[ERR] (ctrl) %s", err)
		}

		rw.logDepSize(50, i)
	}
}

// A single run through of all the units/tasks/templates
// Returned error channel closes when done with all units
func (rw *ReadWrite) runUnits(ctx context.Context) chan error {
	// keep error chan and waitgroup here to keep runTask simple (on task)
	errCh := make(chan error, 1)
	wg := sync.WaitGroup{}
	for _, u := range rw.units {
		wg.Add(1)
		go func(u unit) {
			_, err := rw.checkApply(ctx, u, true)
			if err != nil {
				errCh <- err
			}
			wg.Done()
		}(u)
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

	completed := make(map[string]bool, len(rw.units))
	for i := int64(0); ; i++ {
		done := true
		for _, u := range rw.units {
			if !completed[u.taskName] {
				complete, err := rw.checkApply(ctx, u, false)
				if err != nil {
					return err
				}
				completed[u.taskName] = complete
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

// Single run, render, apply of a unit (task).
//
// Stores event data on error or on successful _full_ execution of task.
// Note: We do not store event for each successful yet incomplete template fetching
// since there could be many per full task execution i.e. when resolver.Run()
// returns result.Complete == false, no event is stored.
func (rw *ReadWrite) checkApply(ctx context.Context, u unit, retry bool) (bool, error) {
	tmpl := u.template
	taskName := u.taskName

	// setup to store event information
	ev, err := event.NewEvent(taskName, &event.Config{
		Providers: u.providers,
		Services:  u.services,
		Source:    u.source,
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

	log.Printf("[TRACE] (ctrl) checking dependency changes for task %s", taskName)
	var result hcat.ResolveEvent
	if result, storedErr = rw.resolver.Run(tmpl, rw.watcher); storedErr != nil {
		defer storeEvent()
		return false, fmt.Errorf("error fetching template dependencies for task %s: %s",
			taskName, storedErr)
	}

	// result.Complete is only `true` if the template has new data that has been
	// completely fetched. Rendering a template for the first time may take several
	// cycles to load all the dependencies asynchronously.
	if result.Complete {
		log.Printf("[DEBUG] (ctrl) change detected for task %s", taskName)
		defer storeEvent()

		var rendered hcat.RenderResult
		if rendered, storedErr = tmpl.Render(result.Contents); storedErr != nil {
			return false, fmt.Errorf("error rendering template for task %s: %s",
				taskName, storedErr)
		}
		log.Printf("[TRACE] (ctrl) template for task %q rendered: %+v", taskName, rendered)

		d := u.driver
		log.Printf("[INFO] (ctrl) executing task %s", taskName)
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

	return result.Complete, nil
}

// setTemplateBufferPeriods applies the task buffer period config to its template
func (rw *ReadWrite) setTemplateBufferPeriods() {
	if rw.watcher == nil || rw.conf == nil {
		return
	}

	taskConfigs := make(map[string]*config.TaskConfig)
	for _, t := range *rw.conf.Tasks {
		taskConfigs[*t.Name] = t
	}

	var unsetIDs []string
	for _, u := range rw.units {
		taskConfig := taskConfigs[u.taskName]
		if buffPeriod := *taskConfig.BufferPeriod; *buffPeriod.Enabled {
			rw.watcher.SetBufferPeriod(*buffPeriod.Min, *buffPeriod.Max, u.template.ID())
		} else {
			unsetIDs = append(unsetIDs, u.template.ID())
		}
	}

	// Set default buffer period for unset templates
	if buffPeriod := *rw.conf.BufferPeriod; *buffPeriod.Enabled {
		rw.watcher.SetBufferPeriod(*buffPeriod.Min, *buffPeriod.Max, unsetIDs...)
	}
}
