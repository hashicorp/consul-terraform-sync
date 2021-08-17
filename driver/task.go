package driver

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/consul-terraform-sync/client"
	"github.com/hashicorp/consul-terraform-sync/config"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/client"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl"
)

const (
	// RunOptionNow runs the task immediate (now) once the task has been updated
	RunOptionNow = "now"
	// RunOptionInspect does a dry-run task update and returns dry-run info
	RunOptionInspect = "inspect"
)

// PatchTask holds the information to patch update a task. It will only include
// fields that we support updating at this time
type PatchTask struct {
	// RunOption is a set of options on how to handle the patch update
	// current options are "now" and "inspect". See constants for more details
	RunOption string

	Enabled bool
}

// Service contains service configuration information
type Service struct {
	Datacenter      string
	Description     string
	Name            string
	Namespace       string
	Tag             string
	Filter          string
	UserDefinedMeta map[string]string
}

// BufferPeriod contains the task's buffer period configuration information
// if enabled
type BufferPeriod struct {
	Min time.Duration
	Max time.Duration
}

// Task contains task configuration information
type Task struct {
	mu sync.RWMutex

	description  string
	name         string
	enabled      bool
	env          map[string]string
	providers    TerraformProviderBlocks // task.providers config info
	providerInfo map[string]interface{}  // driver.required_provider config info
	services     []Service
	source       string
	varFiles     []string
	variables    hcltmpl.Variables // loaded variables from varFiles
	version      string
	bufferPeriod *BufferPeriod // nil when disabled
	condition    config.ConditionConfig
	workingDir   string
}

type TaskConfig struct {
	Description  string
	Name         string
	Enabled      bool
	Env          map[string]string
	Providers    TerraformProviderBlocks
	ProviderInfo map[string]interface{}
	Services     []Service
	Source       string
	VarFiles     []string
	Version      string
	BufferPeriod *BufferPeriod
	Condition    config.ConditionConfig
	WorkingDir   string
}

func NewTask(conf TaskConfig) (*Task, error) {
	loadedVars := make(hcltmpl.Variables)
	for _, vf := range conf.VarFiles {
		tfvars, err := tftmpl.LoadModuleVariables(vf)
		if err != nil {
			return nil, err
		}

		for k, v := range tfvars {
			loadedVars[k] = v
		}
	}
	return &Task{
		description:  conf.Description,
		name:         conf.Name,
		enabled:      conf.Enabled,
		env:          conf.Env,
		providers:    conf.Providers,
		providerInfo: conf.ProviderInfo,
		services:     conf.Services,
		source:       conf.Source,
		varFiles:     conf.VarFiles,
		variables:    loadedVars,
		version:      conf.Version,
		bufferPeriod: conf.BufferPeriod,
		condition:    conf.Condition,
		workingDir:   conf.WorkingDir,
	}, nil
}

// BufferPeriod returns a copy of the buffer period. If the buffer
// period is not configured, the second parameter returns false.
func (t *Task) BufferPeriod() (BufferPeriod, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.bufferPeriod == nil {
		return BufferPeriod{}, false
	}
	return *t.bufferPeriod, true
}

// ConditionType returns the type of condition for the task to run
func (t *Task) Condition() config.ConditionConfig {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.condition
}

// Description returns the task description
func (t *Task) Description() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.description
}

// Name returns the task name
func (t *Task) Name() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.name
}

// IsEnabled returns whether the task is enabled or disabled
func (t *Task) IsEnabled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.enabled
}

// Enable sets the task as enabled
func (t *Task) Enable() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = true
}

// Disable sets the task as disabled
func (t *Task) Disable() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = false
}

// Env returns a copy of task environment variables
func (t *Task) Env() map[string]string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	env := make(map[string]string)
	for k, v := range t.env {
		env[k] = v
	}
	return env
}

// ProviderNames returns the list of providers that the task has configured
func (t *Task) Providers() TerraformProviderBlocks {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.providers.Copy()
}

// ProviderNames returns the list of providers that the task has configured
func (t *Task) ProviderNames() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	names := make([]string, len(t.providers))
	for ix, p := range t.providers {
		names[ix] = p.Name()
	}
	return names
}

// Services returns a copy of the list of services that the task has configured
func (t *Task) Services() []Service {
	t.mu.RLock()
	defer t.mu.RUnlock()

	services := make([]Service, len(t.services))
	for i, s := range t.services {
		services[i] = s.Copy()
	}
	return services
}

