package config

import (
	"fmt"
)

var _ ModuleInputConfig = (*ConsulKVModuleInputConfig)(nil)

// ConsulKVModuleInputConfig configures a module_input configuration block of
// type 'consul-kv'. The consul key-values will be used as input for the
// module variables.
type ConsulKVModuleInputConfig struct {
	ConsulKVMonitorConfig `mapstructure:",squash"`
}

// Copy returns a deep copy of this configuration.
func (c *ConsulKVModuleInputConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	svc, ok := c.ConsulKVMonitorConfig.Copy().(*ConsulKVMonitorConfig)
	if !ok {
		return nil
	}
	return &ConsulKVModuleInputConfig{
		ConsulKVMonitorConfig: *svc,
	}
}

// Merge combines all values in this configuration `c` with the values in the other
// configuration `o`, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *ConsulKVModuleInputConfig) Merge(o MonitorConfig) MonitorConfig {
	if c == nil {
		if isModuleInputNil(o) { // o is interface, use isConditionNil()
			return nil
		}
		return o.Copy()
	}

	if isModuleInputNil(o) {
		return c.Copy()
	}

	scc, ok := o.(*ConsulKVModuleInputConfig)
	if !ok {
		return nil
	}

	merged, ok := c.ConsulKVMonitorConfig.Merge(&scc.ConsulKVMonitorConfig).(*ConsulKVMonitorConfig)
	if !ok {
		return nil
	}

	return &ConsulKVModuleInputConfig{
		ConsulKVMonitorConfig: *merged,
	}
}

// Finalize ensures there are no nil pointers.
func (c *ConsulKVModuleInputConfig) Finalize() {
	if c == nil { // config not required, return early
		return
	}
	c.ConsulKVMonitorConfig.Finalize()
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *ConsulKVModuleInputConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}
	return c.ConsulKVMonitorConfig.Validate()
}

// String defines the printable version of this struct.
func (c ConsulKVModuleInputConfig) String() string {

	return fmt.Sprintf("{"+
		"%s"+
		"}",
		c.ConsulKVMonitorConfig.String(),
	)
}
