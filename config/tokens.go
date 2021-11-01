package config

import (
	"fmt"
)

// TokensConfig is the configuration for the CTS tokens.
type TokensConfig struct {

	// Master allows operators to bootstrap the ACL system with a token Secret ID that is well-known.
	Root *string `mapstructure:"root"`
}

// Copy returns a deep copy of this configuration.
func (c *TokensConfig) Copy() *TokensConfig {
	if c == nil {
		return nil
	}

	var o TokensConfig
	o.Root = StringCopy(c.Root)

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *TokensConfig) Merge(o *TokensConfig) *TokensConfig {
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

	if o.Root != nil {
		r.Root = StringCopy(o.Root)
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *TokensConfig) Finalize() {
	if c == nil {
		return
	}

	if c.Root == nil {
		c.Root = String("")
	}
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *TokensConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("missing acl configuration")
	}

	if c.Root == nil {
		return fmt.Errorf("root token must not be nil")
	}

	return nil
}

// GoString defines the printable version of this struct.
// Sensitive information is redacted.
func (c *TokensConfig) GoString() string {
	if c == nil {
		return "(*TokensConfig)(nil)"
	}

	return fmt.Sprintf("&TokensConfig{"+
		"Root:%s, "+
		"}",
		StringVal(c.Root),
	)
}
