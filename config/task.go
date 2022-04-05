package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

const (
	taskSubsystemName = "task"
)

// TaskConfig is the configuration for a Sync task. This block may be
// specified multiple times to configure multiple tasks.
type TaskConfig struct {
	// Description is a human readable text to describe the task.
	Description *string `mapstructure:"description"`

	// Name is the unique name of the task.
	Name *string `mapstructure:"name"`

	// Providers is the list of provider names the task is dependent on. This is
	// used to map provider configuration to the task.
	Providers []string `mapstructure:"providers"`

	// DeprecatedServices is the list of service IDs or logical service names the task
	// executes on. Sync monitors the Consul Catalog for changes to these
	// services and triggers the task to run. Any service value not explicitly
	// defined by a `service` block with a matching ID is assumed to be a logical
	// service name in the default namespace.
	// - Deprecated in 0.5. Use names field of `condition "services"` and
	// `module_input "services"` instead
	DeprecatedServices []string `mapstructure:"services"`

	// Module is the path to fetch the Terraform Module (local or remote).
	// Previously named Source - Deprecated in 0.5
	Module           *string `mapstructure:"module"`
	DeprecatedSource *string `mapstructure:"source"`

	// ModuleInputs defines the Consul objects (e.g. services, kv) whose values
	// are provided as the task module's input variables.
	// Previously named SourceInput - Deprecated in 0.5
	ModuleInputs           *ModuleInputConfigs `mapstructure:"module_input"`
	DeprecatedSourceInputs *ModuleInputConfigs `mapstructure:"source_input"`

	// VarFiles is a list of paths to files containing variables for the
	// task. For the Terraform driver, these are files ending in `.tfvars` and
	// are used as Terraform input variables passed as arguments to the Terraform
	// module. Variables are loaded in the same order as they appear in the order
	// of the files. Duplicate variables are overwritten with the later value.
	VarFiles []string `mapstructure:"variable_files"`

	// TODO: Not supported by config file yet
	// TODO: Add validation
	Variables map[string]string

	// Version is the module version for the task to use. The latest version
	// will be used as the default if omitted.
	Version *string `mapstructure:"version"`

	// The Terraform client version to use for the task when configured with CTS
	// enterprise and the Terraform Cloud driver. This option is not supported
	// when using CTS OSS or the Terraform driver.
	TFVersion *string `mapstructure:"terraform_version"`

	// The workspace configurations to use for the task when configured with CTS
	// enterprise and the Terraform Cloud driver. This option is not supported
	// when using CTS OSS or the Terraform driver.
	TFCWorkspace *TerraformCloudWorkspaceConfig `mapstructure:"terraform_cloud_workspace"`

	// BufferPeriod configures per-task buffer timers.
	BufferPeriod *BufferPeriodConfig `mapstructure:"buffer_period"`

	// Enabled determines if the task is enabled or not. Enabled by default.
	// If not enabled, this task will not make any changes to resources.
	Enabled *bool `mapstructure:"enabled"`

	// Condition optionally configures a single run condition under which the
	// task will start executing
	Condition ConditionConfig `mapstructure:"condition"`

	// The local working directory for CTS to manage Terraform configuration
	// files and artifacts that are generated for the task. The default option
	// will create a child directory with the task name in the global working
	// directory.
	WorkingDir *string `mapstructure:"working_dir"`
}

// TaskConfigs is a collection of TaskConfig
type TaskConfigs []*TaskConfig

