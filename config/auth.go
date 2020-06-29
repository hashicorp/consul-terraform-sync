package config

import "fmt"

// AuthConfig is the HTTP basic authentication data.
type AuthConfig struct {
	Enabled  *bool   `mapstructure:"enabled"`
	Username *string `mapstructure:"username"`
	Password *string `mapstructure:"password"`
}

// DefaultAuthConfig is the default configuration.
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{}
}

// Copy returns a deep copy of this configuration.
func (c *AuthConfig) Copy() *AuthConfig {
	if c == nil {
		return nil
	}

	var o AuthConfig
	o.Enabled = BoolCopy(c.Enabled)
	o.Username = StringCopy(c.Username)
	o.Password = StringCopy(c.Password)
	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *AuthConfig) Merge(o *AuthConfig) *AuthConfig {
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

	if o.Username != nil {
		r.Username = StringCopy(o.Username)
	}

	if o.Password != nil {
		r.Password = StringCopy(o.Password)
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *AuthConfig) Finalize() {
	if c.Enabled == nil {
		c.Enabled = Bool(false ||
			StringPresent(c.Username) ||
			StringPresent(c.Password))
	}

	if c.Username == nil {
		c.Username = String("")
	}

	if c.Password == nil {
		c.Password = String("")
	}

	if c.Enabled == nil {
		c.Enabled = Bool(*c.Username != "" || *c.Password != "")
	}
}

// GoString defines the printable version of this struct.
// Sensitive information is redacted.
func (c *AuthConfig) GoString() string {
	if c == nil {
		return "(*AuthConfig)(nil)"
	}

	return fmt.Sprintf("&AuthConfig{"+
		"Enabled:%v, "+
		"Username:%s, "+
		"Password:%s"+
		"}",
		BoolVal(c.Enabled),
		StringVal(c.Username),
		senstiveGoString(c.Password),
	)
}

func senstiveGoString(s *string) string {
	if StringPresent(s) {
		return "(redacted)"
	}

	return ""
}
