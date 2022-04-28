package config

import "fmt"

type SelfRegistrationConfig struct {
	Enabled   *bool   `mapstructure:"enabled"`
	Namespace *string `mapstructure:"namespace"`
}

func DefaultSelfRegistrationConfig() *SelfRegistrationConfig {
	return &SelfRegistrationConfig{
		Enabled:   Bool(true),
		Namespace: String(""),
	}
}

func (c *SelfRegistrationConfig) Copy() *SelfRegistrationConfig {
	if c == nil {
		return nil
	}

	var o SelfRegistrationConfig
	o.Enabled = BoolCopy(c.Enabled)
	o.Namespace = StringCopy(c.Namespace)

	return &o
}

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

func (c *SelfRegistrationConfig) Finalize() {
	if c.Enabled == nil {
		c.Enabled = Bool(true)
	}

	if c.Namespace == nil {
		c.Namespace = String("")
	}
}

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
