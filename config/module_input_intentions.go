package config

import (
	"fmt"
)

var _ ModuleInputConfig = (*IntentionsModuleInputConfig)(nil)

// IntentionsModuleInputConfig configures a module_input configuration block of
// type 'intentions'. Data about the intentions monitored will be used as input
// for the module variables.
type IntentionsModuleInputConfig struct {
	IntentionsMonitorConfig `mapstructure:",squash"`
}

// Copy returns a deep copy of this configuration.
func (c *IntentionsModuleInputConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	conf, ok := c.IntentionsMonitorConfig.Copy().(*IntentionsMonitorConfig)
	if !ok {
		return nil
	}
	return &IntentionsModuleInputConfig{
		IntentionsMonitorConfig: *conf,
	}
}

// Merge combines all values in this configuration `c` with the values in the other
// configuration `o`, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *IntentionsModuleInputConfig) Merge(o MonitorConfig) MonitorConfig {
	if c == nil {
		if isModuleInputNil(o) { // o is interface, use isConditionNil()
			return nil
		}
		return o.Copy()
	}

	if isModuleInputNil(o) {
		return c.Copy()
	}

	o2, ok := o.(*IntentionsModuleInputConfig)
	if !ok {
		return nil
	}

	merged, ok := c.IntentionsMonitorConfig.Merge(&o2.IntentionsMonitorConfig).(*IntentionsMonitorConfig)
	if !ok {
		return nil
	}

	return &IntentionsModuleInputConfig{
		IntentionsMonitorConfig: *merged,
	}
}

// Finalize ensures there are no nil pointers.
func (c *IntentionsModuleInputConfig) Finalize() {
	if c == nil { // config not required, return early
		return
	}
	c.IntentionsMonitorConfig.Finalize()
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *IntentionsModuleInputConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}
	if err := c.IntentionsMonitorConfig.Validate(); err != nil {
		return fmt.Errorf("error validating `module_input \"intentions\"`: %s", err)
	}
	return nil
}

// GoString defines the printable version of this struct.
func (c *IntentionsModuleInputConfig) GoString() string {
	if c == nil {
		return "(*IntentionsModuleInputConfig)(nil)"
	}

	return fmt.Sprintf("&IntentionsModuleInputConfig{"+
		"%s"+
		"}",
		c.IntentionsMonitorConfig.GoString(),
	)
}
