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

	// Services is the list of service IDs or logical service names the task
	// executes on. Sync monitors the Consul Catalog for changes to these
	// services and triggers the task to run. Any service value not explicitly
	// defined by a `service` block with a matching ID is assumed to be a logical
	// service name in the default namespace.
	Services []string `mapstructure:"services"`

	// Module is the path to fetch the Terraform Module (local or remote).
	// Previously named Source - Deprecated in 0.5
	Module           *string `mapstructure:"module"`
	DeprecatedSource *string `mapstructure:"source"`

	// ModuleInput defines the Consul objects (e.g. services, kv) whose values are
	// provided as the task module’s input variables.
	// Previously named SourceInput - Deprecated in 0.5
	ModuleInput           ModuleInputConfig `mapstructure:"module_input"`
	DeprecatedSourceInput ModuleInputConfig `mapstructure:"source_input"`

	// VarFiles is a list of paths to files containing variables for the
	// task. For the Terraform driver, these are files ending in `.tfvars` and
	// are used as Terraform input variables passed as arguments to the Terraform
	// module. Variables are loaded in the same order as they appear in the order
	// of the files. Duplicate variables are overwritten with the later value.
	VarFiles []string `mapstructure:"variable_files"`

	// TODO: Not supported by config file yet
	Variables map[string]string

	// Version is the version of source the task will use. For the Terraform
	// driver, this is the module version. The latest version will be used as
	// the default if omitted.
	Version *string `mapstructure:"version"`

	// The Terraform client version to use for the task when configured with CTS
	// enterprise and the Terraform Cloud driver. This option is not supported
	// when using CTS OSS or the Terraform driver.
	TFVersion *string `mapstructure:"terraform_version"`

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

	o.Services = append(o.Services, c.Services...)

	o.Module = StringCopy(c.Module)
	o.DeprecatedSource = StringCopy(c.DeprecatedSource)

	if !isModuleInputNil(c.ModuleInput) {
		o.ModuleInput = c.ModuleInput.Copy()
	}
	if !isModuleInputNil(c.DeprecatedSourceInput) {
		o.DeprecatedSourceInput = c.DeprecatedSourceInput.Copy()
	}

	o.VarFiles = append(o.VarFiles, c.VarFiles...)

	if c.Variables != nil {
		o.Variables = make(map[string]string)
		for k, v := range c.Variables {
			o.Variables[k] = v
		}
	}

	o.Version = StringCopy(c.Version)

	o.TFVersion = StringCopy(c.TFVersion)

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

	r.Services = append(r.Services, o.Services...)

	if o.Module != nil {
		r.Module = StringCopy(o.Module)
	}
	if o.DeprecatedSource != nil {
		r.DeprecatedSource = StringCopy(o.DeprecatedSource)
	}

	if !isModuleInputNil(o.ModuleInput) {
		if isModuleInputNil(r.ModuleInput) {
			r.ModuleInput = o.ModuleInput.Copy()
		} else {
			r.ModuleInput = r.ModuleInput.Merge(o.ModuleInput)
		}
	}
	if !isModuleInputNil(o.DeprecatedSourceInput) {
		if isModuleInputNil(r.DeprecatedSourceInput) {
			r.DeprecatedSourceInput = o.DeprecatedSourceInput.Copy()
		} else {
			r.DeprecatedSourceInput = r.DeprecatedSourceInput.Merge(o.DeprecatedSourceInput)
		}
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

	if c.Services == nil {
		c.Services = []string{}
	}

	if c.Module == nil {
		c.Module = String("")
	}
	if c.DeprecatedSource != nil && *c.DeprecatedSource != "" {
		logger.Warn("Task's 'source' field was marked for deprecation in v0.5.0. " +
			"Please update your configuration to use the 'module' field instead")
		if *c.Module != "" {
			logger.Warn("Task's 'source' and 'module' field were both "+
				"configured. Defaulting to 'module' value", "module", c.Module)
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

	if !isModuleInputNil(c.DeprecatedSourceInput) {
		logger.Warn("Task's 'source_input' block was marked for deprecation " +
			"in v0.5.0. Please update your configuration to use 'module_input'" +
			" instead.")

		if !isModuleInputNil(c.ModuleInput) {
			logger.Warn("Task is configured with both 'source_input' block and "+
				"'module_input' block. Defaulting to 'module_input' block's value",
				"module_input", c.ModuleInput)
		} else {
			// Merge SourceInput with ModuleInput. Use ModuleInput onwards
			c.ModuleInput = c.DeprecatedSourceInput
		}

	}
	if isModuleInputNil(c.ModuleInput) {
		c.ModuleInput = EmptyModuleInputConfig()
	}
	c.ModuleInput.Finalize()

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

	err = c.validateSourceInput()
	if err != nil {
		return err
	}

	if c.TFVersion != nil && *c.TFVersion != "" {
		return fmt.Errorf("unsupported configuration 'terraform_version' for "+
			"task %q. This option is available for Consul-Terraform-Sync Enterprise "+
			"when using the Terraform Cloud driver, or configure the Terraform client "+
			"version within the Terraform driver block", *c.Name)
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

	if c.Variables == nil {
		return fmt.Errorf("variables cannot be nil")
	}

	if err := c.BufferPeriod.Validate(); err != nil {
		return err
	}

	if !isConditionNil(c.Condition) {
		if err := c.Condition.Validate(); err != nil {
			return err
		}
	}

	if !isModuleInputNil(c.ModuleInput) {
		if err := c.ModuleInput.Validate(); err != nil {
			return err
		}
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
		"Services:%s, "+
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
		c.Services,
		StringVal(c.Module),
		c.VarFiles,
		StringVal(c.Version),
		StringVal(c.TFVersion),
		c.BufferPeriod.GoString(),
		BoolVal(c.Enabled),
		c.Condition.GoString(),
		c.ModuleInput.GoString(),
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

func (c *TaskConfig) validateCondition() error {
	if len(c.Services) == 0 {
		if isConditionNil(c.Condition) {
			return fmt.Errorf("at least one service or a condition must be " +
				"configured")
		}
		switch cond := c.Condition.(type) {
		case *CatalogServicesConditionConfig:
			if cond.Regexp == nil {
				return fmt.Errorf("catalog-services condition requires either" +
					"task.condition.regexp or at least one service in " +
					"task.services to be configured")
			}
		case *ConsulKVConditionConfig:
			return fmt.Errorf("consul-kv condition requires at least one service to " +
				"be configured in task.services")
		case *ScheduleConditionConfig:
			if isModuleInputNil(c.ModuleInput) || isModuleInputEmpty(c.ModuleInput) {
				return fmt.Errorf("schedule condition requires at least one service to " +
					"be configured in task.services or a module_input must be provided")
			}
		}
	} else {
		switch c.Condition.(type) {
		case *ServicesConditionConfig:
			err := fmt.Errorf("a task cannot be configured with both " +
				"`services` field and `module_input` block. only one can be " +
				"configured per task")
			logging.Global().Named(logSystemName).Named(taskSubsystemName).
				Error("list of services and service condition block both "+
					"provided. If both are needed, consider combining the "+
					"list into the condition block or creating separate tasks",
					"task_name", StringVal(c.Name), "error", err)
			return err
		}
	}

	return nil
}

func (c *TaskConfig) validateSourceInput() error {
	// For now only schedule condition allows for module_input, so a condition
	// of type ScheduleConditionConfig is the only supported type
	switch c.Condition.(type) {
	case *ScheduleConditionConfig:
		if len(c.Services) == 0 {
			switch c.ModuleInput.(type) {
			case *ConsulKVModuleInputConfig:
				return fmt.Errorf("consul-kv module_input requires at least one service to " +
					"be configured in task.services")
			}
		} else {
			switch c.ModuleInput.(type) {
			case *ServicesModuleInputConfig:
				err := fmt.Errorf("a task cannot be configured with both " +
					"`services` field and `condition` block. only one can be " +
					"configured per task")
				logging.Global().Named(logSystemName).Named(taskSubsystemName).
					Error("list of services and service condition block both "+
						"provided. If both are needed, consider combining the "+
						"list into the condition block or creating separate tasks",
						"task_name", StringVal(c.Name), "error", err)
				return err
			}
		}
	default:
		if !isModuleInputNil(c.ModuleInput) && !isModuleInputEmpty(c.ModuleInput) {
			return fmt.Errorf("module_input is only supported when a schedule condition is configured")
		}
	}

	return nil
}
