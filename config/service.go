package config

import (
	"fmt"
	"log"
	"strings"
)

// ServiceConfig defines the explicit configuration for Sync to monitor
// a service. This block may be specified multiple times to configure multiple
// services.
type ServiceConfig struct {
	// Datacenter is the datacenter the service is deployed in.
	Datacenter *string `mapstricture:"datacenter"`

	// Description is the human readable text to describe the service.
	Description *string `mapstructure:"description"`

	// ID identifies the service for Sync. This is used to explicitly
	// identify the service config for a task to use.
	ID *string `mapstructure:"id"`

	// Name is the Consul logical name of the service (required).
	Name *string `mapstructure:"name"`

	// Namespace is the namespace of the service (Consul Enterprise only). If not
	// provided, the namespace will be inferred from the Sync ACL token, or
	// default to the `default` namespace.
	Namespace *string `mapstructure:"namespace"`

	// Tag is used to filter nodes based on the tag for the service.
	// Deprecated in favor of Filter.
	Tag *string `mapstructure:"tag"`

	// Filter is used to filter nodes based on a Consul compatible filter expression.
	Filter *string `mapstructure:"filter"`

	// CTSUserDefinedMeta is metadata added to a service automated by CTS for
	// network infrastructure automation.
	CTSUserDefinedMeta map[string]string `mapstructure:"cts_user_defined_meta"`
}

// ServiceConfigs is a collection of ServiceConfig
type ServiceConfigs []*ServiceConfig

// ServicesMeta is a useful type to abstract from the nested map of string which
// represents the user defined meta for each service a task monitors
type ServicesMeta map[string]map[string]string

