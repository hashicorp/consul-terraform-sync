package config

import (
	"fmt"
)

var _ ConditionConfig = (*CatalogServicesConditionConfig)(nil)

// CatalogServicesConditionConfig configures a condition configuration block
// of type 'catalog-services'. A catalog-services condition is triggered by changes
// that occur to services in the catalog-services api.
type CatalogServicesConditionConfig struct {
	CatalogServicesMonitorConfig `mapstructure:",squash"`
}

// Copy returns a deep copy of this configuration.
func (c *CatalogServicesConditionConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	m, ok := c.CatalogServicesMonitorConfig.Copy().(*CatalogServicesMonitorConfig)
	if !ok {
		return nil
	}
	return &CatalogServicesConditionConfig{
		CatalogServicesMonitorConfig: *m,
	}
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *CatalogServicesConditionConfig) Merge(o MonitorConfig) MonitorConfig {
	if c == nil {
		if isConditionNil(o) { // o is interface, use isConditionNil()
			return nil
		}
		return o.Copy()
	}

	if isConditionNil(o) {
		return c.Copy()
	}

	cscc, ok := o.(*CatalogServicesConditionConfig)
	if !ok {
		return nil
	}

	merged, ok := c.CatalogServicesMonitorConfig.Merge(&cscc.CatalogServicesMonitorConfig).(*CatalogServicesMonitorConfig)
	if !ok {
		return nil
	}

	return &CatalogServicesConditionConfig{
		CatalogServicesMonitorConfig: *merged,
	}
}

// Finalize ensures there no nil pointers with the _exception_ of Regexp. There
// is a need to distinguish betweeen nil regex (unconfigured regex) and empty
// string regex ("" regex pattern) at Validate()
func (c *CatalogServicesConditionConfig) Finalize() {
	if c == nil { // config not required, return early
		return
	}
	c.CatalogServicesMonitorConfig.Finalize()
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
// Note, it handles the possibility of nil Regexp value even after Finalize().
func (c *CatalogServicesConditionConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}
	return c.CatalogServicesMonitorConfig.Validate()
}

// GoString defines the printable version of this struct.
func (c *CatalogServicesConditionConfig) GoString() string {
	if c == nil {
		return "(*CatalogServicesConditionConfig)(nil)"
	}

	return fmt.Sprintf("&CatalogServicesConditionConfig{"+
		"%s"+
		"}",
		c.CatalogServicesMonitorConfig.GoString(),
	)
}
