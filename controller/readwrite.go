package controller

import (
	"context"
	"io/ioutil"
	"log"
	"time"

	"github.com/hashicorp/consul-nia/config"
	"github.com/hashicorp/consul-nia/driver"
	"github.com/hashicorp/consul-nia/handler"
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

	log.Printf("[INFO] (controller.readwrite) driver initialized")

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
			log.Printf("[ERR] (controller.readwrite) error initializing task %q to be executed: %s", task.Name, err)
			return err
		}

		log.Printf("[DEBUG] (controller.readwrite) initializing worker for task %q", task.Name)
		err = d.InitWorker(task)
		if err != nil {
			log.Printf("[ERR] (controller.readwrite) error initializing worker for task %q: %s", task.Name, err)
			return err
		}
		template, err := newTaskTemplate(task.Name, rw.conf, rw.fileReader)
		if err != nil {
			log.Printf("[ERR] (controller.readwrite) error initializing template: %s", err)
			return err
		}
		units = append(units, unit{
			taskName: task.Name,
			template: template,
			driver:   d,
		})
	}

	rw.units = units

	return nil
}

// Run runs the controller in read-write mode by continuously monitoring Consul
// catalog and using the driver to apply network infrastructure changes for
// any work that have been updated.
// Blocking call that runs main consul monitoring loop
func (rw *ReadWrite) Run(ctx context.Context) <-chan error {
	errCh := make(chan error)
	go rw.loop(ctx, errCh)
	return errCh
}

// placeholder until we update hashicat version which has this same code
// as a convenience function
func waitCh(w watcher, timeout time.Duration) <-chan error {
	errCh := make(chan error)
	go func() {
		errCh <- w.Wait(timeout)
	}()
	return errCh
}

// main loop
func (rw *ReadWrite) loop(ctx context.Context, errCh chan<- error) {
	for {
		if err := rw.run(ctx); err != nil {
			errCh <- err
			return
		}

		select {
		case err := <-waitCh(rw.watcher, time.Minute):
			if err != nil {
				errCh <- err
				return
			}
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		}
	}
}

// A single run through of all the templates
func (rw *ReadWrite) run(ctx context.Context) error {
	log.Printf("[DEBUG] (controller.readwrite) preparing work")
	// renders all templates once and applies for demo purposes
	// TODO: improve and refactor this control loop of template rendering to
	// trigger task runs instead of waiting for all to render once
	// init/apply work on all tasks.
	results := make([]string, 0, len(rw.units))
	for _, unit := range rw.units {
		tmpl := unit.template
		taskName := unit.taskName
		log.Print("[DEBUG] (controller.readwrite) running task:", taskName)

		// This only returns result.Complete if the template has new data
		// that has been completely fetched.
		result, err := rw.resolver.Run(tmpl, rw.watcher)
		if err != nil {
			// TODO handle when rendering and execution is per task instead of bulk
			log.Printf("[ERR] error running template for task %s: %s",
				taskName, err)
			return err
		}
		// If true the template should be rendered and driver work run.
		if result.Complete {
			log.Printf("[DEBUG] (controller.readwrite) task %s complete", taskName)
			rendered, err := tmpl.Render(result.Contents)
			if err != nil {
				log.Printf("[ERR] error rendering template for task %s: %s",
					taskName, err)
				continue
			}

			log.Printf("[DEBUG] %q rendered: %+v", taskName, rendered)
			results = append(results, taskName)
			log.Printf("[INFO] (controller.readwrite) init work")
			d := unit.driver
			if err := d.InitWork(ctx); err != nil {
				log.Printf("[ERR] (controller.readwrite) could not initialize: %s",
					err)
				return err
			}

			log.Printf("[INFO] (controller.readwrite) apply work")
			if err := d.ApplyWork(ctx); err != nil {
				log.Printf("[ERR] (controller.readwrite) could not apply: %s", err)
				return err
			}
		}
	}

	if rw.postApply != nil {
		log.Printf("[INFO] (controller.readwrite) post-apply out-of-band actions")
		// TODO: improvement to only trigger handlers for tasks that were updated
		rw.postApply.Do()
	}
	return nil
}
