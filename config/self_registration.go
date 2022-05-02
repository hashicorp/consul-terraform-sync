package config

import "fmt"

// SelfRegistrationConfig is a configuration that controls how CTS will
// self-register itself as a service with Consul.
type SelfRegistrationConfig struct {
	Enabled   *bool   `mapstructure:"enabled"`
	Namespace *string `mapstructure:"namespace"`
}

// DefaultSelfRegistrationConfig returns a SelfRegistrationConfig with
// default values.
func DefaultSelfRegistrationConfig() *SelfRegistrationConfig {
	return &SelfRegistrationConfig{
		Enabled:   Bool(true),
		Namespace: String(""),
	}
}

// Copy returns a deep copy of this configuration.
func (c *SelfRegistrationConfig) Copy() *SelfRegistrationConfig {
	if c == nil {
		return nil
	}

	var o SelfRegistrationConfig
	o.Enabled = BoolCopy(c.Enabled)
	o.Namespace = StringCopy(c.Namespace)

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

	if o.Namespace != nil {
		r.Namespace = StringCopy(o.Namespace)
	}

	return r
}

// Finalize ensures that the receiver contains no nil pointers.
func (c *SelfRegistrationConfig) Finalize() {
	if c.Enabled == nil {
		c.Enabled = Bool(true)
	}

	if c.Namespace == nil {
		c.Namespace = String("")
	}
}

// GoString defines the printable version of this struct.
func (c *SelfRegistrationConfig) GoString() string {
	if c == nil {
		return "(*SelfRegistrationConfig)(nil)"
	}

	return fmt.Sprintf("&SelfRegistrationConfig{"+
		"Enabled:%v, "+
		"Namespace:%s"+
		"}",
		BoolVal(c.Enabled),
		StringVal(c.Namespace),
	)
}
