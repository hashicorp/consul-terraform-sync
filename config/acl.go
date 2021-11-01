package config

import (
	"fmt"
)

const (
	// TODO: This will be changed to true in a later release
	defaultACLIsEnabled = false
)

// ACLConfig is the configuration for an Access Control List (ACL).
type ACLConfig struct {
	// Boolean if ACLs are enabled or not. True if enabled, false otherwise.
	Enabled *bool `mapstructure:"enabled"`

	// The list of tokens supported for this CTS instance
	Tokens *TokensConfig `mapstructure:"tokens"`
}

// DefaultACLConfig returns the default configuration struct.
func DefaultACLConfig() *ACLConfig {
	return &ACLConfig{
		Enabled: Bool(defaultACLIsEnabled),
	}
}

// Copy returns a deep copy of this configuration.
func (c *ACLConfig) Copy() *ACLConfig {
	if c == nil {
		return nil
	}

	var o ACLConfig
	o.Enabled = BoolCopy(c.Enabled)
	o.Tokens = c.Tokens.Copy()

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *ACLConfig) Merge(o *ACLConfig) *ACLConfig {
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

	if o.Tokens != nil {
		r.Tokens = o.Tokens.Copy()
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *ACLConfig) Finalize() {
	if c == nil {
		return
	}

	if c.Enabled == nil {
		c.Enabled = Bool(defaultACLIsEnabled)
	}

	if c.Tokens == nil {
		c.Tokens = &TokensConfig{}
	}
	c.Tokens.Finalize()
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *ACLConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("missing acl configuration")
	}

	if c.Enabled == nil {
		return fmt.Errorf("enabled must not be nil")
	}

	if c.Tokens == nil {
		return fmt.Errorf("tokens must not be nil")
	}

	return nil
}

// GoString defines the printable version of this struct.
// Sensitive information is redacted.
func (c *ACLConfig) GoString() string {
	if c == nil {
		return "(*ACLConfig)(nil)"
	}

	return fmt.Sprintf("&ACLConfig{"+
		"Enabled:%t, "+
		"Tokens:%v, "+
		"}",
		BoolVal(c.Enabled),
		c.Tokens.GoString(),
	)
}
