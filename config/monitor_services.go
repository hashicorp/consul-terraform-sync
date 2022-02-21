package config

import (
	"fmt"
	"regexp"
)

const servicesType = "services"

var _ MonitorConfig = (*ServicesMonitorConfig)(nil)

// ServicesMonitorConfig configures a configuration block adhering to the
// monitor interface of type 'services'. A services monitor watches for changes
// that occur to services. ServicesMonitorConfig shares similar fields as the
// deprecated ServiceConfig
type ServicesMonitorConfig struct {
	// Regexp configures the services to monitor by matching on the service name.
	// Either Regexp or Names must be configured, not both. When Regexp is unset,
	// it will retain a nil value even after Finalize().
	Regexp *string `mapstructure:"regexp"`

	// Names configures the services to monitor by listing the service name.
	// Either Regexp or Names must be configured, not both.
	Names []string `mapstructure:"names"`

	// Datacenter is the datacenter the service is deployed in.
	Datacenter *string `mapstricture:"datacenter"`

	// Namespace is the namespace of the service (Consul Enterprise only). If
	// not provided, the namespace will be inferred from the CTS ACL token, or
	// default to the `default` namespace.
	Namespace *string `mapstructure:"namespace"`

	// Filter is used to filter nodes based on a Consul compatible filter
	// expression.
	Filter *string `mapstructure:"filter"`

	// CTSUserDefinedMeta is metadata added to a service automated by CTS for
	// network infrastructure automation.
	CTSUserDefinedMeta map[string]string `mapstructure:"cts_user_defined_meta"`
}

func (c *ServicesMonitorConfig) VariableType() string {
	return "services"
}

// Copy returns a deep copy of this configuration.
func (c *ServicesMonitorConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	var o ServicesMonitorConfig
	o.Regexp = StringCopy(c.Regexp)
	o.Names = append(o.Names, c.Names...)
	o.Datacenter = StringCopy(c.Datacenter)
	o.Namespace = StringCopy(c.Namespace)
	o.Filter = StringCopy(c.Filter)
	if c.CTSUserDefinedMeta != nil {
		o.CTSUserDefinedMeta = make(map[string]string)
		for k, v := range c.CTSUserDefinedMeta {
			o.CTSUserDefinedMeta[k] = v
		}
	}

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *ServicesMonitorConfig) Merge(o MonitorConfig) MonitorConfig {
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
	o2, ok := o.(*ServicesMonitorConfig)
	if !ok {
		return r
	}

	r2 := r.(*ServicesMonitorConfig)

	if o2.Regexp != nil {
		r2.Regexp = StringCopy(o2.Regexp)
	}

	r2.Names = append(r2.Names, o2.Names...)

	if o2.Datacenter != nil {
		r2.Datacenter = StringCopy(o2.Datacenter)
	}
	if o2.Namespace != nil {
		r2.Namespace = StringCopy(o2.Namespace)
	}
	if o2.Filter != nil {
		r2.Filter = StringCopy(o2.Filter)
	}
	if o2.CTSUserDefinedMeta != nil {
		if r2.CTSUserDefinedMeta == nil {
			r2.CTSUserDefinedMeta = make(map[string]string)
		}
		for k, v := range o2.CTSUserDefinedMeta {
			r2.CTSUserDefinedMeta[k] = v
		}
	}

	return r2
}

// Finalize ensures there no nil pointers.
//
// Exception: `Regexp` is never finalized and can potentially be nil.
//  - There is a need to distinguish betweeen nil regex (unconfigured regex) and
//  empty string regex ("" regex pattern) at Validate().
//  - Setting `Regexp` as an empty string is not idempotent. There is a need to
//  call Finalize() and Validate() multiple times.
func (c *ServicesMonitorConfig) Finalize() {
	if c == nil { // config not required, return early
		return
	}

	if c.Names == nil {
		c.Names = []string{}
	}
	if c.Datacenter == nil {
		c.Datacenter = String("")
	}
	if c.Namespace == nil {
		c.Namespace = String("")
	}
	if c.Filter == nil {
		c.Filter = String("")
	}
	if c.CTSUserDefinedMeta == nil {
		c.CTSUserDefinedMeta = make(map[string]string)
	}
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
// Note, it handles the possibility of nil Regexp value even after Finalize().
func (c *ServicesMonitorConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}

	// Check that either regex or names is configured but not both
	namesConfigured := c.Names != nil && len(c.Names) > 0
	regexConfigured := c.Regexp != nil
	if namesConfigured && regexConfigured {
		return fmt.Errorf("regexp and names fields cannot both be " +
			"unconfigured. If both are needed, consider including the list of " +
			"names as part of the regex or creating separate tasks")
	}
	if !namesConfigured && !regexConfigured {
		return fmt.Errorf("either the regexp or names field must be configured")
	}

	// Validate regex
	if regexConfigured {
		if _, err := regexp.Compile(StringVal(c.Regexp)); err != nil {
			return fmt.Errorf("unable to compile services regexp: %s", err)
		}
	}

	// Check that names does not contain empty strings
	if namesConfigured {
		for _, name := range c.Names {
			if name == "" {
				return fmt.Errorf("names field includes empty string(s). " +
					"services names cannot be empty")
			}
		}
	}

	return nil
}

// GoString defines the printable version of this struct.
func (c *ServicesMonitorConfig) GoString() string {
	if c == nil {
		return "(*ServicesMonitorConfig)(nil)"
	}

	return fmt.Sprintf("&ServicesMonitorConfig{"+
		"Regexp:%s, "+
		"Names:%s, "+
		"Datacenter:%s, "+
		"Namespace:%s, "+
		"Filter:%s, "+
		"CTSUserDefinedMeta:%s"+
		"}",
		StringVal(c.Regexp),
		c.Names,
		StringVal(c.Datacenter),
		StringVal(c.Namespace),
		StringVal(c.Filter),
		c.CTSUserDefinedMeta,
	)
}
