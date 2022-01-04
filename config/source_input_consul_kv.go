package config

import (
	"fmt"
)

var _ SourceInputConfig = (*ConsulKVSourceInputConfig)(nil)

// ConsulKVSourceInputConfig configures a source_input configuration block of type
// 'consul-kv'. The consul key-values will be used as input for the source variables.
type ConsulKVSourceInputConfig struct {
	ConsulKVMonitorConfig `mapstructure:",squash"`
}

// Copy returns a deep copy of this configuration.
func (c *ConsulKVSourceInputConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	svc, ok := c.ConsulKVMonitorConfig.Copy().(*ConsulKVMonitorConfig)
	if !ok {
		return nil
	}
	return &ConsulKVSourceInputConfig{
		ConsulKVMonitorConfig: *svc,
	}
}

// Merge combines all values in this configuration `c` with the values in the other
// configuration `o`, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *ConsulKVSourceInputConfig) Merge(o MonitorConfig) MonitorConfig {
	if c == nil {
		if isSourceInputNil(o) { // o is interface, use isConditionNil()
			return nil
		}
		return o.Copy()
	}

	if isSourceInputNil(o) {
		return c.Copy()
	}

	scc, ok := o.(*ConsulKVSourceInputConfig)
	if !ok {
		return nil
	}

	merged, ok := c.ConsulKVMonitorConfig.Merge(&scc.ConsulKVMonitorConfig).(*ConsulKVMonitorConfig)
	if !ok {
		return nil
	}

	return &ConsulKVSourceInputConfig{
		ConsulKVMonitorConfig: *merged,
	}
}

// Finalize ensures there are no nil pointers.
func (c *ConsulKVSourceInputConfig) Finalize() {
	if c == nil { // config not required, return early
		return
	}
	c.ConsulKVMonitorConfig.Finalize()
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *ConsulKVSourceInputConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}
	return c.ConsulKVMonitorConfig.Validate()
}

// GoString defines the printable version of this struct.
func (c *ConsulKVSourceInputConfig) GoString() string {
	if c == nil {
		return "(*ConsulKVSourceInputConfig)(nil)"
	}

	return fmt.Sprintf("&ConsulKVSourceInputConfig{"+
		"%s"+
		"}",
		c.ConsulKVMonitorConfig.GoString(),
	)
}
