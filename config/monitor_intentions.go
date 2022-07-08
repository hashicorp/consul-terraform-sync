package config

import (
	"fmt"
)

const intentionsType = "intentions"

var _ MonitorConfig = (*IntentionsMonitorConfig)(nil)

// IntentionsMonitorConfig configures a configuration block adhering to the
// monitor interface of type 'intentions'. An intention monitor watches for changes
// that occur to intentions.
type IntentionsMonitorConfig struct {
	// Datacenter is the datacenter the intention is in.
	Datacenter *string `mapstructure:"datacenter"`

	// Namespace is the namespace of the intention (Consul Enterprise only). If
	// not provided, the namespace will be inferred from the CTS ACL token, or
	// default to the `default` namespace.
	Namespace *string `mapstructure:"namespace"`

	// SourceServices configures regexp/list of names of source services.
	SourceServices *IntentionsServicesConfig `mapstructure:"source_services"`

	// DestinationServices configures regexp/list of names of destination services.
	DestinationServices *IntentionsServicesConfig `mapstructure:"destination_services"`
}

func (c *IntentionsMonitorConfig) VariableType() string {
	return intentionsType
}

// Copy returns a deep copy of this configuration.
func (c *IntentionsMonitorConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	var o IntentionsMonitorConfig

	o.Datacenter = StringCopy(c.Datacenter)
	o.Namespace = StringCopy(c.Namespace)
	o.SourceServices = c.SourceServices.Copy()
	o.DestinationServices = c.DestinationServices.Copy()

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *IntentionsMonitorConfig) Merge(o MonitorConfig) MonitorConfig {
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
	o2, ok := o.(*IntentionsMonitorConfig)
	if !ok {
		return r
	}

	r2 := r.(*IntentionsMonitorConfig)

	if o2.Datacenter != nil {
		r2.Datacenter = StringCopy(o2.Datacenter)
	}
	if o2.Namespace != nil {
		r2.Namespace = StringCopy(o2.Namespace)
	}

	if o2.DestinationServices != nil {
		r2.DestinationServices = r2.DestinationServices.Merge(o2.DestinationServices)
	}

	if o2.SourceServices != nil {
		r2.SourceServices = r2.SourceServices.Merge(o2.SourceServices)
	}

	return r2
}

// Finalize ensures there no nil pointers.
func (c *IntentionsMonitorConfig) Finalize() {
	if c == nil { // config not required, return early
		return
	}

	if c.Datacenter == nil {
		c.Datacenter = String("")
	}
	if c.Namespace == nil {
		c.Namespace = String("")
	}

	c.SourceServices.Finalize()
	c.DestinationServices.Finalize()
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
// Note, it handles the possibility of nil Regexp value even after Finalize().
func (c *IntentionsMonitorConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}

	if c.SourceServices.Validate() != nil && c.DestinationServices.Validate() == nil {
		return fmt.Errorf("both source services and destination services must be configured")
	}

	if c.SourceServices.Validate() == nil && c.DestinationServices.Validate() != nil {
		return fmt.Errorf("both source services and destination services must be configured")
	}

	if err := c.SourceServices.Validate(); err != nil {
		return err
	}

	if err := c.DestinationServices.Validate(); err != nil {
		return err
	}

	return nil
}

// GoString defines the printable version of this struct.
func (c *IntentionsMonitorConfig) GoString() string {
	if c == nil {
		return "(*IntentionsMonitorConfig)(nil)"
	}
	return fmt.Sprintf("&IntentionsMonitorConfig{"+
		"Datacenter:%s, "+
		"Namespace:%s, "+
		"Source Services %s, "+
		"Destination Services %s"+
		"}",
		StringVal(c.Datacenter),
		StringVal(c.Namespace),
		c.SourceServices.GoString(),
		c.DestinationServices.GoString(),
	)
}
