package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/hcat"
)

// Controller describes the interface for monitoring Consul for relevant changes
// and triggering the driver to update network infrastructure.
type Controller interface {
	// Init initializes elements needed by controller. Returns a map of
	// taskname to driver
	Init(context.Context) error

	// Run runs the controller by monitoring Consul and triggering the driver as needed
	Run(context.Context) error

	// ServeAPI runs the API server for the controller
	ServeAPI(context.Context) error

	// Stop stops underlying clients and connections
	Stop()
}

// Oncer describes the interface a controller that can run in once mode
type Oncer interface {
	Once(ctx context.Context) error
}

type baseController struct {
	conf      *config.Config
	newDriver func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error)
	drivers   *driver.Drivers
	watcher   templates.Watcher
	resolver  templates.Resolver
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
		conf:      conf,
		newDriver: nd,
		drivers:   driver.NewDrivers(),
		watcher:   watcher,
		resolver:  hcat.NewResolver(),
	}, nil
}

func (ctrl *baseController) Stop() {
	ctrl.watcher.Stop()
}

func (ctrl *baseController) init(ctx context.Context) error {
	log.Printf("[INFO] (ctrl) initializing driver")

	// Load provider configuration and evaluate dynamic values
	providerConfigs, err := ctrl.loadProviderConfigs(ctx)
	if err != nil {
		return err
	}

	// Future: improve by combining tasks into workflows.
	log.Printf("[INFO] (ctrl) initializing all tasks")
	tasks, err := newDriverTasks(ctrl.conf, providerConfigs)
	if err != nil {
		return err
	}

	ctrl.drivers.Reset()

	for _, task := range tasks {
		select {
		case <-ctx.Done():
			// Stop initializing remaining tasks if context has stopped.
			return ctx.Err()
		default:
		}

		taskName := task.Name()
		log.Printf("[DEBUG] (ctrl) initializing task %q", taskName)
		d, err := ctrl.newDriver(ctrl.conf, task, ctrl.watcher)
		if err != nil {
			return err
		}

		err = d.InitTask(ctx)
		if err != nil {
			log.Printf("[ERR] (ctrl) error initializing task %q", taskName)
			return err
		}

		ctrl.drivers.Add(taskName, d)
	}

	log.Printf("[INFO] (ctrl) driver initialized")
	return nil
}

// loadProviderConfigs loads provider configs and evaluates provider blocks
// for dynamic values in parallel.
func (ctrl *baseController) loadProviderConfigs(ctx context.Context) ([]driver.TerraformProviderBlock, error) {
	numBlocks := len(*ctrl.conf.TerraformProviders)
	var wg sync.WaitGroup
	wg.Add(numBlocks)

	var lastErr error
	providerConfigs := make([]driver.TerraformProviderBlock, numBlocks)
	for i, providerConf := range *ctrl.conf.TerraformProviders {
		go func(i int, conf map[string]interface{}) {
			ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			defer wg.Done()

			block, err := hcltmpl.LoadDynamicConfig(ctxTimeout, ctrl.watcher, ctrl.resolver, conf)
			if err != nil {
				log.Printf("[ERR] (ctrl) error loading dynamic configuration for provider %q: %s",
					block.Name, err)
				lastErr = err
				return
			}
			providerConfigs[i] = driver.NewTerraformProviderBlock(block)
		}(i, *providerConf)
	}

	wg.Wait()
	if lastErr != nil {
		return nil, lastErr
	}
	return providerConfigs, nil
}

// logDepSize logs the watcher dependency size every nth iteration. Set the
// iterator to a negative value to log each iteration.
func (ctrl *baseController) logDepSize(n uint, i int64) {
	depSize := ctrl.watcher.Size()
	if i%int64(n) == 0 || i < 0 {
		log.Printf("[DEBUG] (ctrl) watching %d dependencies", depSize)
		if depSize > templates.DepSizeWarning {
			log.Printf("[WARN] (ctrl) watching more than %d dependencies could "+
				"DDoS your Consul cluster: %d", templates.DepSizeWarning, depSize)
		}
	}
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
	func(conf *config.Config, task *driver.Task, w templates.Watcher) (driver.Driver, error), error) {
	if conf.Driver.Terraform != nil {
		return newTerraformDriver, nil
	}
	return nil, errors.New("unsupported driver")
}

// newTerraformDriver maps user configuration to initialize a Terraform driver
// for a task
func newTerraformDriver(conf *config.Config, task *driver.Task, w templates.Watcher) (driver.Driver, error) {
	tfConf := *conf.Driver.Terraform
	return driver.NewTerraform(&driver.TerraformConfig{
		Task:              task,
		Watcher:           w,
		Log:               *tfConf.Log,
		PersistLog:        *tfConf.PersistLog,
		Path:              *tfConf.Path,
		Backend:           tfConf.Backend,
		RequiredProviders: tfConf.RequiredProviders,
		ClientType:        *conf.ClientType,
	})
}