// Copy returns a deep copy of this configuration.
func (c *TaskConfig) Copy() *TaskConfig {
	if c == nil {
		return nil
	}

	var o TaskConfig
	o.Description = StringCopy(c.Description)
	o.Name = StringCopy(c.Name)

	o.Providers = append(o.Providers, c.Providers...)

	o.DeprecatedServices = append(o.DeprecatedServices, c.DeprecatedServices...)

	o.Module = StringCopy(c.Module)
	o.DeprecatedSource = StringCopy(c.DeprecatedSource)

	o.ModuleInputs = c.ModuleInputs.Copy()
	o.DeprecatedSourceInputs = c.DeprecatedSourceInputs.Copy()

	o.VarFiles = append(o.VarFiles, c.VarFiles...)

	if c.Variables != nil {
		o.Variables = make(map[string]string)
		for k, v := range c.Variables {
			o.Variables[k] = v
		}
	}

	o.Version = StringCopy(c.Version)

	o.TFVersion = StringCopy(c.TFVersion)

	if c.TFCWorkspace != nil {
		o.TFCWorkspace = c.TFCWorkspace.Copy()
	}

	o.BufferPeriod = c.BufferPeriod.Copy()

	o.Enabled = BoolCopy(c.Enabled)

	if !isConditionNil(c.Condition) {
		o.Condition = c.Condition.Copy()
	}

	if c.WorkingDir != nil {
		o.WorkingDir = StringCopy(c.WorkingDir)
	}

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *TaskConfig) Merge(o *TaskConfig) *TaskConfig {
	if c == nil {
		if o == nil {
			return nil
		}
		return o.Copy()
	}

	if o == nil {
		return c.Copy()
	}

	r := c.Copy()

	if o.Description != nil {
		r.Description = StringCopy(o.Description)
	}

	if o.Name != nil {
		r.Name = StringCopy(o.Name)
	}

	r.Providers = append(r.Providers, o.Providers...)

	r.DeprecatedServices = append(r.DeprecatedServices, o.DeprecatedServices...)

	if o.Module != nil {
		r.Module = StringCopy(o.Module)
	}
	if o.DeprecatedSource != nil {
		r.DeprecatedSource = StringCopy(o.DeprecatedSource)
	}

	if o.ModuleInputs != nil {
		r.ModuleInputs = r.ModuleInputs.Merge(o.ModuleInputs)
	}
	if o.DeprecatedSourceInputs != nil {
		r.DeprecatedSourceInputs = r.DeprecatedSourceInputs.Merge(o.DeprecatedSourceInputs)
	}

	r.VarFiles = append(r.VarFiles, o.VarFiles...)

	for k, v := range o.Variables {
		r.Variables[k] = v
	}

	if o.Version != nil {
		r.Version = StringCopy(o.Version)
	}

	if o.TFVersion != nil {
		r.TFVersion = StringCopy(o.TFVersion)
	}

	if o.TFCWorkspace != nil {
		r.TFCWorkspace = r.TFCWorkspace.Merge(o.TFCWorkspace)
	}

	if o.BufferPeriod != nil {
		r.BufferPeriod = r.BufferPeriod.Merge(o.BufferPeriod)
	}

	if o.Enabled != nil {
		r.Enabled = BoolCopy(o.Enabled)
	}

	if !isConditionNil(o.Condition) {
		if isConditionNil(r.Condition) {
			r.Condition = o.Condition.Copy()
		} else {
			r.Condition = r.Condition.Merge(o.Condition)
		}
	}

	if o.WorkingDir != nil {
		r.WorkingDir = StringCopy(o.WorkingDir)
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *TaskConfig) Finalize(globalBp *BufferPeriodConfig, wd string) {
	if c == nil {
		return
	}
	logger := logging.Global().Named(logSystemName).Named(taskSubsystemName)

	if c.Description == nil {
		c.Description = String("")
	}

	if c.Name == nil {
		c.Name = String("")
	}

	if c.Providers == nil {
		c.Providers = []string{}
	}

	if c.DeprecatedServices == nil {
		c.DeprecatedServices = []string{}
	} else {
		logger.Warn(servicesFieldLogMsg)
	}

	if c.Module == nil {
		c.Module = String("")
	}
	if c.DeprecatedSource != nil && *c.DeprecatedSource != "" {
		logger.Warn(sourceFieldLogMsg)
		if *c.Module != "" {
			logger.Warn("the task block's 'source' and 'module' field were both "+
				"configured. Defaulting to the 'module' value", "module", *c.Module)
		} else {
			// Merge Source with Module and use Module onwards
			c.Module = c.DeprecatedSource
		}
	}

	if c.VarFiles == nil {
		c.VarFiles = []string{}
	}

	if c.Variables == nil {
		c.Variables = make(map[string]string)
	}

	if c.Version == nil {
		c.Version = String("")
	}

	if c.TFCWorkspace == nil {
		c.TFCWorkspace = &TerraformCloudWorkspaceConfig{}
	}
	c.TFCWorkspace.Finalize()

	if c.TFVersion == nil {
		c.TFVersion = String("")
	}

	bp := globalBp
	if _, ok := c.Condition.(*ScheduleConditionConfig); ok {
		// disable buffer_period for schedule condition
		if c.BufferPeriod != nil {
			logger.Warn("disabling buffer_period for schedule condition. "+
				"overriding buffer_period configured for this task",
				"task_name", StringVal(c.Name), "buffer_period", c.BufferPeriod.GoString())
		}
		bp = &BufferPeriodConfig{
			Enabled: Bool(false),
			Min:     TimeDuration(0 * time.Second),
			Max:     TimeDuration(0 * time.Second),
		}
	}
	if c.BufferPeriod == nil {
		c.BufferPeriod = bp
	}
	c.BufferPeriod.Finalize(bp)

	if c.Enabled == nil {
		c.Enabled = Bool(true)
	}

	if isConditionNil(c.Condition) {
		c.Condition = EmptyConditionConfig()
	}
	c.Condition.Finalize()

	if c.DeprecatedSourceInputs != nil {
		if len(*c.DeprecatedSourceInputs) > 0 {
			logger.Warn(sourceInputBlockLogMsg)
			c.ModuleInputs = c.ModuleInputs.Merge(c.DeprecatedSourceInputs)
		}

		c.DeprecatedSourceInputs = nil
	}
	if c.ModuleInputs == nil {
		c.ModuleInputs = DefaultModuleInputConfigs()
	}
	c.ModuleInputs.Finalize()

	if c.WorkingDir == nil {
		c.WorkingDir = String(filepath.Join(wd, *c.Name))
	}
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *TaskConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("missing task configuration")
	}

	if c.Name == nil || len(*c.Name) == 0 {
		return fmt.Errorf("unique name for the task is required")
	}

	// For the Terraform driver, the task name is used as the module local name.
	// We'll validate early resembling Terraform restrictions to surface any errors
	// before a task is ran.
	if !hclsyntax.ValidIdentifier(*c.Name) {
		return fmt.Errorf("a task name must start with a letter or underscore and "+
			"may contain only letters, digits, underscores, and dashes: %q", *c.Name)
	}

	err := c.validateCondition()
	if err != nil {
		return err
	}

	if c.Module == nil || len(*c.Module) == 0 {
		return fmt.Errorf("module for the task is required")
	}

	if c.TFVersion != nil && *c.TFVersion != "" {
		return fmt.Errorf("unsupported configuration 'terraform_version' for "+
			"task %q. This option is available for Consul-Terraform-Sync Enterprise "+
			"when using the Terraform Cloud driver, or configure the Terraform client "+
			"version within the Terraform driver block", *c.Name)
	}

	if c.TFCWorkspace != nil && !c.TFCWorkspace.IsEmpty() {
		return fmt.Errorf("unsupported configuration 'terraform_cloud_workspace' for "+
			"task %q. This option is available for Consul-Terraform-Sync Enterprise "+
			"when using the Terraform Cloud driver", *c.Name)
	}

	// Restrict only one provider instance per task
	pNames := make(map[string]bool)
	for _, p := range c.Providers {
		name := strings.Split(p, ".")[0]
		if ok := pNames[name]; ok {
			return fmt.Errorf("only one provider instance per task")
		}
		pNames[name] = true
	}

	// TODO validate c.Variables

	if err := c.BufferPeriod.Validate(); err != nil {
		return err
	}

	if !isConditionNil(c.Condition) {
		if err := c.Condition.Validate(); err != nil {
			return err
		}
	}

	if err := c.ModuleInputs.Validate(c.DeprecatedServices, c.Condition); err != nil {
		return err
	}

	return nil
}

