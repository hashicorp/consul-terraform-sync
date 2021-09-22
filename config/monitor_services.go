package config

import (
	"fmt"
	"regexp"
)

const servicesMonitorType = "services"

var _ MonitorConfig = (*ServicesMonitorConfig)(nil)

type ServicesMonitorConfig struct {
	Regexp *string `mapstructure:"regexp"`
}

// Copy returns a deep copy of this configuration.
func (c *ServicesMonitorConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	var o ServicesMonitorConfig
	o.Regexp = StringCopy(c.Regexp)

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *ServicesMonitorConfig) Merge(o MonitorConfig) MonitorConfig {
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
	o2, ok := o.(*ServicesMonitorConfig)
	if !ok {
		return r
	}

	r2 := r.(*ServicesMonitorConfig)

	if o2.Regexp != nil {
		r2.Regexp = StringCopy(o2.Regexp)
	}

	return r2
}

// Finalize ensures there no nil pointers.
func (c *ServicesMonitorConfig) Finalize([]string) {
	if c == nil { // config not required, return early
		return
	}

	if c.Regexp == nil {
		c.Regexp = String("")
	}
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *ServicesMonitorConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}

	if c.Regexp != nil && *c.Regexp != "" {
		if _, err := regexp.Compile(StringVal(c.Regexp)); err != nil {
			return fmt.Errorf("unable to compile services regexp: %s", err)
		}
	}

	return nil
}

// GoString defines the printable version of this struct.
func (c *ServicesMonitorConfig) GoString() string {
	if c == nil {
		return "(*ServicesMonitorConfig)(nil)"
	}

	return fmt.Sprintf("&ServicesMonitorConfig{"+
		"Regexp:%s, "+
		"}",
		StringVal(c.Regexp),
	)
}
