package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/hcat"
)

type baseController struct {
	newDriver func(*config.Config, *driver.Task, templates.Watcher) (driver.Driver, error)
	drivers   *driver.Drivers
	watcher   templates.Watcher
	resolver  templates.Resolver
	logger    logging.Logger
	providers []driver.TerraformProviderBlock

	// config that CTS is initialized with i.e. only used by base controller.
	// subsequent access to the configs should be through the state store.
	initConf *config.Config
}

func newBaseController(conf *config.Config, watcher templates.Watcher) (*baseController, error) {
	nd, err := newDriverFunc(conf)
	if err != nil {
		return nil, err
	}

	logger := logging.Global().Named(ctrlSystemName)

	return &baseController{
		newDriver: nd,
		drivers:   driver.NewDrivers(),
		watcher:   watcher,
		resolver:  hcat.NewResolver(),
		logger:    logger,
		initConf:  conf,
	}, nil
}

func (ctrl *baseController) init(ctx context.Context) error {
	ctrl.logger.Info("initializing driver")

	// Load provider configuration and evaluate dynamic values
	var err error
	ctrl.providers, err = ctrl.loadProviderConfigs(ctx)
	if err != nil {
		return err
	}

	// Future: improve by combining tasks into workflows.
	ctrl.logger.Info("initializing all tasks")
	ctrl.drivers.Reset()

	// Create and initialize task drivers
	for _, t := range *ctrl.initConf.Tasks {
		select {
		case <-ctx.Done():
			// Stop initializing remaining tasks if context has stopped.
			return ctx.Err()
		default:
		}

		d, err := ctrl.makeDriver(ctx, ctrl.initConf, *t)
		if err != nil {
			return err
		}

		taskName := *t.Name
		err = ctrl.drivers.Add(taskName, d)
		if err != nil {
			ctrl.logger.Error("error adding task driver to drivers list", taskNameLogKey, taskName)
			return err
		}
	}

	ctrl.logger.Info("drivers initialized")
	return nil
}

func (ctrl *baseController) makeDriver(ctx context.Context, conf *config.Config,
	taskConf config.TaskConfig) (driver.Driver, error) {

	taskName := *taskConf.Name
	logger := ctrl.logger.With(taskNameLogKey, taskName)

	d, err := ctrl.createNewTaskDriver(conf, taskConf)
	if err != nil {
		logger.Error("error creating new task driver")
		return nil, err
	}

	// Using the newly created driver, initialize the task
	err = d.InitTask(ctx)
	if err != nil {
		logger.Error("error initializing task")

		// Cleanup the task
		d.DestroyTask(ctx)
		logger.Debug("cleaned up task that errored initializing")
		return nil, err
	}

	logger.Trace("driver initialized")
	return d, nil
}

func (ctrl *baseController) createNewTaskDriver(conf *config.Config, taskConfig config.TaskConfig) (driver.Driver, error) {
	logger := ctrl.logger.With("task_name", *taskConfig.Name)
	logger.Trace("creating new task driver")
	task, err := newDriverTask(conf, &taskConfig, ctrl.providers)
	if err != nil {
		return nil, err
	}

	d, err := ctrl.newDriver(conf, task, ctrl.watcher)
	if err != nil {
		return nil, err
	}

	logger.Trace("driver created")
	return d, nil
}

// loadProviderConfigs loads provider configs and evaluates provider blocks
// for dynamic values in parallel.
func (ctrl *baseController) loadProviderConfigs(ctx context.Context) ([]driver.TerraformProviderBlock, error) {
	numBlocks := len(*ctrl.initConf.TerraformProviders)
	var wg sync.WaitGroup
	wg.Add(numBlocks)

	var lastErr error
	providerConfigs := make([]driver.TerraformProviderBlock, numBlocks)
	for i, providerConf := range *ctrl.initConf.TerraformProviders {
		go func(i int, initConf map[string]interface{}) {
			ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			defer wg.Done()

			block, err := hcltmpl.LoadDynamicConfig(ctxTimeout, ctrl.watcher, ctrl.resolver, initConf)
			if err != nil {
				ctrl.logger.Error("error loading dynamic configuration for provider",
					"provider", block.Name, "error", err)
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

func newDriverTask(conf *config.Config, taskConfig *config.TaskConfig,
	providerConfigs driver.TerraformProviderBlocks) (*driver.Task, error) {
	if conf == nil || conf.Driver == nil {
		// only expected for testing
		return nil, nil
	}

	meta := conf.DeprecatedServices.CTSUserDefinedMeta(taskConfig.DeprecatedServices)
	services := make([]driver.Service, len(taskConfig.DeprecatedServices))
	for si, service := range taskConfig.DeprecatedServices {
		services[si] = getService(conf.DeprecatedServices, service, meta)
	}

	tfConf := conf.Driver.Terraform

	providers := make(driver.TerraformProviderBlocks, len(taskConfig.Providers))
	providerInfo := make(map[string]interface{})
	for pi, providerID := range taskConfig.Providers {
		providers[pi] = getProvider(providerConfigs, providerID)

		// This is Terraform specific to pass version and source info for
		// providers from the required_provider block
		name, _ := splitProviderID(providerID)
		if tfConf != nil {
			if pInfo, ok := tfConf.RequiredProviders[name]; ok {
				providerInfo[name] = pInfo
			}
		}
	}

	var bp *driver.BufferPeriod // nil if disabled
	if *taskConfig.BufferPeriod.Enabled {
		bp = &driver.BufferPeriod{
			Min: *taskConfig.BufferPeriod.Min,
			Max: *taskConfig.BufferPeriod.Max,
		}
	}

	task, err := driver.NewTask(driver.TaskConfig{
		Description:  *taskConfig.Description,
		Name:         *taskConfig.Name,
		Enabled:      *taskConfig.Enabled,
		Env:          buildTaskEnv(conf, providers.Env()),
		Providers:    providers,
		ProviderInfo: providerInfo,
		Services:     services,
		Module:       *taskConfig.Module,
		VarFiles:     taskConfig.VarFiles,
		Version:      *taskConfig.Version,
		Variables:    taskConfig.Variables,
		BufferPeriod: bp,
		Condition:    taskConfig.Condition,
		ModuleInputs: *taskConfig.ModuleInputs,
		WorkingDir:   *taskConfig.WorkingDir,

		// Enterprise
		DeprecatedTFVersion: *taskConfig.DeprecatedTFVersion,
		TFCWorkspace:        *taskConfig.TFCWorkspace,
	})
	if err != nil {
		return nil, fmt.Errorf("error initializing task %s: %s", *taskConfig.Name, err)
	}

	return task, nil
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
