package controller

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
)

var _ Controller = (*ReadOnly)(nil)

// ReadOnly is the controller to run in read-only mode
type ReadOnly struct {
	*baseController
}

// NewReadOnly configures and initializes a new ReadOnly controller
func NewReadOnly(conf *config.Config) (Controller, error) {
	// Run the driver with logging to output the Terraform plan to stdout
	if tfConfig := conf.Driver.Terraform; tfConfig != nil {
		tfConfig.Log = config.Bool(true)
	}

	baseCtrl, err := newBaseController(conf)
	if err != nil {
		return nil, err
	}

	return &ReadOnly{baseController: baseCtrl}, nil
}

// Init initializes the controller before it can be run
func (ctrl *ReadOnly) Init(ctx context.Context) (*driver.Drivers, error) {
	drivers, err := ctrl.init(ctx)
	if err != nil {
		return nil, err
	}

	// Sort units for consistent ordering when inspecting tasks
	sort.Slice(ctrl.units, func(i, j int) bool {
		return ctrl.units[i].taskName < ctrl.units[j].taskName
	})

	return drivers, nil
}

// Run runs the controller in read-only mode by checking Consul catalog once for
// latest and using the driver to plan network infrastructure changes
func (ctrl *ReadOnly) Run(ctx context.Context) error {
	log.Println("[INFO] (ctrl) inspecting all tasks")

	completed := make(map[string]bool, len(ctrl.units))
	for i := int64(0); ; i++ {
		done := true
		for _, u := range ctrl.units {
			if !completed[u.taskName] {
				complete, err := ctrl.checkInspect(ctx, u)
				if err != nil {
					return err
				}
				completed[u.taskName] = complete
				if !complete && done {
					done = false
				}
			}
		}
		ctrl.logDepSize(50, i)
		if done {
			log.Println("[INFO] (ctrl) completed task inspections")
			return nil
		}

		select {
		case err := <-ctrl.watcher.WaitCh(ctx):
			if err != nil {
				log.Printf("[ERR] (ctrl) error watching template dependencies: %s", err)
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (ctrl *ReadOnly) checkInspect(ctx context.Context, u unit) (bool, error) {
	taskName := u.taskName

	log.Printf("[TRACE] (ctrl) checking dependencies changes for task %s", taskName)

	d := u.driver
	rendered, err := d.RenderTemplate(ctx)
	if err != nil {
		return false, fmt.Errorf("error rendering template for task %s: %s",
			taskName, err)
	}

	// rendering a template may take several cycles in order to completely fetch
	// new data
	if rendered {
		log.Printf("[TRACE] (ctrl) template for task %q rendered: %+v", taskName, rendered)

		log.Printf("[INFO] (ctrl) inspecting task %s", taskName)
		if err := d.InspectTask(ctx); err != nil {
			return false, fmt.Errorf("could not apply changes for task %s: %s", taskName, err)
		}

		log.Printf("[INFO] (ctrl) inspected task %s", taskName)
	}

	return rendered, nil
}
