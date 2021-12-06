package config

// NoSourceInputConfig is used to set a non-null value to a task's source input
// configuration block when it is unconfigured.
type NoSourceInputConfig struct{}

// Copy returns a deep copy of this configuration.
func (c *NoSourceInputConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}
	return &NoSourceInputConfig{}
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *NoSourceInputConfig) Merge(o MonitorConfig) MonitorConfig {
	if c == nil {
		return nil
	}
	return &NoSourceInputConfig{}
}

// Finalize ensures there no nil pointers.
func (c *NoSourceInputConfig) Finalize(services []string) {
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *NoSourceInputConfig) Validate() error {
	return nil
}

// GoString defines the printable version of this struct.
func (c *NoSourceInputConfig) GoString() string {
	return ""
}