// GoString defines the printable version of this struct.
// Sensitive information is redacted.
func (c *TaskConfig) GoString() string {
	if c == nil {
		return "(*TaskConfig)(nil)"
	}

	return fmt.Sprintf("&TaskConfig{"+
		"Name:%s, "+
		"Description:%s, "+
		"Providers:%s, "+
		"Services (deprecated):%s, "+
		"Module:%s, "+
		"VarFiles:%s, "+
		"Version:%s, "+
		"TFVersion: %s, "+
		"BufferPeriod:%s, "+
		"Enabled:%t, "+
		"Condition:%s, "+
		"ModuleInput:%s"+
		"}",
		StringVal(c.Name),
		StringVal(c.Description),
		c.Providers,
		c.DeprecatedServices,
		StringVal(c.Module),
		c.VarFiles,
		StringVal(c.Version),
		StringVal(c.TFVersion),
		c.BufferPeriod.GoString(),
		BoolVal(c.Enabled),
		c.Condition.GoString(),
		c.ModuleInputs.GoString(),
	)
}

// DefaultTaskConfigs returns a configuration that is populated with the
// default values.
func DefaultTaskConfigs() *TaskConfigs {
	return &TaskConfigs{}
}

// Len is a helper method to get the length of the underlying config list
func (c *TaskConfigs) Len() int {
	if c == nil {
		return 0
	}

	return len(*c)
}

// Copy returns a deep copy of this configuration.
func (c *TaskConfigs) Copy() *TaskConfigs {
	if c == nil {
		return nil
	}

	o := make(TaskConfigs, c.Len())
	for i, t := range *c {
		o[i] = t.Copy()
	}
	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *TaskConfigs) Merge(o *TaskConfigs) *TaskConfigs {
	if c == nil {
		if o == nil {
			return nil
		}
		return o.Copy()
	}

	if o == nil {
		return c.Copy()
	}

	r := c.Copy()

	*r = append(*r, *o...)

	return r
}

// Finalize ensures the configuration has no nil pointers and sets default
// values.
func (c *TaskConfigs) Finalize(bp *BufferPeriodConfig, wd string) {
	if c == nil {
		*c = *DefaultTaskConfigs()
	}

	for _, t := range *c {
		t.Finalize(bp, wd)
	}
}

