package config

import (
	"fmt"
)

var _ ConditionConfig = (*ServicesConditionConfig)(nil)

// ServicesConditionConfig configures a condition configuration block of type
// 'services'. This is the default type of condition. A services condition is
// triggered when changes occur to the task's services.
type ServicesConditionConfig struct {
	ServicesMonitorConfig `mapstructure:",squash"`
}

// Copy returns a deep copy of this configuration.
func (c *ServicesConditionConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	svc, ok := c.ServicesMonitorConfig.Copy().(*ServicesMonitorConfig)
	if !ok {
		return nil
	}
	return &ServicesConditionConfig{
		ServicesMonitorConfig: *svc,
	}
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *ServicesConditionConfig) Merge(o MonitorConfig) MonitorConfig {
	if c == nil {
		if isConditionNil(o) { // o is interface, use isConditionNil()
			return nil
		}
		return o.Copy()
	}

	if isConditionNil(o) {
		return c.Copy()
	}

	scc, ok := o.(*ServicesConditionConfig)
	if !ok {
		return nil
	}

	merged, ok := c.ServicesMonitorConfig.Merge(&scc.ServicesMonitorConfig).(*ServicesMonitorConfig)
	if !ok {
		return nil
	}

	return &ServicesConditionConfig{
		ServicesMonitorConfig: *merged,
	}
}

// Finalize ensures there no nil pointers.
func (c *ServicesConditionConfig) Finalize(services []string) {
	if c == nil { // config not required, return early
		return
	}
	c.ServicesMonitorConfig.Finalize(services)
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *ServicesConditionConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}
	return c.ServicesMonitorConfig.Validate()
}

// String defines the printable version of this struct.
func (c ServicesConditionConfig) String() string {

	return fmt.Sprintf("{"+
		"%s"+
		"}",
		c.ServicesMonitorConfig.String(),
	)
}
