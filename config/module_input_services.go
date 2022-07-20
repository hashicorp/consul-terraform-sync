package config

import (
	"fmt"
)

var _ ModuleInputConfig = (*ServicesModuleInputConfig)(nil)

// ServicesModuleInputConfig configures a module_input configuration block of
// type 'services'. Data about the services monitored will be used as input for
// the module variables.
type ServicesModuleInputConfig struct {
	ServicesMonitorConfig `mapstructure:",squash" json:"services"`
}

// Copy returns a deep copy of this configuration.
func (c *ServicesModuleInputConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	svc, ok := c.ServicesMonitorConfig.Copy().(*ServicesMonitorConfig)
	if !ok {
		return nil
	}
	return &ServicesModuleInputConfig{
		ServicesMonitorConfig: *svc,
	}
}

// Merge combines all values in this configuration `c` with the values in the other
// configuration `o`, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *ServicesModuleInputConfig) Merge(o MonitorConfig) MonitorConfig {
	if c == nil {
		if isModuleInputNil(o) { // o is interface, use isConditionNil()
			return nil
		}
		return o.Copy()
	}

	if isModuleInputNil(o) {
		return c.Copy()
	}

	scc, ok := o.(*ServicesModuleInputConfig)
	if !ok {
		return nil
	}

	merged, ok := c.ServicesMonitorConfig.Merge(&scc.ServicesMonitorConfig).(*ServicesMonitorConfig)
	if !ok {
		return nil
	}

	return &ServicesModuleInputConfig{
		ServicesMonitorConfig: *merged,
	}
}

// Finalize ensures there are no nil pointers.
func (c *ServicesModuleInputConfig) Finalize() {
	if c == nil { // config not required, return early
		return
	}
	c.ServicesMonitorConfig.Finalize()
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *ServicesModuleInputConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}
	if err := c.ServicesMonitorConfig.Validate(); err != nil {
		return fmt.Errorf("error validating `module_input \"services\"`: %s", err)
	}
	return nil
}

// GoString defines the printable version of this struct.
func (c *ServicesModuleInputConfig) GoString() string {
	if c == nil {
		return "(*ServicesModuleInputConfig)(nil)"
	}

	return fmt.Sprintf("&ServicesModuleInputConfig{"+
		"%s"+
		"}",
		c.ServicesMonitorConfig.GoString(),
	)
}