// newDriverTasks converts user-defined task configurations to the task object
// used by drivers.
func newDriverTasks(conf *config.Config, providerConfigs driver.TerraformProviderBlocks) ([]*driver.Task, error) {
	if conf == nil {
		return []*driver.Task{}, nil
	}

	tasks := make([]*driver.Task, len(*conf.Tasks))
	for i, t := range *conf.Tasks {

		meta := conf.Services.CTSUserDefinedMeta(t.Services)
		services := make([]driver.Service, len(t.Services))
		for si, service := range t.Services {
			services[si] = getService(conf.Services, service, meta)
		}

		providers := make(driver.TerraformProviderBlocks, len(t.Providers))
		providerInfo := make(map[string]interface{})
		for pi, providerID := range t.Providers {
			providers[pi] = getProvider(providerConfigs, providerID)

			// This is Terraform specific to pass version and source info for
			// providers from the required_provider block
			name, _ := splitProviderID(providerID)
			if tfConf := conf.Driver.Terraform; tfConf != nil {
				if pInfo, ok := tfConf.RequiredProviders[name]; ok {
					providerInfo[name] = pInfo
				}
			}
		}

		task, err := driver.NewTask(driver.TaskConfig{
			Description:  *t.Description,
			Name:         *t.Name,
			Enabled:      *t.Enabled,
			Env:          buildTaskEnv(conf, providers.Env()),
			Providers:    providers,
			ProviderInfo: providerInfo,
			Services:     services,
			Source:       *t.Source,
			VarFiles:     t.VarFiles,
			Version:      *t.Version,
			BufferPeriod: getTemplateBufferPeriod(conf, t),
			Condition:    t.Condition,
			WorkingDir:   *t.WorkingDir,
		})
		if err != nil {
			return nil, fmt.Errorf("error initializing task %s: %s", *t.Name, err)
		}
		tasks[i] = task
	}

	return tasks, nil
}

// getService is a helper to find and convert a user-defined service
// configuration by ID to a driver service type. If a service is not
// explicitly configured, it assumes the service is a logical service name
// in the default namespace.
func getService(services *config.ServiceConfigs, id string, meta config.ServicesMeta) driver.Service {
	for _, s := range *services {
		if *s.ID == id {
			return driver.Service{
				Datacenter:      *s.Datacenter,
				Description:     *s.Description,
				Name:            *s.Name,
				Namespace:       *s.Namespace,
				Tag:             *s.Tag,
				Filter:          *s.Filter,
				UserDefinedMeta: meta[*s.Name],
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
// terraform_provider "name" { }
func getProvider(providers driver.TerraformProviderBlocks, id string) driver.TerraformProviderBlock {
	name, alias := splitProviderID(id)

	for _, p := range providers {
		// Find the provider by name
		if p.Name() != name {
			continue
		}

		if alias == "" {
			return p
		}

		// Match by alias
		a, ok := p.ProviderBlock().Variables["alias"]
		if ok && a.AsString() == alias {
			return p
		}
	}

	return driver.NewTerraformProviderBlock(hcltmpl.NamedBlock{
		Name:      name,
		Variables: make(hcltmpl.Variables),
	})
}

// getTemplateBufferPeriod applies the task buffer period config to its template
func getTemplateBufferPeriod(conf *config.Config,
	taskConfig *config.TaskConfig) *driver.BufferPeriod {

	if buffPeriod := taskConfig.BufferPeriod; *buffPeriod.Enabled {
		return &driver.BufferPeriod{
			Min: *buffPeriod.Min,
			Max: *buffPeriod.Max,
		}
	}

	if conf == nil {
		return nil
	}

	// Set default buffer period
	if buffPeriod := conf.BufferPeriod; *buffPeriod.Enabled {
		return &driver.BufferPeriod{
			Min: *buffPeriod.Min,
			Max: *buffPeriod.Max,
		}
	}

	return nil
}

func buildTaskEnv(conf *config.Config, customEnv map[string]string) map[string]string {
	consulEnv := conf.Consul.Env()
	if len(customEnv) == 0 && len(consulEnv) == 0 {
		return nil
	}

	// Merge the Consul environment if Consul KV is used as the Terraform backend
	env := make(map[string]string)
	if conf.Driver.Terraform.IsConsulBackend() {
		for k, v := range consulEnv {
			env[k] = v
		}
	}

	for k, v := range customEnv {
		env[k] = v
	}

	return env
}
