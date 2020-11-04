package controller

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl"
	"github.com/hashicorp/hcat"
)

const filePerm = os.FileMode(0640)

// Controller describes the interface for monitoring Consul for relevant changes
// and triggering the driver to update network infrastructure.
type Controller interface {
	// Init initializes elements needed by controller
	Init(ctx context.Context) error

	// Run runs the controller by monitoring Consul and triggering the driver as needed
	Run(ctx context.Context) error
}

// Oncer describes the interface a controller that can run in once mode
type Oncer interface {
	Once(ctx context.Context) error
}

// unit of work per template/task
type unit struct {
	taskName string
	driver   driver.Driver
	template template

	providers []string
	services  []string
	source    string
}

type baseController struct {
	conf       *config.Config
	newDriver  func(*config.Config, driver.Task) (driver.Driver, error)
	fileReader func(string) ([]byte, error)
	units      []unit
	watcher    watcher
	resolver   resolver
}

func newBaseController(conf *config.Config) (*baseController, error) {
	nd, err := newDriverFunc(conf)
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] (ctrl) initializing Consul client and testing connection")
	watcher, err := newWatcher(conf)
	if err != nil {
		return nil, err
	}

	return &baseController{
		conf:       conf,
		newDriver:  nd,
		fileReader: ioutil.ReadFile,
		watcher:    watcher,
		resolver:   hcat.NewResolver(),
	}, nil
}

func (ctrl *baseController) init(ctx context.Context) error {
	log.Printf("[INFO] (ctrl) initializing driver")

	// Future: improve by combining tasks into workflows.
	log.Printf("[INFO] (ctrl) initializing all tasks")
	tasks := newDriverTasks(ctrl.conf)
	units := make([]unit, 0, len(tasks))

	for _, task := range tasks {
		select {
		case <-ctx.Done():
			// Stop initializing remaining tasks if context has stopped.
			return ctx.Err()
		default:
		}

		log.Printf("[DEBUG] (ctrl) initializing task %q", task.Name)
		d, err := ctrl.newDriver(ctrl.conf, task)
		if err != nil {
			return err
		}

		err = d.InitTask(true)
		if err != nil {
			log.Printf("[ERR] (ctrl) error initializing task %q: %s", task.Name, err)
			return err
		}

		template, err := newTaskTemplate(task.Name, ctrl.conf, ctrl.fileReader)
		if err != nil {
			log.Printf("[ERR] (ctrl) error initializing template "+
				"for task %q: %s", task.Name, err)
			return err
		}

		units = append(units, unit{
			taskName:  task.Name,
			template:  template,
			driver:    d,
			providers: task.ProviderNames(),
			services:  task.ServiceNames(),
			source:    task.Source,
		})
	}
	ctrl.units = units

	log.Printf("[INFO] (ctrl) driver initialized")
	return nil
}

// InstallDriver installs necessary drivers based on user configuration.
func InstallDriver(ctx context.Context, conf *config.Config) error {
	if conf.Driver.Terraform != nil {
		return driver.InstallTerraform(ctx, conf.Driver.Terraform)
	}
	return errors.New("unsupported driver")
}

// newDriverFunc is a constructor abstraction for all of supported drivers
func newDriverFunc(conf *config.Config) (
	func(conf *config.Config, task driver.Task) (driver.Driver, error), error) {
	if conf.Driver.Terraform != nil {
		return newTerraformDriver, nil
	}
	return nil, errors.New("unsupported driver")
}

// newTerraformDriver maps user configuration to initialize a Terraform driver
// for a task
func newTerraformDriver(conf *config.Config, task driver.Task) (driver.Driver, error) {
	tfConf := *conf.Driver.Terraform
	return driver.NewTerraform(&driver.TerraformConfig{
		Task:              task,
		Log:               *tfConf.Log,
		PersistLog:        *tfConf.PersistLog,
		Path:              *tfConf.Path,
		WorkingDir:        filepath.Join(*tfConf.WorkingDir, task.Name),
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

	tmplFullpath := filepath.Join(*conf.Driver.Terraform.WorkingDir, taskName, tftmpl.TFVarsTmplFilename)
	tfvarsFilepath := filepath.Join(*conf.Driver.Terraform.WorkingDir, taskName, tftmpl.TFVarsFilename)

	content, err := fileReader(tmplFullpath)
	if err != nil {
		return nil, err
	}

	renderer := hcat.NewFileRenderer(hcat.FileRendererInput{
		Path:  tfvarsFilepath,
		Perms: filePerm,
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
