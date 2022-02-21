package config

// NoConditionConfig is used to set a non-null value to a task's condition
// configuration block when it is unconfigured.
type NoConditionConfig struct{}

func (c *NoConditionConfig) VariableType() string {
	return ""
}

// Copy returns a deep copy of this configuration.
func (c *NoConditionConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}
	return &NoConditionConfig{}
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *NoConditionConfig) Merge(o MonitorConfig) MonitorConfig {
	if c == nil {
		return nil
	}
	return &NoConditionConfig{}
}

// Finalize ensures there no nil pointers.
func (c *NoConditionConfig) Finalize() {
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *NoConditionConfig) Validate() error {
	return nil
}

// GoString defines the printable version of this struct.
func (c *NoConditionConfig) GoString() string {
	return "{}"
}