// Validate validates the values and nested values of the configuration struct
func (c *TaskConfigs) Validate() error {
	if c == nil || len(*c) == 0 {
		return fmt.Errorf("missing tasks configuration")
	}

	unique := make(map[string]bool)
	for _, t := range *c {
		if err := t.Validate(); err != nil {
			return err
		}

		taskName := *t.Name
		if _, ok := unique[taskName]; ok {
			return fmt.Errorf("duplicate task name: %s", taskName)
		}
		unique[taskName] = true
	}

	return nil
}

// GoString defines the printable version of this struct.
func (c *TaskConfigs) GoString() string {
	if c == nil {
		return "(*TaskConfigs)(nil)"
	}

	s := make([]string, len(*c))
	for i, t := range *c {
		s[i] = t.GoString()
	}

	return "{" + strings.Join(s, ", ") + "}"
}

// FilterTasks filters the task configurations by task name.
func FilterTasks(tasks *TaskConfigs, names []string) (*TaskConfigs, error) {
	allTasks := make(map[string]*TaskConfig)
	for _, t := range *tasks {
		allTasks[*t.Name] = t
	}

	var filtered TaskConfigs
	for _, name := range names {
		tconf, ok := allTasks[name]
		if !ok {
			return nil, fmt.Errorf("task not found: %s", name)
		}
		filtered = append(filtered, tconf)
	}
	return &filtered, nil
}

// validateCondition validates condition block taking into account services list
// - ensure task is configured with a condition (condition block or services
//   list)
// - if services list is configured, condition block's monitored variable type
// cannot be services
//
// Note: checking that a condition block's monitored variable type is different
// from module_input blocks is handled in ModuleInputConfigs.Validate()
func (c *TaskConfig) validateCondition() error {
	// Confirm task is configured with a condition
	if len(c.DeprecatedServices) == 0 {
		if isConditionNil(c.Condition) {
			// Error message omits task.services option since it is deprecated
			return fmt.Errorf("task should be configured with a condition block")
		}

		// task.services not configured. No need to worry about condition's
		// variable type
		return nil
	}

	// Confirm that condition's variable type is not services since task.services
	// is configured
	if _, ok := c.Condition.(*ServicesConditionConfig); ok {
		err := fmt.Errorf("task's `services` field and `condition " +
			"'services'` block both monitor \"services\" variable type. only " +
			"one of these can be configured per task")
		logging.Global().Named(logSystemName).Named(taskSubsystemName).
			Error("list of `services` and `condition 'services'` block cannot "+
				"both be configured. Consider combining the list into the "+
				"condition block or creating separate tasks",
				"task_name", StringVal(c.Name), "error", err)
		return err
	}
	return nil
}

// sourceFieldLogMsg is the log message for deprecating the `source` field.
const sourceFieldLogMsg = `the 'source' field in the task block is deprecated ` +
	`in v0.5.0 and will be removed in a future major version after v0.8.0.

Please replace 'source' with 'module' in your task configuration.

We will be releasing a tool to help upgrade your configuration for this deprecation.

Example upgrade:
|    task {
|  -   source =  "path/to/module"
|  +   module =  "path/to/module"
|      ...
|    }

For more details and examples, please see:
https://consul.io/docs/nia/release-notes/0-5-0#deprecate-source-field
`

// sourceInputBlockLogMsg is the log message for deprecating the `source_input`
// block.
const sourceInputBlockLogMsg = `the 'source_input' block in the task ` +
	`block is deprecated in v0.5.0 and will be removed in v0.8.0.

Please replace 'source_input' with 'module_input' in your task configuration.

We will be releasing a tool to help upgrade your configuration for this deprecation.

Example upgrade:
|    task {
|  -   source_input "<input-type>" {
|  +   module_input "<input-type>" {
|        ...
|      }
|      ...
|    }

For more details and examples, please see:
https://consul.io/docs/nia/release-notes/0-5-0#deprecate-source_input-block
`

// servicesFieldLogMsg is the log message for deprecating the `services` field.
const servicesFieldLogMsg = `the 'services' field in the task block is deprecated ` +
	`in v0.5.0 and will be removed in a future major version after v0.8.0.

Please replace 'services' in your task configuration with one of the options below:
 * condition "services": if there is _no_ preexisting condition block configured in your task
 * module_input "services": if there is a preexisting condition block configured in your task

We will be releasing a tool to help upgrade your configuration for this deprecation.

Example upgrade for a task with no preexisting condition block:
|    task {
|  -   services = ["api", "web"]
|  +   condition "services" {
|  +     names = ["api", "web"]
|  +   }
|      ...
|    }

For more details and additional examples, please see:
https://consul.io/docs/nia/release-notes/0-5-0#deprecate-services-field
`
