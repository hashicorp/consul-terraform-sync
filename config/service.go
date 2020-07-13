package config

import (
	"fmt"
	"strings"
)

// ServiceConfig defines the explicit configuration for Consul NIA to monitor
// a service. This block may be specified multiple times to configure multiple
// services.
type ServiceConfig struct {
	// Description is the human readable text to describe the service.
	Description *string `mapstructure:"description"`

	// Name is the Consul logical name of the service (required).
	Name *string `mapstructure:"name"`

	// Namespace is the namespace of the service (Consul Enterprise only). If not
	// provided, the namespace will be inferred from the Consul NIA ACL token, or
	// default to the `default` namespace.
	Namespace *string `mapstructure:"namespace"`
}

// ServiceConfigs is a collection of ServiceConfig
type ServiceConfigs []*ServiceConfig

// Copy returns a deep copy of this configuration.
func (c *ServiceConfig) Copy() *ServiceConfig {
	if c == nil {
		return nil
	}

	var o ServiceConfig
	o.Description = StringCopy(c.Description)
	o.Name = StringCopy(c.Name)
	o.Namespace = StringCopy(c.Namespace)
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

	if o.Description != nil {
		r.Description = StringCopy(o.Description)
	}

	if o.Name != nil {
		r.Name = StringCopy(o.Name)
	}

	if o.Namespace != nil {
		r.Namespace = StringCopy(o.Namespace)
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *ServiceConfig) Finalize() {
	if c == nil {
		return
	}

	if c.Description == nil {
		c.Description = String("")
	}

	if c.Name == nil {
		c.Name = String("")
	}

	if c.Namespace == nil {
		c.Namespace = String("")
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
		"Description:%s"+
		"}",
		StringVal(c.Name),
		StringVal(c.Namespace),
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

	for _, s := range *c {
		if err := s.Validate(); err != nil {
			return err
		}
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
