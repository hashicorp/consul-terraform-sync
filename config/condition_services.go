package config

import (
	"fmt"
	"regexp"
)

const servicesConditionType = "services"

var _ ConditionConfig = (*ServicesConditionConfig)(nil)

// ServicesConditionConfig configures a condition configuration block of type
// 'services'. This is the default type of condition. A services condition is
// triggered when changes occur to the the task's services.
type ServicesConditionConfig struct {
	Regexp *string `mapstructure:"regexp"`
}

// Copy returns a deep copy of this configuration.
func (c *ServicesConditionConfig) Copy() ConditionConfig {
	if c == nil {
		return nil
	}

	var o ServicesConditionConfig
	o.Regexp = StringCopy(c.Regexp)

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *ServicesConditionConfig) Merge(o ConditionConfig) ConditionConfig {
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
		return r
	}

	r2 := r.(*ServicesConditionConfig)

	if o2.Regexp != nil {
		r2.Regexp = StringCopy(o2.Regexp)
	}

	return r2
}

// Finalize ensures there no nil pointers.
func (c *ServicesConditionConfig) Finalize(services []string) {
	if c == nil { // config not required, return early
		return
	}

	if c.Regexp == nil {
		c.Regexp = String("")
	}
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *ServicesConditionConfig) Validate() error {
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
func (c *ServicesConditionConfig) GoString() string {
	if c == nil {
		return "(*ServicesConditionConfig)(nil)"
	}

	return fmt.Sprintf("&ServicesConditionConfig{"+
		"Regexp:%s, "+
		"}",
		StringVal(c.Regexp),
	)
}
