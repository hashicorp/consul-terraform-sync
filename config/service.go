package config

import (
	"fmt"
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
	Tag *string `mapstructure:"tag"`
}

// ServiceConfigs is a collection of ServiceConfig
type ServiceConfigs []*ServiceConfig

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
		"Description:%s"+
		"}",
		StringVal(c.Name),
		StringVal(c.Namespace),
		StringVal(c.Datacenter),
		StringVal(c.Tag),
		StringVal(c.Description),
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
