package controller

import (
	"context"
	"errors"
	"log"
	"path"
	"strings"

	"github.com/hashicorp/consul-nia/config"
	"github.com/hashicorp/consul-nia/driver"
	"github.com/hashicorp/consul-nia/handler"
	"github.com/hashicorp/consul-nia/templates/tftmpl"
	"github.com/hashicorp/hcat"
)

// Controller describes the interface for monitoring Consul for relevant changes
// and triggering the driver to update network infrastructure.
type Controller interface {
	// Init initializes elements needed by controller
	Init(ctx context.Context) error

	// Run runs the controller by monitoring Consul and triggering the driver as needed
	Run(ctx context.Context) error
}

type Oncer interface {
	Once(ctx context.Context) error
}

func newDriverFunc(conf *config.Config) (func(*config.Config) driver.Driver, error) {
	if conf.Driver.Terraform != nil {
		log.Printf("[INFO] (controller) setting up Terraform driver")
		return newTerraformDriver, nil
	}
	return nil, errors.New("Unsupported driver")
}

func newTerraformDriver(conf *config.Config) driver.Driver {
	tfConf := *conf.Driver.Terraform
	return driver.NewTerraform(&driver.TerraformConfig{
		Log:               *tfConf.Log,
		PersistLog:        *tfConf.PersistLog,
		Path:              *tfConf.Path,
		WorkingDir:        *tfConf.WorkingDir,
		Backend:           tfConf.Backend,
		RequiredProviders: tfConf.RequiredProviders,
		ClientType:        *conf.ClientType,
	})
}

// newDriverTasks converts user-defined task configurations to the task object
// used by drivers.
func newDriverTasks(conf *config.Config) []driver.Task {
	if conf == nil {
		return []driver.Task{}
	}
	tasks := make([]driver.Task, len(*conf.Tasks))
	for i, t := range *conf.Tasks {

		services := make([]driver.Service, len(t.Services))
		for si, service := range t.Services {
			services[si] = getService(conf.Services, service)
		}

		providers := make([]map[string]interface{}, len(t.Providers))
		providerInfo := make(map[string]interface{})
		for pi, providerID := range t.Providers {
			providers[pi] = getProvider(conf.Providers, providerID)

			// This is Terraform specific to pass version and source info for
			// providers from the required_provider block
			name, _ := splitProviderID(providerID)
			if tfConf := conf.Driver.Terraform; tfConf != nil {
				if pInfo, ok := tfConf.RequiredProviders[name]; ok {
					providerInfo[name] = pInfo
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
			VarFiles:     t.VarFiles,
			Version:      *t.Version,
		}
	}

	return tasks
}

// newTaskTemplate creates templates to be monitored and rendered.
func newTaskTemplate(taskName string, conf *config.Config, fileReader func(string) ([]byte, error)) (template, error) {
	if conf.Driver.Terraform == nil {
		return nil, errors.New("unsupported driver to run tasks")
	}

	tmplFullpath := path.Join(*conf.Driver.Terraform.WorkingDir, taskName, tftmpl.TFVarsTmplFilename)
	tfvarsFilepath := strings.TrimRight(tmplFullpath, ".tmpl")

	content, err := fileReader(tmplFullpath)
	if err != nil {
		return nil, err
	}

	renderer := hcat.NewFileRenderer(hcat.FileRendererInput{
		Path: tfvarsFilepath,
	})

	return hcat.NewTemplate(hcat.TemplateInput{
		Contents:     string(content),
		Renderer:     renderer,
		FuncMapMerge: tftmpl.HCLTmplFuncMap,
	}), nil
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

func splitProviderID(id string) (string, string) {
	var name, alias string
	split := strings.SplitN(id, ".", 2)
	if len(split) == 2 {
		name, alias = split[0], split[1]
	} else {
		name = id
	}
	return name, alias
}

// getProvider is a helper to find and convert a user-defined provider
// configuration by the provider ID, which is either the provider name
// or <name>.<alias>. If a provider is not explicitly configured, it
// assumes the default provider block that is empty.
//
// provider "name" { }
func getProvider(providers *config.ProviderConfigs, id string) map[string]interface{} {
	name, alias := splitProviderID(id)

	for _, p := range *providers {
		// Find the provider by name
		if pRaw, ok := (*p)[name]; ok {
			if alias == "" {
				return *p
			}

			// Find the provider by alias
			pConf, ok := pRaw.(map[string]interface{})
			if !ok {
				// Not much we can do if the provider block has unexpected structure
				// at this point. We'll move forward with the default empty block.
				break
			}

			a, ok := pConf["alias"].(string)
			if ok && a == alias {
				return *p
			}
		}
	}

	return map[string]interface{}{name: make(map[string]interface{})}
}

// getPostApplyHandlers returns the first handler in a chain of handlers to be
// called post-apply for a type of driver.
//
// Returned handler may be nil even if returned err is nil. This happens when
// no providers have a handler.
func getPostApplyHandlers(conf *config.Config) (handler.Handler, error) {
	if conf.Driver.Terraform != nil {
		return getTerraformHandlers(conf)
	}
	return nil, errors.New("Unsupported driver")
}

// getTerraformHandlers returns the first handler in a chain of handlers
// for a Terraform driver.
//
// Returned handler may be nil even if returned err is nil. This happens when
// no providers have a handler.
func getTerraformHandlers(conf *config.Config) (handler.Handler, error) {
	counter := 0
	var next handler.Handler = nil
	for _, p := range *conf.Providers {
		for k, v := range *p {
			h, err := handler.TerraformProviderHandler(k, v)
			if err != nil {
				log.Printf(
					"[ERR] (controller) could not initialize handler for provider '%s': %s", k, err)
				return nil, err
			}
			if h != nil {
				counter++
				log.Printf(
					"[DEBUG] (controller) retrieved handler for provider '%s'", k)
				h.SetNext(next)
				next = h
			}
		}
	}
	log.Printf("[INFO] (controller) retrieved %d Terraform handlers", counter)
	return next, nil
}
