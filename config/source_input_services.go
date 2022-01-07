package config

import (
	"fmt"
)

var _ SourceInputConfig = (*ServicesSourceInputConfig)(nil)

// ServicesSourceInputConfig configures a source_input configuration block of type
// 'services'. Data about the services monitored will be used as input for the source variables.
type ServicesSourceInputConfig struct {
	ServicesMonitorConfig `mapstructure:",squash"`
}

// Copy returns a deep copy of this configuration.
func (c *ServicesSourceInputConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	svc, ok := c.ServicesMonitorConfig.Copy().(*ServicesMonitorConfig)
	if !ok {
		return nil
	}
	return &ServicesSourceInputConfig{
		ServicesMonitorConfig: *svc,
	}
}

// Merge combines all values in this configuration `c` with the values in the other
// configuration `o`, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *ServicesSourceInputConfig) Merge(o MonitorConfig) MonitorConfig {
	if c == nil {
		if isModuleInputNil(o) { // o is interface, use isConditionNil()
			return nil
		}
		return o.Copy()
	}

	if isModuleInputNil(o) {
		return c.Copy()
	}

	scc, ok := o.(*ServicesSourceInputConfig)
	if !ok {
		return nil
	}

	merged, ok := c.ServicesMonitorConfig.Merge(&scc.ServicesMonitorConfig).(*ServicesMonitorConfig)
	if !ok {
		return nil
	}

	return &ServicesSourceInputConfig{
		ServicesMonitorConfig: *merged,
	}
}

// Finalize ensures there are no nil pointers.
func (c *ServicesSourceInputConfig) Finalize() {
	if c == nil { // config not required, return early
		return
	}
	c.ServicesMonitorConfig.Finalize()
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *ServicesSourceInputConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}
	if err := c.ServicesMonitorConfig.Validate(); err != nil {
		return fmt.Errorf("error validating `module_input \"services\"`: %s", err)
	}
	return nil
}

// GoString defines the printable version of this struct.
func (c *ServicesSourceInputConfig) GoString() string {
	if c == nil {
		return "(*ServicesSourceInputConfig)(nil)"
	}

	return fmt.Sprintf("&ServicesSourceInputConfig{"+
		"%s"+
		"}",
		c.ServicesMonitorConfig.GoString(),
	)
}
