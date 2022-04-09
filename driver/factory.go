package driver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/hcat"
)

// Factory creates drivers from configuration
type Factory struct {
	logger logging.Logger

	newDriver func(*config.Config, *Task, templates.Watcher) (Driver, error)
	watcher   templates.Watcher
	resolver  templates.Resolver

	providers []TerraformProviderBlock

	// config that CTS is initialized with i.e. only used by base controller.
	// subsequent access to the configs should be through the state store.
	initConf *config.Config
}

func NewFactory(conf *config.Config, watcher templates.Watcher) (*Factory, error) {
	nd, err := newDriverFunc(conf)
	if err != nil {
		return nil, err
	}

	logger := logging.Global().Named(logSystemName)
	logger.Info("initializing Consul client and testing connection")

	return &Factory{
		newDriver: nd,
		logger:    logger,
		watcher:   watcher,
		resolver:  hcat.NewResolver(),
		initConf:  conf,
	}, nil
}

func (df *Factory) Init(ctx context.Context) error {
	var err error
	df.providers, err = df.initLoadProviderConfigs(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (df *Factory) Make(ctx context.Context, conf *config.Config, taskConf *config.TaskConfig) (Driver, error) {
	taskName := *taskConf.Name
	d, err := df.createNewTaskDriver(conf, taskConf)
	if err != nil {
		df.logger.Error("error creating new task driver", taskNameLogKey, taskName)
		return nil, err
	}

	// Using the newly created driver, initialize the task
	err = d.InitTask(ctx)
	if err != nil {
		df.logger.Error("error initializing task", taskNameLogKey, taskName)
		// Cleanup the task
		d.DestroyTask(ctx)
		df.logger.Error("task destroyed", taskNameLogKey, taskName)
		return nil, err
	}

	return d, nil
}

// initLoadProviderConfigs loads provider configs and evaluates provider blocks
// for dynamic values in parallel.
func (df *Factory) initLoadProviderConfigs(ctx context.Context) ([]TerraformProviderBlock, error) {
	numBlocks := len(*df.initConf.TerraformProviders)
	var wg sync.WaitGroup
	wg.Add(numBlocks)

	var lastErr error
	providerConfigs := make([]TerraformProviderBlock, numBlocks)
	for i, providerConf := range *df.initConf.TerraformProviders {
		go func(i int, initConf map[string]interface{}) {
			ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			defer wg.Done()

			block, err := hcltmpl.LoadDynamicConfig(ctxTimeout, df.watcher, df.resolver, initConf)
			if err != nil {
				df.logger.Error("error loading dynamic configuration for provider",
					"provider", block.Name, "error", err)
				lastErr = err
				return
			}
			providerConfigs[i] = NewTerraformProviderBlock(block)
		}(i, *providerConf)
	}

	wg.Wait()
	if lastErr != nil {
		return nil, lastErr
	}
	return providerConfigs, nil
}

func (df *Factory) createNewTaskDriver(conf *config.Config, taskConfig *config.TaskConfig) (Driver, error) {
	logger := df.logger.With("task_name", *taskConfig.Name)
	logger.Trace("creating new task driver")
	task, err := newDriverTask(conf, taskConfig, df.providers)
	if err != nil {
		return nil, err
	}

	d, err := df.newDriver(conf, task, df.watcher)
	if err != nil {
		return nil, err
	}

	logger.Trace("driver created")
	return d, nil
}

func newDriverTask(conf *config.Config, taskConfig *config.TaskConfig,
	providerConfigs TerraformProviderBlocks) (*Task, error) {
	if conf == nil || conf.Driver == nil {
		// only expected for testing
		return nil, nil
	}

	meta := conf.DeprecatedServices.CTSUserDefinedMeta(taskConfig.DeprecatedServices)
	services := make([]Service, len(taskConfig.DeprecatedServices))
	for si, service := range taskConfig.DeprecatedServices {
		services[si] = getService(conf.DeprecatedServices, service, meta)
	}

	tfConf := conf.Driver.Terraform

	providers := make(TerraformProviderBlocks, len(taskConfig.Providers))
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

	var bp *BufferPeriod // nil if disabled
	if *taskConfig.BufferPeriod.Enabled {
		bp = &BufferPeriod{
			Min: *taskConfig.BufferPeriod.Min,
			Max: *taskConfig.BufferPeriod.Max,
		}
	}

	task, err := NewTask(TaskConfig{
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
		TFVersion:    *taskConfig.TFVersion,
		TFCWorkspace: *taskConfig.TFCWorkspace,
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
func getService(services *config.ServiceConfigs, id string, meta config.ServicesMeta) Service {
	for _, s := range *services {
		if *s.ID == id {
			return Service{
				Datacenter:      *s.Datacenter,
				Description:     *s.Description,
				Name:            *s.Name,
				Namespace:       *s.Namespace,
				Filter:          *s.Filter,
				UserDefinedMeta: meta[*s.Name],
			}
		}
	}

	return Service{Name: id}
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
func getProvider(providers TerraformProviderBlocks, id string) TerraformProviderBlock {
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

	return NewTerraformProviderBlock(hcltmpl.NamedBlock{
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

// newDriverFunc is a constructor abstraction for all of supported drivers
func newDriverFunc(conf *config.Config) (
	func(conf *config.Config, task *Task, w templates.Watcher) (Driver, error), error) {
	if conf.Driver.Terraform != nil {
		return newTerraformDriver, nil
	}
	return nil, errors.New("unsupported driver")
}

// newTerraformDriver maps user configuration to initialize a Terraform driver
// for a task
func newTerraformDriver(conf *config.Config, task *Task, w templates.Watcher) (Driver, error) {
	tfConf := *conf.Driver.Terraform
	return NewTerraform(&TerraformConfig{
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
