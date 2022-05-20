package config

import "fmt"

const (
	DefaultServiceName = "Consul-Terraform-Sync"
)

// ServiceRegistrationConfig is a configuration that controls how CTS will
// register itself as a service with Consul.
type ServiceRegistrationConfig struct {
	Enabled      *bool               `mapstructure:"enabled"`
	ServiceName  *string             `mapstructure:"service_name"`
	Address      *string             `mapstructure:"address"`
	Namespace    *string             `mapstructure:"namespace"`
	DefaultCheck *DefaultCheckConfig `mapstructure:"default_check"`
}

// DefaultServiceRegistrationConfig returns a ServiceRegistrationConfig with
// default values.
func DefaultServiceRegistrationConfig() *ServiceRegistrationConfig {
	return &ServiceRegistrationConfig{
		Enabled:     Bool(true),
		ServiceName: String(DefaultServiceName),
		Namespace:   String(""),
		Address:     String(""),
		DefaultCheck: &DefaultCheckConfig{
			Enabled: Bool(true),
			Address: String(""),
		},
	}
}

// Copy returns a deep copy of this configuration.
func (c *ServiceRegistrationConfig) Copy() *ServiceRegistrationConfig {
	if c == nil {
		return nil
	}

	var o ServiceRegistrationConfig
	o.Enabled = BoolCopy(c.Enabled)
	o.ServiceName = StringCopy(c.ServiceName)
	o.Address = StringCopy(c.Address)
	o.Namespace = StringCopy(c.Namespace)

	if c.DefaultCheck != nil {
		o.DefaultCheck = c.DefaultCheck.Copy()
	}

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
func (c *ServiceRegistrationConfig) Merge(o *ServiceRegistrationConfig) *ServiceRegistrationConfig {
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

	if o.Enabled != nil {
		r.Enabled = BoolCopy(o.Enabled)
	}

	if o.ServiceName != nil {
		r.ServiceName = StringCopy(o.ServiceName)
	}

	if o.Address != nil {
		r.Address = StringCopy(o.Address)
	}

	if o.Namespace != nil {
		r.Namespace = StringCopy(o.Namespace)
	}

	if o.DefaultCheck != nil {
		r.DefaultCheck = r.DefaultCheck.Merge(o.DefaultCheck)
	}

	return r
}

// Finalize ensures that the receiver contains no nil pointers.
func (c *ServiceRegistrationConfig) Finalize() {
	if c.Enabled == nil {
		c.Enabled = Bool(true)
	}

	if c.ServiceName == nil {
		c.ServiceName = String(DefaultServiceName)
	}

	if c.Address == nil {
		c.Address = String("")
	}

	if c.Namespace == nil {
		c.Namespace = String("")
	}

	if c.DefaultCheck == nil {
		c.DefaultCheck = &DefaultCheckConfig{}
		c.DefaultCheck.Finalize()
	}
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *ServiceRegistrationConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}

	if c.DefaultCheck != nil {
		if err := c.DefaultCheck.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// GoString defines the printable version of this struct.
func (c *ServiceRegistrationConfig) GoString() string {
	if c == nil {
		return "(*ServiceRegistrationConfig)(nil)"
	}

	return fmt.Sprintf("&ServiceRegistrationConfig{"+
		"Enabled:%v, "+
		"ServiceName:%s, "+
		"Address:%s, "+
		"Namespace:%s, "+
		"DefaultCheck: %s"+
		"}",
		BoolVal(c.Enabled),
		StringVal(c.ServiceName),
		StringVal(c.Address),
		StringVal(c.Namespace),
		c.DefaultCheck.GoString(),
	)
}
