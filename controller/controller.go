package controller

import (
	"context"
	"errors"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
)

const (
	ctrlSystemName = "ctrl"
	taskNameLogKey = "task_name"
)

// Controller describes the interface for monitoring Consul for relevant changes
// and triggering the driver to update network infrastructure.
type Controller interface {
	// Init initializes elements needed by controller. Returns a map of
	// taskname to driver
	Init(ctx context.Context) error

	// Run runs the controller by monitoring Consul and triggering the driver as needed
	Run(ctx context.Context) error

	// Stop stops underlying clients and connections
	Stop()
}

// InstallDriver installs necessary drivers based on user configuration.
func InstallDriver(ctx context.Context, conf *config.Config) error {
	if conf.Driver.Terraform != nil {
		return driver.InstallTerraform(ctx, conf.Driver.Terraform)
	}
	return errors.New("unsupported driver")
}
