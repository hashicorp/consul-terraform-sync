package config

import (
	"fmt"

	"github.com/hashicorp/consul-terraform-sync/logging"
)

var _ ConditionConfig = (*ConsulKVConditionConfig)(nil)

// ConsulKVConditionConfig configures a condition configuration block
// of type 'consul-kv'. A consul-kv condition is triggered by changes
// that occur to consul key-values.
type ConsulKVConditionConfig struct {
	ConsulKVMonitorConfig `mapstructure:",squash"`

	// UseAsModuleInput was previously named SourceIncludesVar - deprecated v0.5
	UseAsModuleInput            *bool `mapstructure:"use_as_module_input" json:"use_as_module_input"`
	DeprecatedSourceIncludesVar *bool `mapstructure:"source_includes_var" json:"source_includes_var"`
}

// Copy returns a deep copy of this configuration.
func (c *ConsulKVConditionConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	var o ConsulKVConditionConfig
	o.UseAsModuleInput = BoolCopy(c.UseAsModuleInput)
	o.DeprecatedSourceIncludesVar = BoolCopy(c.DeprecatedSourceIncludesVar)

	m, ok := c.ConsulKVMonitorConfig.Copy().(*ConsulKVMonitorConfig)
	if !ok {
		return nil
	}

	o.ConsulKVMonitorConfig = *m

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
func (c *ConsulKVConditionConfig) Merge(o MonitorConfig) MonitorConfig {
	if c == nil {
		if isConditionNil(o) { // o is interface, use isConditionNil()
			return nil
		}
		return o.Copy()
	}

	if isConditionNil(o) {
		return c.Copy()
	}

	r := c.Copy()
	o2, ok := o.(*ConsulKVConditionConfig)
	if !ok {
		return nil
	}

	r2 := r.(*ConsulKVConditionConfig)

	if o2.UseAsModuleInput != nil {
		r2.UseAsModuleInput = BoolCopy(o2.UseAsModuleInput)
	}
	if o2.DeprecatedSourceIncludesVar != nil {
		r2.DeprecatedSourceIncludesVar = BoolCopy(o2.DeprecatedSourceIncludesVar)
	}

	mm, ok := c.ConsulKVMonitorConfig.Merge(&o2.ConsulKVMonitorConfig).(*ConsulKVMonitorConfig)
	if !ok {
		return nil
	}
	r2.ConsulKVMonitorConfig = *mm

	return r2
}

// Finalize ensures there no nil pointers.
func (c *ConsulKVConditionConfig) Finalize() {
	if c == nil { // config not required, return early
		return
	}

	logger := logging.Global().Named(logSystemName).Named(taskSubsystemName)
	if c.DeprecatedSourceIncludesVar != nil {
		logger.Warn("Consul-KV condition block's 'source_includes_var' " +
			"field was marked for deprecation in v0.5.0. Please update your " +
			"configuration to use the 'use_as_module_input' field instead")

		if c.UseAsModuleInput != nil {
			logger.Warn("Consul-KV condition block is configured with "+
				"both 'source_includes_var' and 'use_as_module_input' field. "+
				"Defaulting to 'use_as_module_input' value",
				"use_as_module_input", c.UseAsModuleInput)
		} else {
			// Merge SourceIncludesVar with UseAsModuleInput. Use UseAsModuleInput onwards
			c.UseAsModuleInput = c.DeprecatedSourceIncludesVar
		}
	}
	if c.UseAsModuleInput == nil {
		c.UseAsModuleInput = Bool(true)
	}

	c.ConsulKVMonitorConfig.Finalize()
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *ConsulKVConditionConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}

	return c.ConsulKVMonitorConfig.Validate()
}

// String defines the printable version of this struct.
func (c ConsulKVConditionConfig) String() string {

<<<<<<< HEAD
	return fmt.Sprintf("{"+
		"SourceIncludesVar:%v, "+
		"%s"+
		"}",
		BoolVal(c.SourceIncludesVar),
		c.ConsulKVMonitorConfig.String(),
=======
	return fmt.Sprintf("&ConsulKVConditionConfig{"+
		"%s, "+
		"UseAsModuleInput:%v"+
		"}",
		c.ConsulKVMonitorConfig.GoString(),
		BoolVal(c.UseAsModuleInput),
>>>>>>> upstream/main
	)
}
