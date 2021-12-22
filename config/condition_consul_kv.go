package config

import (
	"fmt"
)

var _ ConditionConfig = (*ConsulKVConditionConfig)(nil)

// ConsulKVConditionConfig configures a condition configuration block
// of type 'consul-kv'. A consul-kv condition is triggered by changes
// that occur to consul key-values.
type ConsulKVConditionConfig struct {
	ConsulKVMonitorConfig `mapstructure:",squash"`
	SourceIncludesVar     *bool `mapstructure:"source_includes_var" json:"source_includes_var"`
}

// Copy returns a deep copy of this configuration.
func (c *ConsulKVConditionConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	var o ConsulKVConditionConfig
	o.SourceIncludesVar = BoolCopy(c.SourceIncludesVar)

	m, ok := c.ConsulKVMonitorConfig.Copy().(*ConsulKVMonitorConfig)
	if !ok {
		return nil
	}

	o.ConsulKVMonitorConfig = *m

	return &o
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

	r := c.Copy()
	o2, ok := o.(*ConsulKVConditionConfig)
	if !ok {
		return nil
	}

	r2 := r.(*ConsulKVConditionConfig)
	if o2.SourceIncludesVar != nil {
		r2.SourceIncludesVar = BoolCopy(o2.SourceIncludesVar)
	}

	mm, ok := c.ConsulKVMonitorConfig.Merge(&o2.ConsulKVMonitorConfig).(*ConsulKVMonitorConfig)
	if !ok {
		return nil
	}
	r2.ConsulKVMonitorConfig = *mm

	return r2
}

// Finalize ensures there no nil pointers.
func (c *ConsulKVConditionConfig) Finalize(consulkv []string) {
	if c == nil { // config not required, return early
		return
	}

	if c.SourceIncludesVar == nil {
		c.SourceIncludesVar = Bool(false)
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

// String defines the printable version of this struct.
func (c ConsulKVConditionConfig) String() string {

	return fmt.Sprintf("{"+
		"SourceIncludesVar:%v, "+
		"%s"+
		"}",
		BoolVal(c.SourceIncludesVar),
		c.ConsulKVMonitorConfig.String(),
	)
}
