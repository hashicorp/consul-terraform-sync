package controller

import (
	"context"
	"errors"
	"log"

	"github.com/hashicorp/consul-nia/config"
	"github.com/hashicorp/consul-nia/driver"
)

// Controller describes the interface for monitoring Consul for relevant changes
// and triggering the driver to update network infrastructure.
type Controller interface {
	// Init initializes elements needed by controller
	Init() error

	// Run runs the controller by monitoring Consul and triggering the driver as needed
	Run(ctx context.Context) error
}

func newDriver(conf *config.Config) (driver.Driver, error) {
	var driver driver.Driver
	if conf.Driver.Terraform != nil {
		log.Printf("[INFO] (controller) setting up Terraform driver")
		driver = newTerraformDriver(conf)
		return driver, nil
	}
	return nil, errors.New("Unsupported driver")
}

func newTerraformDriver(conf *config.Config) *driver.Terraform {
	tfConf := *conf.Driver.Terraform
	return driver.NewTerraform(&driver.TerraformConfig{
		LogLevel:          *tfConf.LogLevel,
		Path:              *tfConf.Path,
		DataDir:           *tfConf.DataDir,
		WorkingDir:        *tfConf.WorkingDir,
		SkipVerify:        *tfConf.SkipVerify,
		Backend:           tfConf.Backend,
		RequiredProviders: tfConf.RequiredProviders,
	})
}

// newDriverTasks converts user-defined task configurations to the task object
// used by drivers.
func newDriverTasks(conf *config.Config) []driver.Task {
	tasks := make([]driver.Task, len(*conf.Tasks))
	for i, t := range *conf.Tasks {

		services := make([]driver.Service, len(t.Services))
		for si, service := range t.Services {
			services[si] = getService(conf.Services, service)
		}

		providers := make([]map[string]interface{}, len(t.Providers))
		providerInfo := make(map[string]interface{})
		for pi, pName := range t.Providers {
			providers[pi] = getProvider(conf.Providers, pName)

			// This is Terraform specific to pass version and source info for
			// providers from the required_provider block
			if tfConf := conf.Driver.Terraform; tfConf != nil {
				if pInfo, ok := tfConf.RequiredProviders[pName]; ok {
					providerInfo[pName] = pInfo
				}
			}
		}

		tasks[i] = driver.Task{
			Description:  *t.Description,
			Name:         *t.Name,
			Providers:    providers,
			ProviderInfo: providerInfo,
			Services:     services,
			Source:       *t.Source,
			Version:      *t.Version,
		}
	}

	return tasks
}

// getService is a helper to find and convert a user-defined service
// configuration by ID to a driver service type. If a service is not
// explicitly configured, it assumes the service is a logical service name
// in the default namespace.
func getService(services *config.ServiceConfigs, id string) driver.Service {
	for _, s := range *services {
		if *s.ID == id {
			return driver.Service{
				Datacenter:  *s.Datacenter,
				Description: *s.Description,
				Name:        *s.Name,
				Namespace:   *s.Namespace,
				Tag:         *s.Tag,
			}
		}
	}

	return driver.Service{Name: id}
}

// getProvider is a helper to find and convert a user-defined provider
// configuration by the provider name. If a provider is not explicitly
// configured, it assumes the default provider block that is empty.
//
// provider "name" { }
func getProvider(providers *config.ProviderConfigs, name string) map[string]interface{} {
	for _, p := range *providers {
		if _, ok := (*p)[name]; ok {
			return *p
		}
	}

	return map[string]interface{}{name: make(map[string]interface{})}
}
