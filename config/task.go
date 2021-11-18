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

	// Source is the location the driver uses to fetch dependencies. The source
	// format is dependent on the driver. For the Terraform driver, the source
	// is the module path (local or remote).
	Source *string `mapstructure:"source"`

	// SourceInput defines the Consul objects (e.g. services, kv) whose values are
	// provided as the task source’s input variables
	SourceInput SourceInputConfig `mapstructure:"source_input"`

	// VarFiles is a list of paths to files containing variables for the
	// task. For the Terraform driver, these are files ending in `.tfvars` and
	// are used as Terraform input variables passed as arguments to the Terraform
	// module. Variables are loaded in the same order as they appear in the order
	// of the files. Duplicate variables are overwritten with the later value.
	VarFiles []string `mapstructure:"variable_files"`

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

	o.Source = StringCopy(c.Source)

	if !isSourceInputNil(c.SourceInput) {
		o.SourceInput = c.SourceInput.Copy()
	}

	o.VarFiles = append(o.VarFiles, c.VarFiles...)

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

	if o.Source != nil {
		r.Source = StringCopy(o.Source)
	}

	if !isSourceInputNil(o.SourceInput) {
		if isSourceInputNil(r.SourceInput) {
			r.SourceInput = o.SourceInput.Copy()
		} else {
			r.SourceInput = r.SourceInput.Merge(o.SourceInput)
		}
	}

	r.VarFiles = append(r.VarFiles, o.VarFiles...)

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

	if c.Source == nil {
		c.Source = String("")
	}

	if c.VarFiles == nil {
		c.VarFiles = []string{}
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
			logging.Global().Named(logSystemName).Named(taskSubsystemName).Warn(
				"disabling buffer_period for schedule condition. overriding "+
					"buffer_period configured for this task",
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
		c.Condition = DefaultConditionConfig()
	}
	c.Condition.Finalize(c.Services)

	if isSourceInputNil(c.SourceInput) {
		c.SourceInput = DefaultSourceInputConfig()
	}
	c.SourceInput.Finalize(c.Services)

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

	if c.Source == nil || len(*c.Source) == 0 {
		return fmt.Errorf("source for the task is required")
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

	if err := c.BufferPeriod.Validate(); err != nil {
		return err
	}

	if !isConditionNil(c.Condition) {
		if err := c.Condition.Validate(); err != nil {
			return err
		}
	}

	if !isSourceInputNil(c.SourceInput) {
		if err := c.SourceInput.Validate(); err != nil {
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
		"Source:%s, "+
		"VarFiles:%s, "+
		"Version:%s, "+
		"TFVersion: %s, "+
		"BufferPeriod:%s, "+
		"Enabled:%t, "+
		"Condition:%v"+
		"SourceInput:%v"+
		"}",
		StringVal(c.Name),
		StringVal(c.Description),
		c.Providers,
		c.Services,
		StringVal(c.Source),
		c.VarFiles,
		StringVal(c.Version),
		StringVal(c.TFVersion),
		c.BufferPeriod.GoString(),
		BoolVal(c.Enabled),
		c.Condition.GoString(),
		c.SourceInput.GoString(),
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
		case *ServicesConditionConfig:
			if cond.Regexp == nil || *cond.Regexp == "" {
				return fmt.Errorf("at least one service is required in task.services " +
					"or task.condition.regexp must be configured")
			}
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
			if isSourceInputNil(c.SourceInput) || isSourceInputEmpty(c.SourceInput) {
				return fmt.Errorf("schedule condition requires at least one service to " +
					"be configured in task.services or a source_input must be provided")
			}
		}
	} else {
		switch cond := c.Condition.(type) {
		case *ServicesConditionConfig:
			if cond.Regexp != nil && *cond.Regexp != "" {
				err := fmt.Errorf("task.services is not allowed if task.condition.regexp " +
					"is configured for a services condition")
				logging.Global().Named(logSystemName).Named(taskSubsystemName).
					Error("list of services and service condition regex both "+
						"provided. If both are needed, consider including the "+
						"list in the regex or creating separate tasks",
						"task_name", StringVal(c.Name), "error", err)
				return err
			}
		}
	}

	return nil
}

func (c *TaskConfig) validateSourceInput() error {
	// For now only schedule condition allows for source_input, so a condition of type ScheduleConditionConfig
	// is the only supported type
	switch c.Condition.(type) {
	case *ScheduleConditionConfig:
		if len(c.Services) == 0 {
			switch si := c.SourceInput.(type) {
			case *ServicesSourceInputConfig:
				// source_input "services" follows the same rules as condition "services"
				if si.Regexp == nil || *si.Regexp == "" {
					return fmt.Errorf("at least one service is required in task.services " +
						"or task.source_input.regexp must be configured")
				}
			case *ConsulKVSourceInputConfig:
				return fmt.Errorf("consul-kv source_input requires at least one service to " +
					"be configured in task.services")
			}
		} else {
			switch si := c.SourceInput.(type) {
			case *ServicesSourceInputConfig:
				if si.Regexp != nil && *si.Regexp != "" {
					err := fmt.Errorf("task.services is not allowed if task.source_input.regexp " +
						"is configured for a services source_input")
					logging.Global().Named(logSystemName).Named(taskSubsystemName).
						Error("list of services and service condition regex both "+
							"provided. If both are needed, consider including the "+
							"list in the regex or creating separate tasks",
							"task_name", StringVal(c.Name), "error", err)
					return err
				}
			}
		}
	default:
		if !isSourceInputNil(c.SourceInput) && !isSourceInputEmpty(c.SourceInput) {
			return fmt.Errorf("source_input is only supported when a schedule condition is configured")
		}
	}

	return nil
}