// ServiceNames returns the list of services that the task has configured
func (t *Task) ServiceNames() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	names := make([]string, len(t.services))
	for ix, s := range t.services {
		names[ix] = s.Name
	}
	return names
}

// Source returns the module source for the task
func (t *Task) Source() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.source
}

// VariableFiles returns a copy of the list of configured variable files
// for the task's module.
func (t *Task) VariableFiles() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	varFiles := make([]string, len(t.varFiles))
	for i, vf := range t.varFiles {
		varFiles[i] = vf
	}
	return varFiles
}

// Variables returns a copy of the loaded input variables for a module
// from configured variable files.
func (t *Task) Variables() hcltmpl.Variables {
	t.mu.RLock()
	defer t.mu.RUnlock()

	vars := make(hcltmpl.Variables)
	for k, v := range t.variables {
		vars[k] = v
	}
	return vars
}

// Version returns the configured version for the module of the task
func (t *Task) Version() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.version
}

// WorkingDir returns the working directory to manage generated artifacts for
// the task.
func (t *Task) WorkingDir() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.workingDir
}

func (s Service) Copy() Service {
	// All other Service attributes are simple types, this sets the meta to a new
	// copy of the map
	meta := make(map[string]string)
	for k, v := range s.UserDefinedMeta {
		meta[k] = v
	}
	copy := s
	copy.UserDefinedMeta = meta
	return copy
}

// configureRootModuleInput sets task values for the module input.
func (t *Task) configureRootModuleInput(input *tftmpl.RootModuleInputData) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	input.Task = tftmpl.Task{
		Description: t.description,
		Name:        t.name,
		Source:      t.source,
		Version:     t.version,
	}

	input.Services = make([]tftmpl.Service, len(t.services))
	for i, s := range t.services {
		input.Services[i] = tftmpl.Service{
			Datacenter:         s.Datacenter,
			Description:        s.Description,
			Name:               s.Name,
			Namespace:          s.Namespace,
			Tag:                s.Tag,
			Filter:             s.Filter,
			CTSUserDefinedMeta: s.UserDefinedMeta,
		}
	}

	var condition tftmpl.Condition
	switch v := t.condition.(type) {
	case *config.CatalogServicesConditionConfig:
		condition = &tftmpl.CatalogServicesCondition{
			Regexp:            *v.Regexp,
			SourceIncludesVar: *v.SourceIncludesVar,
			Datacenter:        *v.Datacenter,
			Namespace:         *v.Namespace,
			NodeMeta:          v.NodeMeta,
		}
	case *config.ServicesConditionConfig:
		condition = &tftmpl.ServicesCondition{
			Regexp: *v.Regexp,
		}
	case *config.NodesConditionConfig:
		condition = &tftmpl.NodesCondition{
			Datacenter: *v.Datacenter,
		}
	default:
		// expected only for test scenarios
		log.Printf("[WARN] (driver.terraform) task '%s' condition config unset."+
			" defaulting to services condition", t.name)
		condition = &tftmpl.ServicesCondition{}
	}
	input.Condition = condition

	input.Providers = t.providers.ProviderBlocks()
	input.ProviderInfo = make(map[string]interface{})
	for k, v := range t.providerInfo {
		input.ProviderInfo[k] = v
	}

	input.Variables = make(hcltmpl.Variables)
	for k, v := range t.variables {
		input.Variables[k] = v
	}
}

// clientConfig configures a driver client for a task
type clientConfig struct {
	clientType string
	log        bool
	taskName   string
	persistLog bool
	path       string
	varFiles   []string
	workingDir string
}

// newClient initializes a specific type of client given a task
func newClient(conf *clientConfig) (client.Client, error) {
	var err error
	var c client.Client
	taskName := conf.taskName

	switch conf.clientType {
	case developmentClient:
		log.Printf("[TRACE] (driver) creating development client for task '%s'", taskName)
		c, err = client.NewPrinter(&client.PrinterConfig{
			ExecPath:   conf.path,
			WorkingDir: conf.workingDir,
			Workspace:  taskName,
			Writer:     os.Stdout,
		})
	case testClient:
		log.Printf("[TRACE] (driver) creating mock client for task '%s'", taskName)
		c = new(mocks.Client)
	default:
		log.Printf("[TRACE] (driver) creating terraform cli client for task '%s'", taskName)
		c, err = client.NewTerraformCLI(&client.TerraformCLIConfig{
			Log:        conf.log,
			PersistLog: conf.persistLog,
			ExecPath:   conf.path,
			WorkingDir: conf.workingDir,
			Workspace:  taskName,
			VarFiles:   conf.varFiles,
		})
	}

	return c, err
}
