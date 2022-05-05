package config

import "fmt"

const (
	DefaultServiceName = "Consul-Terraform-Sync"
)

// SelfRegistrationConfig is a configuration that controls how CTS will
// self-register itself as a service with Consul.
type SelfRegistrationConfig struct {
	Enabled      *bool               `mapstructure:"enabled"`
	ServiceName  *string             `mapstructure:"service_name"`
	Namespace    *string             `mapstructure:"namespace"`
	DefaultCheck *DefaultCheckConfig `mapstructure:"default_check"`
}

// DefaultSelfRegistrationConfig returns a SelfRegistrationConfig with
// default values.
func DefaultSelfRegistrationConfig() *SelfRegistrationConfig {
	return &SelfRegistrationConfig{
		Enabled:     Bool(true),
		ServiceName: String(DefaultServiceName),
		Namespace:   String(""),
		DefaultCheck: &DefaultCheckConfig{
			Enabled: Bool(true),
			Address: String(""),
		},
	}
}

// Copy returns a deep copy of this configuration.
func (c *SelfRegistrationConfig) Copy() *SelfRegistrationConfig {
	if c == nil {
		return nil
	}

	var o SelfRegistrationConfig
	o.Enabled = BoolCopy(c.Enabled)
	o.ServiceName = StringCopy(c.ServiceName)
	o.Namespace = StringCopy(c.Namespace)

	if c.DefaultCheck != nil {
		o.DefaultCheck = c.DefaultCheck.Copy()
	}

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
func (c *SelfRegistrationConfig) Merge(o *SelfRegistrationConfig) *SelfRegistrationConfig {
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

	if o.Namespace != nil {
		r.Namespace = StringCopy(o.Namespace)
	}

	if o.DefaultCheck != nil {
		r.DefaultCheck = r.DefaultCheck.Merge(o.DefaultCheck)
	}

	return r
}

// Finalize ensures that the receiver contains no nil pointers.
func (c *SelfRegistrationConfig) Finalize() {
	if c.Enabled == nil {
		c.Enabled = Bool(true)
	}

	if c.ServiceName == nil {
		c.ServiceName = String(DefaultServiceName)
	}

	if c.Namespace == nil {
		c.Namespace = String("")
	}

	if c.DefaultCheck == nil {
		c.DefaultCheck = &DefaultCheckConfig{}
		c.DefaultCheck.Finalize()
	}
}

// GoString defines the printable version of this struct.
func (c *SelfRegistrationConfig) GoString() string {
	if c == nil {
		return "(*SelfRegistrationConfig)(nil)"
	}

	return fmt.Sprintf("&SelfRegistrationConfig{"+
		"Enabled:%v, "+
		"ServiceName:%s, "+
		"Namespace:%s, "+
		"DefaultCheck: %s"+
		"}",
		BoolVal(c.Enabled),
		StringVal(c.ServiceName),
		StringVal(c.Namespace),
		c.DefaultCheck.GoString(),
	)
}