// Copy returns a deep copy of this configuration.
func (c *ServiceConfig) Copy() *ServiceConfig {
	if c == nil {
		return nil
	}

	var o ServiceConfig
	o.Datacenter = StringCopy(c.Datacenter)
	o.Description = StringCopy(c.Description)
	o.ID = StringCopy(c.ID)
	o.Name = StringCopy(c.Name)
	o.Namespace = StringCopy(c.Namespace)
	o.Tag = StringCopy(c.Tag)
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
func (c *ServiceConfig) Merge(o *ServiceConfig) *ServiceConfig {
	if c == nil {
		if o == nil {
			return nil
		}
		return o.Copy()
	}

	if o == nil {
		return c.Copy()
	}

	r := c.Copy()

	if o.Datacenter != nil {
		r.Datacenter = StringCopy(o.Datacenter)
	}

	if o.Description != nil {
		r.Description = StringCopy(o.Description)
	}

	if o.ID != nil {
		r.ID = StringCopy(o.ID)
	}

	if o.Name != nil {
		r.Name = StringCopy(o.Name)
	}

	if o.Namespace != nil {
		r.Namespace = StringCopy(o.Namespace)
	}

	if o.Tag != nil {
		r.Tag = StringCopy(o.Tag)
	}

	if o.Filter != nil {
		r.Filter = StringCopy(o.Filter)
	}

	if o.CTSUserDefinedMeta != nil {
		if r.CTSUserDefinedMeta == nil {
			r.CTSUserDefinedMeta = make(map[string]string)
		}
		for k, v := range o.CTSUserDefinedMeta {
			r.CTSUserDefinedMeta[k] = v
		}
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *ServiceConfig) Finalize() {
	if c == nil {
		return
	}

	if c.Datacenter == nil {
		c.Datacenter = String("")
	}

	if c.Description == nil {
		c.Description = String("")
	}

	if c.Name == nil {
		c.Name = String("")
	}

	if c.ID == nil {
		c.ID = StringCopy(c.Name)
	}

	if c.Namespace == nil {
		c.Namespace = String("")
	}

	if c.Tag == nil {
		c.Tag = String("")
	} else {
		log.Println("[WARN] (config) The 'tag' attribute was marked for " +
			"deprecation in v0.2.0 and will be removed in v0.4.0 " +
			"of Consul-Terraform-Sync. Please update your configuration to " +
			"use 'filter' and provide a filter expression using the " +
			"Service.Tags selector.")
	}

	if c.Filter == nil {
		c.Filter = String("")
	}

	if c.CTSUserDefinedMeta == nil {
		c.CTSUserDefinedMeta = make(map[string]string)
	}
}

// Validate validates the values and nested values of the configuration struct
func (c *ServiceConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("missing service configuration")
	}

	if c.Name == nil || len(*c.Name) == 0 {
		return fmt.Errorf("logical name for the Consul service is required")
	}

	return nil
}

// GoString defines the printable version of this struct.
// Sensitive information is redacted.
func (c *ServiceConfig) GoString() string {
	if c == nil {
		return "(*ServiceConfig)(nil)"
	}

	return fmt.Sprintf("&ServiceConfig{"+
		"Name:%s, "+
		"Namespace:%s, "+
		"Datacenter:%s, "+
		"Tag:%s, "+
		"Filter:%s, "+
		"Description:%s, "+
		"CTSUserDefinedMeta:%s"+
		"}",
		StringVal(c.Name),
		StringVal(c.Namespace),
		StringVal(c.Datacenter),
		StringVal(c.Tag),
		StringVal(c.Filter),
		StringVal(c.Description),
		c.CTSUserDefinedMeta,
	)
}

// DefaultServiceConfigs returns a configuration that is populated with the
// default values.
func DefaultServiceConfigs() *ServiceConfigs {
	return &ServiceConfigs{}
}

// Len is a helper method to get the length of the underlying config list
func (c *ServiceConfigs) Len() int {
	if c == nil {
		return 0
	}

	return len(*c)
}

// Copy returns a deep copy of this configuration.
func (c *ServiceConfigs) Copy() *ServiceConfigs {
	if c == nil {
		return nil
	}

	o := make(ServiceConfigs, c.Len())
	for i, t := range *c {
		o[i] = t.Copy()
	}
	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *ServiceConfigs) Merge(o *ServiceConfigs) *ServiceConfigs {
	if c == nil {
		if o == nil {
			return nil
		}
		return o.Copy()
	}

	if o == nil {
		return c.Copy()
	}

	r := c.Copy()

	*r = append(*r, *o...)

	return r
}

// Finalize ensures the configuration has no nil pointers and sets default
// values.
func (c *ServiceConfigs) Finalize() {
	if c == nil {
		*c = *DefaultServiceConfigs()
	}

	for _, t := range *c {
		t.Finalize()
	}
}

// Validate validates the values and nested values of the configuration struct
func (c *ServiceConfigs) Validate() error {
	if c == nil {
		return fmt.Errorf("missing services configuration")
	}

	ids := make(map[string]bool)
	for _, s := range *c {
		if err := s.Validate(); err != nil {
			return err
		}

		id := *s.Name
		if s.ID != nil {
			id = *s.ID
		}
		if ids[id] {
			return fmt.Errorf("unique service IDs are required: %s", id)
		}
		ids[id] = true
	}

	return nil
}

// CTSUserDefinedMeta generates a map of service name to user defined metadata
// from a list of service IDs or service names.
func (c *ServiceConfigs) CTSUserDefinedMeta(serviceList []string) ServicesMeta {
	if c == nil {
		return nil
	}

	services := make(map[string]bool)
	for _, s := range serviceList {
		services[s] = true
	}

	m := make(ServicesMeta)
	for _, s := range *c {
		if len(s.CTSUserDefinedMeta) == 0 {
			continue
		}

		serviceName := *s.Name
		if StringPresent(s.ID) {
			if _, ok := services[*s.ID]; ok {
				m[serviceName] = s.CTSUserDefinedMeta
				continue
			}
		}

		if _, ok := services[serviceName]; ok {
			if !StringPresent(s.ID) {
				m[serviceName] = s.CTSUserDefinedMeta
			}
		}
	}
	return m
}

// GoString defines the printable version of this struct.
func (c *ServiceConfigs) GoString() string {
	if c == nil {
		return "(*ServiceConfigs)(nil)"
	}

	s := make([]string, len(*c))
	for i, t := range *c {
		s[i] = t.GoString()
	}

	return "{" + strings.Join(s, ", ") + "}"
}
