package driver

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/consul-terraform-sync/client"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
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
	variables    hcltmpl.Variables // loaded variables from varFiles
	version      string
	bufferPeriod *BufferPeriod // nil when disabled
	condition    config.ConditionConfig
	sourceInput  config.SourceInputConfig
	workingDir   string
	logger       logging.Logger
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
	Variables    map[string]string
	Version      string
	BufferPeriod *BufferPeriod
	Condition    config.ConditionConfig
	SourceInput  config.SourceInputConfig
	WorkingDir   string
}

func NewTask(conf TaskConfig) (*Task, error) {
	// Load all variables from passed in variable files
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

	// Load all variables from passed in variables map
	tfvars, err := tftmpl.ParseModuleVariablesFromMap(conf.Variables)
	if err != nil {
		return nil, err
	}
	for k, v := range tfvars {
		loadedVars[k] = v
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
		variables:    loadedVars,
		version:      conf.Version,
		bufferPeriod: conf.BufferPeriod,
		condition:    conf.Condition,
		sourceInput:  conf.SourceInput,
		workingDir:   conf.WorkingDir,
		logger:       logging.Global().Named(logSystemName),
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

// Condition returns the type of condition for the task to run
func (t *Task) Condition() config.ConditionConfig {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.condition
}

// SourceInput returns the type of sourceInput for the task to run
func (t *Task) SourceInput() config.SourceInputConfig {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.sourceInput
}

// IsScheduled returns if the task is a scheduled task or not (a dynamic task)
func (t *Task) IsScheduled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.condition.(*config.ScheduleConditionConfig)
	return ok
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
	cp := s
	cp.UserDefinedMeta = meta
	return cp
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
			Filter:             s.Filter,
			CTSUserDefinedMeta: s.UserDefinedMeta,
		}
	}

	var templates []tftmpl.Template
	var condition tftmpl.Template
	switch v := t.condition.(type) {
	case *config.CatalogServicesConditionConfig:
		condition = &tftmpl.CatalogServicesTemplate{
			Regexp:            *v.Regexp,
			Datacenter:        *v.Datacenter,
			Namespace:         *v.Namespace,
			NodeMeta:          v.NodeMeta,
			SourceIncludesVar: *v.SourceIncludesVar,
		}
	case *config.ServicesConditionConfig:
		if len(v.Names) > 0 {
			condition = &tftmpl.ServicesTemplate{
				Names:      v.Names,
				Datacenter: *v.Datacenter,
				Namespace:  *v.Namespace,
				Filter:     *v.Filter,
				// SourceIncludesVar=false not yet supported
				SourceIncludesVar: true,
			}
		} else {
			condition = &tftmpl.ServicesRegexTemplate{
				Regexp:     *v.Regexp,
				Datacenter: *v.Datacenter,
				Namespace:  *v.Namespace,
				Filter:     *v.Filter,
				// SourceIncludesVar=false not yet supported
				SourceIncludesVar: true,
			}
		}
	case *config.ConsulKVConditionConfig:
		condition = &tftmpl.ConsulKVTemplate{
			Path:              *v.Path,
			Datacenter:        *v.Datacenter,
			Recurse:           *v.Recurse,
			Namespace:         *v.Namespace,
			SourceIncludesVar: *v.SourceIncludesVar,
		}
	default:
		// no-op: condition block currently not required since services.list
		// can be used alternatively
	}

	if condition != nil {
		templates = append(templates, condition)
		t.logger.Trace("condition block template configured", "template_type",
			fmt.Sprintf("%T", condition))
	}

	var sourceInput tftmpl.Template
	switch v := t.sourceInput.(type) {
	case *config.ServicesSourceInputConfig:
		if len(v.Names) > 0 {
			sourceInput = &tftmpl.ServicesTemplate{
				Names:      v.Names,
				Datacenter: *v.Datacenter,
				Namespace:  *v.Namespace,
				Filter:     *v.Filter,
				// always include for source_input config
				SourceIncludesVar: true,
			}
		} else {
			sourceInput = &tftmpl.ServicesRegexTemplate{
				Regexp:     *v.Regexp,
				Datacenter: *v.Datacenter,
				Namespace:  *v.Namespace,
				Filter:     *v.Filter,
				// always include for source_input config
				SourceIncludesVar: true,
			}
		}
	case *config.ConsulKVSourceInputConfig:
		sourceInput = &tftmpl.ConsulKVTemplate{
			Path:       *v.Path,
			Datacenter: *v.Datacenter,
			Recurse:    *v.Recurse,
			Namespace:  *v.Namespace,
			// always include for source_input config
			SourceIncludesVar: true,
		}
	default:
		// no-op: source_input block config not required
	}

	if sourceInput != nil {
		templates = append(templates, sourceInput)
		t.logger.Trace("source_input block template configured", "template_type",
			fmt.Sprintf("%T", sourceInput))
	}

	input.Templates = templates

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
	workingDir string
}

// newClient initializes a specific type of client given a task
func newClient(conf *clientConfig) (client.Client, error) {
	var err error
	var c client.Client
	taskName := conf.taskName

	tnlog := logging.Global().Named(logSystemName).With(taskNameLogKey, taskName)
	switch conf.clientType {
	case developmentClient:
		tnlog.Trace("creating development client for task")
		c, err = client.NewPrinter(&client.PrinterConfig{
			ExecPath:   conf.path,
			WorkingDir: conf.workingDir,
			Workspace:  taskName,
			Writer:     os.Stdout,
		})
	case testClient:
		tnlog.Trace("creating mock client for task")
		c = new(mocks.Client)
	default:
		tnlog.Trace("creating terraform cli client for task")
		c, err = client.NewTerraformCLI(&client.TerraformCLIConfig{
			Log:        conf.log,
			PersistLog: conf.persistLog,
			ExecPath:   conf.path,
			WorkingDir: conf.workingDir,
			Workspace:  taskName,
		})
	}

	return c, err
}
