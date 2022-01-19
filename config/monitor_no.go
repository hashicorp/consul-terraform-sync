package config

// NoMonitorConfig is used to set a non-null value to a task's source input or
// condition configuration block when it is unconfigured.
type NoMonitorConfig struct{}

func (c *NoMonitorConfig) VariableType() string {
	return ""
}

// Copy returns a deep copy of this configuration.
func (c *NoMonitorConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}
	return &NoMonitorConfig{}
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *NoMonitorConfig) Merge(o MonitorConfig) MonitorConfig {
	if c == nil {
		return nil
	}
	return &NoMonitorConfig{}
}

// Finalize ensures there no nil pointers.
func (c *NoMonitorConfig) Finalize() {
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *NoMonitorConfig) Validate() error {
	return nil
}

// GoString defines the printable version of this struct.
func (c *NoMonitorConfig) GoString() string {
	return "{}"
}
