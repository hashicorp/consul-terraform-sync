package controller

import (
	"context"
	"log"
	"time"

	"github.com/hashicorp/consul-nia/config"
	"github.com/hashicorp/consul-nia/driver"
	"github.com/hashicorp/hcat"
)

var _ Controller = (*ReadWrite)(nil)

// ReadWrite is the controller to run in read-write mode
type ReadWrite struct {
	driver    driver.Driver
	conf      *config.Config
	templates map[string]*hcat.Template
	watcher   *hcat.Watcher
}

// NewReadWrite configures and initializes a new ReadWrite controller
func NewReadWrite(conf *config.Config) (*ReadWrite, error) {
	d, err := newDriver(conf)
	if err != nil {
		return nil, err
	}

	return &ReadWrite{
		driver:  d,
		conf:    conf,
		watcher: newWatcher(conf, newConsulClient(conf)),
	}, nil
}

// Init initializes the controller before it can be run. Ensures that
// driver is initializes, works are created for each task.
func (rw *ReadWrite) Init() error {
	log.Printf("[INFO] (controller.readwrite) initializing driver")
	if err := rw.driver.Init(); err != nil {
		log.Printf("[ERR] (controller.readwrite) error initializing driver: %s", err)
		return err
	}

	log.Printf("[INFO] (controller.readwrite) driver initialized")

	// initialize tasks. this is hardcoded in main function for demo purposes
	// TODO: separate by provider instances using workspaces.
	// Future: improve by combining tasks into workflows.
	log.Printf("[INFO] (controller.readwrite) initializing all tasks and workers")
	tasks := newDriverTasks(rw.conf)
	for _, task := range tasks {
		log.Printf("[DEBUG] (controller.readwrite) initializing task %q", task.Name)
		err := rw.driver.InitTask(task, true)
		if err != nil {
			log.Printf("[ERR] (controller.readwrite) error initializing task %q to be executed: %s", task.Name, err)
			return err
		}

		log.Printf("[DEBUG] (controller.readwrite) initializing worker for task %q", task.Name)
		err = rw.driver.InitWorker(task)
		if err != nil {
			log.Printf("[ERR] (controller.readwrite) error initializing worker for task %q: %s", task.Name, err)
			return err
		}
	}

	templates, err := newTaskTemplates(rw.conf)
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
	r := hcat.NewResolver()
	results := make([]string, 0, len(rw.templates))
	for {
		for taskName, tmpl := range rw.templates {
			result, err := r.Run(tmpl, rw.watcher)
			if err != nil {
				// TODO handle when rendering and execution is per task instead of bulk
				log.Fatalf("[ERR] error running template for task %s: %s", taskName, err)
			}
			if result.Complete {
				rendered, err := tmpl.Render(result.Contents)
				if err != nil {
					log.Printf("[ERR] error rendering template for task %s: %s", taskName, err)
					continue
				}

				log.Printf("[DEBUG] %q rendered: %+v", taskName, rendered)
				results = append(results, taskName)
			}
		}
		if len(results) == len(rw.templates) {
			break
		}

		// TODO configure
		err := rw.watcher.Wait(time.Second)
		if err != nil {
			log.Fatalf("[ERR] templates could not render: %s", err)
		}
	}

	log.Printf("[INFO] (controller.readwrite) init work")
	rw.driver.InitWork(ctx)

	log.Printf("[INFO] (controller.readwrite) apply work")
	rw.driver.ApplyWork(ctx)

	return nil
}
