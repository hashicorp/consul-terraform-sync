package config

import (
	"fmt"
)

var _ ConditionConfig = (*ConsulKVConditionConfig)(nil)

type ConsulKVConditionConfig struct {
	ConsulKVMonitorConfig `mapstructure:",squash"`
}

// Copy returns a deep copy of this configuration.
func (c *ConsulKVConditionConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	m, ok := c.ConsulKVMonitorConfig.Copy().(*ConsulKVMonitorConfig)
	if !ok {
		return nil
	}
	return &ConsulKVConditionConfig{
		ConsulKVMonitorConfig: *m,
	}
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
func (c *ConsulKVConditionConfig) Merge(o MonitorConfig) MonitorConfig {
	if c == nil {
		if isConditionNil(o) { // o is interface, use isConditionNil()
			return nil
		}
		return o.Copy()
	}

	if isConditionNil(o) {
		return c.Copy()
	}

	ckvc, ok := o.(*ConsulKVConditionConfig)
	if !ok {
		return nil
	}

	merged, ok := c.ConsulKVMonitorConfig.Merge(&ckvc.ConsulKVMonitorConfig).(*ConsulKVMonitorConfig)
	if !ok {
		return nil
	}

	return &ConsulKVConditionConfig{
		ConsulKVMonitorConfig: *merged,
	}
}

// Finalize ensures there no nil pointers.
func (c *ConsulKVConditionConfig) Finalize(consulkv []string) {
	if c == nil { // config not required, return early
		return
	}

	c.ConsulKVMonitorConfig.Finalize(consulkv)
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *ConsulKVConditionConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}

	return c.ConsulKVMonitorConfig.Validate()
}

// GoString defines the printable version of this struct.
func (c *ConsulKVConditionConfig) GoString() string {
	if c == nil {
		return "(*ConsulKVConditionConfig)(nil)"
	}

	return fmt.Sprintf("&ConsulKVConditionConfig{"+
		"%s"+
		"}",
		c.ConsulKVMonitorConfig.GoString(),
	)
}
