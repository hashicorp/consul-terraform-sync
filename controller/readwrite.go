package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"sync"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/handler"
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
	postApply  handler.Handler
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
	h, err := getPostApplyHandlers(conf)
	if err != nil {
		return nil, err
	}
	return &ReadWrite{
		conf:       conf,
		newDriver:  nd,
		fileReader: ioutil.ReadFile,
		watcher:    newWatcher(conf),
		resolver:   hcat.NewResolver(),
		postApply:  h,
	}, nil
}

// Init initializes the controller before it can be run. Ensures that
// driver is initializes, works are created for each task.
func (rw *ReadWrite) Init(ctx context.Context) error {
	log.Printf("[INFO] (controller.readwrite) initializing driver")

	// initialize tasks. this is hardcoded in main function for demo purposes
	// TODO: separate by provider instances using workspaces.
	// Future: improve by combining tasks into workflows.
	log.Printf("[INFO] (controller.readwrite) initializing all tasks and workers")
	tasks := newDriverTasks(rw.conf)
	units := make([]unit, 0, len(tasks))

	for _, task := range tasks {
		d := rw.newDriver(rw.conf)
		if err := d.Init(ctx); err != nil {
			log.Printf("[ERR] (controller.readwrite) error initializing driver: %s", err)
			return err
		}
		log.Printf("[DEBUG] (controller.readwrite) initializing task %q", task.Name)
		err := d.InitTask(task, true)
		if err != nil {
			log.Printf("[ERR] (controller.readwrite) error initializing task %q: %s", task.Name, err)
			return err
		}

		template, err := newTaskTemplate(task.Name, rw.conf, rw.fileReader)
		if err != nil {
			log.Printf("[ERR] (controller.readwrite) error initializing template "+
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

	log.Printf("[INFO] (controller.readwrite) driver initialized")
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
				log.Printf("[ERR] (controller.readwrite) wait error from watcher: %s", err)
			}

		case <-ctx.Done():
			log.Printf("[INFO] (controller.readwrite) stopping controller")
			rw.watcher.Stop()
			return ctx.Err()
		}

		for err := range rw.runUnits(ctx) {
			// aggregate error collector for runUnits, just logs everything for now
			log.Printf("[ERR] (controller.readwrite) %s", err)
		}
	}
}

// A single run through of all the units/tasks/templates
// Returned error channel closes when done with all units
func (rw *ReadWrite) runUnits(ctx context.Context) chan error {
	log.Printf("[TRACE] (controller.readwrite) preparing work")

	// keep error chan and waitgroup here to keep runTask simple (on task)
	errCh := make(chan error, 1)
	wg := sync.WaitGroup{}
	for _, u := range rw.units {
		wg.Add(1)
		go func(u unit) {
			_, err := rw.checkApply(u, ctx)
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
	log.Printf("[TRACE] (controller.readwrite) preparing work")

	completed := make(map[string]bool, len(rw.units))
	for {
		done := true
		for _, u := range rw.units {
			if !completed[u.taskName] {
				complete, err := rw.checkApply(u, ctx)
				if err != nil {
					return fmt.Errorf("[ERR] (controller.readwrite): %s", err)
				}
				completed[u.taskName] = complete
				if !complete && done {
					done = false
				}
			}
		}
		if done {
			return nil
		}

		select {
		case err := <-rw.watcher.WaitCh(ctx):
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Single run, render, apply of a unit (task)
func (rw *ReadWrite) checkApply(u unit, ctx context.Context) (bool, error) {
	tmpl := u.template
	taskName := u.taskName

	log.Print("[TRACE] (controller.readwrite) running task:", taskName)
	// This only returns result.Complete if the template has new data
	// that has been completely fetched.
	result, err := rw.resolver.Run(tmpl, rw.watcher)
	if err != nil {
		return false, fmt.Errorf("error running template for task %s: %s",
			taskName, err)
	}
	// If true the template should be rendered and driver work run.
	if result.Complete {
		log.Printf("[DEBUG] (controller.readwrite) change detected for task %s", taskName)
		rendered, err := tmpl.Render(result.Contents)
		if err != nil {
			return false, fmt.Errorf("error rendering template for task %s: %s",
				taskName, err)
		}
		log.Printf("[DEBUG] template for task %q rendered: %+v", taskName, rendered)

		d := u.driver
		log.Printf("[INFO] (controller.readwrite) executing task %s", taskName)
		if err := d.ApplyTask(ctx); err != nil {
			return false, fmt.Errorf("could not apply: %s", err)
		}

		if rw.postApply != nil {
			log.Printf("[TRACE] (controller.readwrite) post-apply out-of-band actions")
			// TODO: improvement to only trigger handlers for tasks that were updated
			if err := rw.postApply.Do(nil); err != nil {
				return false, err
			}
		}
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
