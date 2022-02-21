package config

import (
	"fmt"
)

const consulKVType = "consul-kv"

var _ MonitorConfig = (*ConsulKVMonitorConfig)(nil)

// ConsulKVMonitorConfig configures a configuration block adhering to the monitor interface
// of type 'consul-kv'. A consul-kv monitor watches for changes
// that occur in the consul kv.
type ConsulKVMonitorConfig struct {
	Path       *string `mapstructure:"path"`
	Recurse    *bool   `mapstructure:"recurse"`
	Datacenter *string `mapstructure:"datacenter"`
	Namespace  *string `mapstructure:"namespace"`
}

func (c *ConsulKVMonitorConfig) VariableType() string {
	return "consul_kv"
}

// Copy returns a deep copy of this configuration.
func (c *ConsulKVMonitorConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	var o ConsulKVMonitorConfig
	o.Path = StringCopy(c.Path)
	o.Recurse = BoolCopy(c.Recurse)
	o.Datacenter = StringCopy(c.Datacenter)
	o.Namespace = StringCopy(c.Namespace)

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
func (c *ConsulKVMonitorConfig) Merge(o MonitorConfig) MonitorConfig {
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
	o2, ok := o.(*ConsulKVMonitorConfig)
	if !ok {
		return r
	}

	r2 := r.(*ConsulKVMonitorConfig)

	if o2.Path != nil {
		r2.Path = StringCopy(o2.Path)
	}

	if o2.Recurse != nil {
		r2.Recurse = BoolCopy(o2.Recurse)
	}

	if o2.Datacenter != nil {
		r2.Datacenter = StringCopy(o2.Datacenter)
	}

	if o2.Namespace != nil {
		r2.Namespace = StringCopy(o2.Namespace)
	}

	return r2
}

// Finalize ensures there no nil pointers.
func (c *ConsulKVMonitorConfig) Finalize() {
	if c == nil { // config not required, return early
		return
	}

	if c.Path == nil {
		c.Path = String("")
	}

	if c.Recurse == nil {
		c.Recurse = Bool(false)
	}

	if c.Datacenter == nil {
		c.Datacenter = String("")
	}

	if c.Namespace == nil {
		c.Namespace = String("")
	}

}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *ConsulKVMonitorConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}

	if c.Path == nil || *c.Path == "" {
		return fmt.Errorf("path is required for consul-kv condition")
	}

	return nil
}

// GoString defines the printable version of this struct.
func (c *ConsulKVMonitorConfig) GoString() string {
	if c == nil {
		return "(*ConsulKVMonitorConfig)(nil)"
	}

	return fmt.Sprintf("&ConsulKVMonitorConfig{"+
		"Path:%s, "+
		"Recurse:%v, "+
		"Datacenter:%v, "+
		"Namespace:%v, "+
		"}",
		StringVal(c.Path),
		BoolVal(c.Recurse),
		StringVal(c.Datacenter),
		StringVal(c.Namespace),
	)
}
