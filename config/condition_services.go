package config

const servicesConditionType = "services"

var _ ConditionConfig = (*ServicesConditionConfig)(nil)

// ServicesConditionConfig configures a condition configuration block of type
// 'services'. This is the default type of condition. A services condition is
// triggered when changes occur to the the task's services.
type ServicesConditionConfig struct{}

// Copy returns a deep copy of this configuration.
func (c *ServicesConditionConfig) Copy() ConditionConfig {
	if c == nil {
		return nil
	}

	var o ServicesConditionConfig
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

	return c.Copy()
}

// Finalize ensures there no nil pointers.
func (c *ServicesConditionConfig) Finalize(services []string) {
	// no-op: no fields to finalize yet
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *ServicesConditionConfig) Validate() error {
	// no-op: no fields to validate yet
	return nil
}

// GoString defines the printable version of this struct.
func (c *ServicesConditionConfig) GoString() string {
	if c == nil {
		return "(*ServicesConditionConfig)(nil)"
	}

	return "&ServicesConditionConfig{}"
}
