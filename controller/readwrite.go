package controller

import (
	"context"
	"io/ioutil"
	"log"
	"time"

	"github.com/hashicorp/consul-nia/config"
	"github.com/hashicorp/consul-nia/driver"
	"github.com/hashicorp/hcat"
)

var _ Controller = (*ReadWrite)(nil)

// ReadWrite is the controller to run in read-write mode
type ReadWrite struct {
	newDriver  func(*config.Config) driver.Driver
	drivers    map[string]driver.Driver // task-name -> driver
	conf       *config.Config
	fileReader func(string) ([]byte, error)
	templates  map[string]template
	watcher    watcher
	resolver   resolver
}

// NewReadWrite configures and initializes a new ReadWrite controller
func NewReadWrite(conf *config.Config) (*ReadWrite, error) {
	nd, err := newDriverFunc(conf)
	if err != nil {
		return nil, err
	}
	return &ReadWrite{
		newDriver:  nd,
		drivers:    make(map[string]driver.Driver),
		conf:       conf,
		templates:  make(map[string]template),
		watcher:    newWatcher(conf),
		fileReader: ioutil.ReadFile,
		resolver:   hcat.NewResolver(),
	}, nil
}

// Init initializes the controller before it can be run. Ensures that
// driver is initializes, works are created for each task.
func (rw *ReadWrite) Init() error {
	log.Printf("[INFO] (controller.readwrite) initializing driver")

	log.Printf("[INFO] (controller.readwrite) driver initialized")

	// initialize tasks. this is hardcoded in main function for demo purposes
	// TODO: separate by provider instances using workspaces.
	// Future: improve by combining tasks into workflows.
	log.Printf("[INFO] (controller.readwrite) initializing all tasks and workers")
	tasks := newDriverTasks(rw.conf)
	for _, task := range tasks {
		d := rw.newDriver(rw.conf)
		if err := d.Init(); err != nil {
			log.Printf("[ERR] (controller.readwrite) error initializing driver: %s", err)
			return err
		}
		rw.drivers[task.Name] = d

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
	}

	templates, err := newTaskTemplates(rw.conf, rw.fileReader)
	if err != nil {
		log.Printf("[ERR] (controller.readwrite) error initializing template: %s", err)
		return err
	}
	rw.templates = templates

	return nil
}

// Run runs the controller in read-write mode by continuously monitoring Consul
// catalog and using the driver to apply network infrastructure changes for
// any work that have been updated.
func (rw *ReadWrite) Run(ctx context.Context) error {
	log.Printf("[DEBUG] (controller.readwrite) preparing work")
	// renders all templates once and applies for demo purposes
	// TODO: improve and refactor this control loop of template rendering to
	// trigger task runs instead of waiting for all to render once
	// init/apply work on all tasks.
	results := make([]string, 0, len(rw.templates))
	for {
		for taskName, tmpl := range rw.templates {
			result, err := rw.resolver.Run(tmpl, rw.watcher)
			if err != nil {
				// TODO handle when rendering and execution is per task instead of bulk
				log.Printf("[ERR] error running template for task %s: %s", taskName, err)
				return err
			}
			if result.Complete {
				rendered, err := tmpl.Render(result.Contents)
				if err != nil {
					log.Printf("[ERR] error rendering template for task %s: %s", taskName, err)
					continue
				}

				log.Printf("[DEBUG] %q rendered: %+v", taskName, rendered)
				// XXX fire off something to run the task here
				results = append(results, taskName)
				log.Printf("[INFO] (controller.readwrite) init work")
				d := rw.drivers[taskName]
				if err := d.InitWork(ctx); err != nil {
					log.Printf("[ERR] (controller.readwrite) could not initialize: %s", err)
					return err
				}

				log.Printf("[INFO] (controller.readwrite) apply work")
				if err := d.ApplyWork(ctx); err != nil {
					log.Printf("[ERR] (controller.readwrite) could not apply: %s", err)
					return err
				}
			}
		}
		if len(results) == len(rw.templates) {
			break
		}

		// TODO configure
		err := rw.watcher.Wait(time.Second)
		if err != nil {
			log.Printf("[ERR] templates could not render: %s", err)
			return err
		}
	}

	return nil
}
