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
	SourceIncludesVar     *bool `mapstructure:"source_includes_var"`
}

// Copy returns a deep copy of this configuration.
func (c *ServicesConditionConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	var o ServicesConditionConfig
	o.SourceIncludesVar = BoolCopy(c.SourceIncludesVar)

	svc, ok := c.ServicesMonitorConfig.Copy().(*ServicesMonitorConfig)
	if !ok {
		return nil
	}

	o.ServicesMonitorConfig = *svc
	return &o
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

	r := c.Copy()
	o2, ok := o.(*ServicesConditionConfig)
	if !ok {
		return nil
	}

	r2 := r.(*ServicesConditionConfig)
	if o2.SourceIncludesVar != nil {
		r2.SourceIncludesVar = BoolCopy(o2.SourceIncludesVar)
	}

	merged, ok := c.ServicesMonitorConfig.Merge(&o2.ServicesMonitorConfig).(*ServicesMonitorConfig)
	if !ok {
		return nil
	}

	r2.ServicesMonitorConfig = *merged
	return r2
}

// Finalize ensures there no nil pointers.
func (c *ServicesConditionConfig) Finalize() {
	if c == nil { // config not required, return early
		return
	}

	if c.SourceIncludesVar == nil {
		c.SourceIncludesVar = Bool(true)
	}

	c.ServicesMonitorConfig.Finalize()
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *ServicesConditionConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}
	if err := c.ServicesMonitorConfig.Validate(); err != nil {
		return fmt.Errorf("error validating `condition \"services\"` block: %s",
			err)
	}
	return nil
}

// GoString defines the printable version of this struct.
func (c *ServicesConditionConfig) GoString() string {
	if c == nil {
		return "(*ServicesConditionConfig)(nil)"
	}

	return fmt.Sprintf("&ServicesConditionConfig{"+
		"%s, "+
		"SourceIncludesVar:%v"+
		"}",
		c.ServicesMonitorConfig.GoString(),
		BoolVal(c.SourceIncludesVar),
	)
}
