package config

import "fmt"

const (
	// DefaultTLSVerify is the default value for TLS verification.
	DefaultTLSVerify = true
)

// TLSConfig is the configuration for TLS.
type TLSConfig struct {
	CACert     *string `mapstructure:"ca_cert"`
	CAPath     *string `mapstructure:"ca_path"`
	Cert       *string `mapstructure:"cert"`
	Enabled    *bool   `mapstructure:"enabled"`
	Key        *string `mapstructure:"key"`
	ServerName *string `mapstructure:"server_name"`
	Verify     *bool   `mapstructure:"verify"`
}

// DefaultTLSConfig returns a configuration that is populated with the
// default values.
func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{}
}

// Copy returns a deep copy of this configuration.
func (c *TLSConfig) Copy() *TLSConfig {
	if c == nil {
		return nil
	}

	var o TLSConfig
	o.CACert = StringCopy(c.CACert)
	o.CAPath = StringCopy(c.CAPath)
	o.Cert = StringCopy(c.Cert)
	o.Enabled = BoolCopy(c.Enabled)
	o.Key = StringCopy(c.Key)
	o.ServerName = StringCopy(c.ServerName)
	o.Verify = BoolCopy(c.Verify)
	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *TLSConfig) Merge(o *TLSConfig) *TLSConfig {
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

	if o.Cert != nil {
		r.Cert = StringCopy(o.Cert)
	}

	if o.CACert != nil {
		r.CACert = StringCopy(o.CACert)
	}

	if o.CAPath != nil {
		r.CAPath = StringCopy(o.CAPath)
	}

	if o.Enabled != nil {
		r.Enabled = BoolCopy(o.Enabled)
	}

	if o.Key != nil {
		r.Key = StringCopy(o.Key)
	}

	if o.ServerName != nil {
		r.ServerName = StringCopy(o.ServerName)
	}

	if o.Verify != nil {
		r.Verify = BoolCopy(o.Verify)
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *TLSConfig) Finalize() {
	if c.Enabled == nil {
		c.Enabled = Bool(false ||
			StringPresent(c.Cert) ||
			StringPresent(c.CACert) ||
			StringPresent(c.CAPath) ||
			StringPresent(c.Key) ||
			StringPresent(c.ServerName) ||
			BoolPresent(c.Verify))
	}

	if c.Cert == nil {
		c.Cert = stringFromEnv([]string{
			"CONSUL_CLIENT_CERT",
		}, "")
	}

	if c.CACert == nil {
		c.CACert = stringFromEnv([]string{
			"CONSUL_CACERT",
		}, "")
	}

	if c.CAPath == nil {
		c.CAPath = stringFromEnv([]string{
			"CONSUL_CAPATH",
		}, "")
	}

	if c.Key == nil {
		c.Key = stringFromEnv([]string{
			"CONSUL_CLIENT_KEY",
		}, "")
	}

	if c.ServerName == nil {
		c.ServerName = stringFromEnv([]string{
			"CONSUL_TLS_SERVER_NAME",
		}, "")
	}

	if c.Verify == nil {
		c.Verify = Bool(DefaultTLSVerify)
	}
}

// GoString defines the printable version of this struct.
func (c *TLSConfig) GoString() string {
	if c == nil {
		return "(*TLSConfig)(nil)"
	}

	return fmt.Sprintf("&TLSConfig{"+
		"CACert:%s, "+
		"CAPath:%s, "+
		"Cert:%s, "+
		"Enabled:%v, "+
		"Key:%s, "+
		"ServerName:%s, "+
		"Verify:%v"+
		"}",
		StringVal(c.CACert),
		StringVal(c.CAPath),
		StringVal(c.Cert),
		BoolVal(c.Enabled),
		StringVal(c.Key),
		StringVal(c.ServerName),
		BoolVal(c.Verify),
	)
}

// ConsulEnv returns an environment map of the TLS configuration for Consul
func (c *TLSConfig) ConsulEnv() map[string]string {
	env := make(map[string]string)

	if !BoolVal(c.Enabled) {
		return env
	}

	if val := StringVal(c.Cert); val != "" {
		env["CONSUL_CLIENT_CERT"] = val
	}

	if val := StringVal(c.CACert); val != "" {
		env["CONSUL_CACERT"] = val
	}

	if val := StringVal(c.CAPath); val != "" {
		env["CONSUL_CAPATH"] = val
	}

	if val := StringVal(c.Key); val != "" {
		env["CONSUL_CLIENT_KEY"] = val
	}

	if val := StringVal(c.ServerName); val != "" {
		env["CONSUL_TLS_SERVER_NAME"] = val
	}

	return env
}
