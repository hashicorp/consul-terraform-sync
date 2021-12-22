package config

import (
	"fmt"
	"regexp"
)

const catalogServicesType = "catalog-services"

var _ ConditionConfig = (*CatalogServicesConditionConfig)(nil)

// CatalogServicesMonitorConfig configures a configuration block adhering to the monitor interface
// of type 'catalog-services'. A catalog-services monitor is triggered by changes
// that occur to services in the catalog-services api.
type CatalogServicesMonitorConfig struct {
	Regexp            *string           `mapstructure:"regexp"`
	SourceIncludesVar *bool             `mapstructure:"source_includes_var"`
	Datacenter        *string           `mapstructure:"datacenter"`
	Namespace         *string           `mapstructure:"namespace"`
	NodeMeta          map[string]string `mapstructure:"node_meta"`
}

// Copy returns a deep copy of this configuration.
func (c *CatalogServicesMonitorConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	var o CatalogServicesMonitorConfig

	o.Regexp = StringCopy(c.Regexp)
	o.SourceIncludesVar = BoolCopy(c.SourceIncludesVar)
	o.Datacenter = StringCopy(c.Datacenter)
	o.Namespace = StringCopy(c.Namespace)

	if c.NodeMeta != nil {
		o.NodeMeta = make(map[string]string)
		for k, v := range c.NodeMeta {
			o.NodeMeta[k] = v
		}
	}

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *CatalogServicesMonitorConfig) Merge(o MonitorConfig) MonitorConfig {
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
	o2, ok := o.(*CatalogServicesMonitorConfig)
	if !ok {
		return r
	}

	r2 := r.(*CatalogServicesMonitorConfig)

	if o2.Regexp != nil {
		r2.Regexp = StringCopy(o2.Regexp)
	}

	if o2.SourceIncludesVar != nil {
		r2.SourceIncludesVar = BoolCopy(o2.SourceIncludesVar)
	}

	if o2.Datacenter != nil {
		r2.Datacenter = StringCopy(o2.Datacenter)
	}

	if o2.Namespace != nil {
		r2.Namespace = StringCopy(o2.Namespace)
	}

	if o2.NodeMeta != nil {
		if r2.NodeMeta == nil {
			r2.NodeMeta = make(map[string]string)
		}
		for k, v := range o2.NodeMeta {
			r2.NodeMeta[k] = v
		}
	}

	return r2
}

// Finalize ensures there no nil pointers with the _exception_ of Regexp. There
// is a need to distinguish between nil regex (unconfigured regex) and empty
// string regex ("" regex pattern) at Validate()
func (c *CatalogServicesMonitorConfig) Finalize(services []string) {
	if c == nil { // config not required, return early
		return
	}

	if c.SourceIncludesVar == nil {
		c.SourceIncludesVar = Bool(false)
	}

	if c.Datacenter == nil {
		c.Datacenter = String("")
	}

	if c.Namespace == nil {
		c.Namespace = String("")
	}

	if c.NodeMeta == nil {
		c.NodeMeta = make(map[string]string)
	}
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
// Note, it handles the possibility of nil Regexp value even after Finalize().
func (c *CatalogServicesMonitorConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}

	if c.Regexp == nil {
		return fmt.Errorf("catalog-services 'regexp' field must be set")
	}

	if _, err := regexp.Compile(StringVal(c.Regexp)); err != nil {
		return fmt.Errorf("unable to compile catalog-services 'regexp': %s", err)
	}
	return nil
}

// GoString defines the printable version of this struct.
func (c *CatalogServicesMonitorConfig) GoString() string {
	if c == nil {
		return "(*CatalogServicesMonitorConfig)(nil)"
	}

	return fmt.Sprintf("&CatalogServicesMonitorConfig{"+
		"Regexp:%s, "+
		"SourceIncludesVar:%v, "+
		"Datacenter:%v, "+
		"Namespace:%v, "+
		"NodeMeta:%s"+
		"}",
		StringVal(c.Regexp),
		BoolVal(c.SourceIncludesVar),
		StringVal(c.Datacenter),
		StringVal(c.Namespace),
		c.NodeMeta,
	)
}
