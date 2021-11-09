package config

import (
	"crypto/tls"
	"fmt"
)

const (
	DefaultMutualTLSVerify = false
)

// CTSTLSConfig is the configuration for TLS and mutual TLS
// on the CTS API.
type CTSTLSConfig struct {
	Enabled        *bool   `mapstructure:"enabled"`
	Cert           *string `mapstructure:"cert"`
	Key            *string `mapstructure:"key"`
	VerifyIncoming *bool   `mapstructure:"verify_incoming"`
	CACert         *string `mapstructure:"ca_cert"`
	CAPath         *string `mapstructure:"ca_path"`
}

// DefaultCTSTLSConfig returns a configuration that is populated with the
// default values.
func DefaultCTSTLSConfig() *CTSTLSConfig {
	return &CTSTLSConfig{}
}

// Copy returns a deep copy of this configuration.
func (c *CTSTLSConfig) Copy() *CTSTLSConfig {
	if c == nil {
		return nil
	}

	var o CTSTLSConfig
	o.CACert = StringCopy(c.CACert)
	o.CAPath = StringCopy(c.CAPath)
	o.Cert = StringCopy(c.Cert)
	o.Enabled = BoolCopy(c.Enabled)
	o.Key = StringCopy(c.Key)
	o.VerifyIncoming = BoolCopy(c.VerifyIncoming)
	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *CTSTLSConfig) Merge(o *CTSTLSConfig) *CTSTLSConfig {
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

	if o.VerifyIncoming != nil {
		r.VerifyIncoming = BoolCopy(o.VerifyIncoming)
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *CTSTLSConfig) Finalize() {
	if c.Enabled == nil {
		c.Enabled = Bool(false ||
			StringPresent(c.Cert) ||
			StringPresent(c.CACert) ||
			StringPresent(c.CAPath) ||
			StringPresent(c.Key) ||
			BoolPresent(c.VerifyIncoming))
	}
	if c.CACert == nil {
		c.CACert = String("")
	}
	if c.CAPath == nil {
		c.CAPath = String("")
	}
	if c.Cert == nil {
		c.Cert = String("")
	}
	if c.Key == nil {
		c.Key = String("")
	}
	if c.VerifyIncoming == nil {
		c.VerifyIncoming = Bool(DefaultMutualTLSVerify)
	}
}

// GoString defines the printable version of this struct.
func (c *CTSTLSConfig) GoString() string {
	if c == nil {
		return "(*CTSTLSConfig)(nil)"
	}

	return fmt.Sprintf("&CTSTLSConfig{"+
		"CACert:%s, "+
		"CAPath:%s, "+
		"Cert:%s, "+
		"Enabled:%v, "+
		"Key:%s, "+
		"VerifyIncoming:%v"+
		"}",
		StringVal(c.CACert),
		StringVal(c.CAPath),
		StringVal(c.Cert),
		BoolVal(c.Enabled),
		StringVal(c.Key),
		BoolVal(c.VerifyIncoming),
	)
}

// Validates TLS configuration for serving the CTS API
func (c *CTSTLSConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}

	// TLS validation
	if (StringVal(c.Cert) != "") && (StringVal(c.Key) == "") {
		return fmt.Errorf("key is required if cert is configured")
	}

	if (StringVal(c.Cert) != "") && (StringVal(c.Key) != "") {
		if _, err := tls.LoadX509KeyPair(*c.Cert, *c.Key); err != nil {
			return err
		}
	}

	if BoolVal(c.Enabled) && ((StringVal(c.Key) == "") || (StringVal(c.Cert) == "")) {
		return fmt.Errorf("key and cert are required if TLS is enabled")
	}

	// mTLS validation
	if BoolVal(c.VerifyIncoming) && ((StringVal(c.Key) == "") || (StringVal(c.Cert) == "")) {
		return fmt.Errorf("key and cert are required if verify_incoming is enabled")
	}

	return nil
}
