package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"sync"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/hcat"
)

var _ Controller = (*ReadWrite)(nil)

// ReadWrite is the controller to run in read-write mode
type ReadWrite struct {
	conf       *config.Config
	newDriver  func(*config.Config) driver.Driver
	fileReader func(string) ([]byte, error)
	units      []unit
	watcher    watcher
	resolver   resolver
}

// unit of work per template/task
type unit struct {
	taskName string
	driver   driver.Driver
	template template
}

// NewReadWrite configures and initializes a new ReadWrite controller
func NewReadWrite(conf *config.Config) (*ReadWrite, error) {
	nd, err := newDriverFunc(conf)
	if err != nil {
		return nil, err
	}
	watcher, err := newWatcher(conf)
	if err != nil {
		return nil, err
	}
	return &ReadWrite{
		conf:       conf,
		newDriver:  nd,
		fileReader: ioutil.ReadFile,
		watcher:    watcher,
		resolver:   hcat.NewResolver(),
	}, nil
}

// Init initializes the controller before it can be run. Ensures that
// driver is initializes, works are created for each task.
func (rw *ReadWrite) Init(ctx context.Context) error {
	log.Printf("[INFO] (ctrl) initializing driver")

	// TODO: separate by provider instances using workspaces.
	// Future: improve by combining tasks into workflows.
	log.Printf("[INFO] (ctrl) initializing all tasks")
	tasks := newDriverTasks(rw.conf)
	units := make([]unit, 0, len(tasks))

	for _, task := range tasks {
		d := rw.newDriver(rw.conf)
		if err := d.Init(ctx); err != nil {
			log.Printf("[ERR] (ctrl) error initializing driver: %s", err)
			return err
		}
		log.Printf("[DEBUG] (ctrl) initializing task %q", task.Name)
		err := d.InitTask(task, true)
		if err != nil {
			log.Printf("[ERR] (ctrl) error initializing task %q: %s", task.Name, err)
			return err
		}

		template, err := newTaskTemplate(task.Name, rw.conf, rw.fileReader)
		if err != nil {
			log.Printf("[ERR] (ctrl) error initializing template "+
				"for task %q: %s", task.Name, err)
			return err
		}

		units = append(units, unit{
			taskName: task.Name,
			template: template,
			driver:   d,
		})
	}

	rw.units = units

	log.Printf("[INFO] (ctrl) driver initialized")
	return nil
}

// Run runs the controller in read-write mode by continuously monitoring Consul
// catalog and using the driver to apply network infrastructure changes for
// any work that have been updated.
// Blocking call that runs main consul monitoring loop
func (rw *ReadWrite) Run(ctx context.Context) error {
	// Only initialize buffer periods for running the full loop and not for Once
	// mode so it can immediately render the first time.
	rw.setTemplateBufferPeriods()

	for {
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
			rw.watcher.Stop()
			return ctx.Err()
		}

		for err := range rw.runUnits(ctx) {
			// aggregate error collector for runUnits, just logs everything for now
			log.Printf("[ERR] (ctrl) %s", err)
		}
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
			_, err := rw.checkApply(ctx, u)
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
	for {
		done := true
		for _, u := range rw.units {
			if !completed[u.taskName] {
				complete, err := rw.checkApply(ctx, u)
				if err != nil {
					return err
				}
				completed[u.taskName] = complete
				if !complete && done {
					done = false
				}
			}
		}
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

// Single run, render, apply of a unit (task)
func (rw *ReadWrite) checkApply(ctx context.Context, u unit) (bool, error) {
	tmpl := u.template
	taskName := u.taskName

	log.Printf("[TRACE] (ctrl) checking dependencies changes for task %s", taskName)
	result, err := rw.resolver.Run(tmpl, rw.watcher)
	if err != nil {
		return false, fmt.Errorf("error fetching template dependencies for task %s: %s",
			taskName, err)
	}

	// result.Complete is only `true` if the template has new data that has been
	// completely fetched. Rendering a template for the first time may take several
	// cycles to load all the dependencies asynchronously.
	if result.Complete {
		log.Printf("[DEBUG] (ctrl) change detected for task %s", taskName)
		rendered, err := tmpl.Render(result.Contents)
		if err != nil {
			return false, fmt.Errorf("error rendering template for task %s: %s",
				taskName, err)
		}
		log.Printf("[TRACE] (ctrl) template for task %q rendered: %+v", taskName, rendered)

		d := u.driver
		log.Printf("[INFO] (ctrl) executing task %s", taskName)
		if err := d.ApplyTask(ctx); err != nil {
			return false, fmt.Errorf("could not apply changes for task %s: %s", taskName, err)
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
