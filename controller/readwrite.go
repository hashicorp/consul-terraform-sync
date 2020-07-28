package controller

import (
	"context"
	"log"

	"github.com/hashicorp/consul-nia/config"
	"github.com/hashicorp/consul-nia/driver"
)

var _ Controller = (*ReadWrite)(nil)

// ReadWrite is the controller to run in read-write mode
type ReadWrite struct {
	driver driver.Driver
	conf   *config.Config
}

// NewReadWrite configures and initializes a new ReadWrite controller
func NewReadWrite(conf *config.Config) (*ReadWrite, error) {
	d, err := newDriver(conf)
	if err != nil {
		return nil, err
	}
	return &ReadWrite{
		driver: d,
		conf:   conf,
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
		log.Printf("[DEBUG] (controller.readwrite) initializing task %s", task)
		err := rw.driver.InitTask(task, true)
		if err != nil {
			log.Printf("[ERR] (controller.readwrite) error initializing task to be executed %s", err)
			return err
		}

		log.Printf("[DEBUG] (controller.readwrite) initializing worker for task %s", task)
		err = rw.driver.InitWorker(task)
		if err != nil {
			log.Printf("[ERR] (controller.readwrite) error initializing worker for task %s: %s", task, err)
			return err
		}
	}

	return nil
}

// Run runs the controller in read-write mode by continuously monitoring Consul
// catalog and using the driver to apply network infrastructure changes for
// any work that have been updated.
func (rw *ReadWrite) Run(ctx context.Context) error {
	log.Printf("[INFO] (controller.readwrite) init work")
	rw.driver.InitWork(ctx)

	// TODO: monitor consul catalog
	log.Printf("[INFO] (controller.readwrite) apply work")
	rw.driver.ApplyWork(ctx)

	return nil
}
