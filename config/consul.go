package config

import "fmt"

const (
	// DefaultConsulAddress is the default address to connect with Consul
	DefaultConsulAddress = "localhost:8500"

	// DefaultConsulKVPath is the default Consul KV path to use for NIA
	// KV operations.
	DefaultConsulKVPath = "consul-nia/"
)

// ConsulConfig is the configuration for Consul client.
type ConsulConfig struct {
	Address     *string     `mapstructure:"address"`
	Token       *string     `mapstructure:"token"`
	Auth        *AuthConfig `mapstructure:"auth"`
	TLS         *TLSConfig  `mapstructure:"tls"`
	KVPath      *string     `mapstructure:"kv_path"`
	KVNamespace *string     `mapstructure:"kv_namespace"`
}

// DefaultConsulConfig returns the default configuration struct
func DefaultConsulConfig() *ConsulConfig {
	return &ConsulConfig{
		Address: String(DefaultConsulAddress),
		Auth:    DefaultAuthConfig(),
		TLS:     DefaultTLSConfig(),
		KVPath:  String(DefaultConsulKVPath),
	}
}

// Copy returns a deep copy of this configuration.
func (c *ConsulConfig) Copy() *ConsulConfig {
	if c == nil {
		return nil
	}

	var o ConsulConfig

	o.Address = StringCopy(c.Address)

	o.Token = StringCopy(c.Token)

	if c.Auth != nil {
		o.Auth = c.Auth.Copy()
	}

	o.KVPath = StringCopy(c.KVPath)

	o.KVNamespace = StringCopy(c.KVNamespace)

	if c.TLS != nil {
		o.TLS = c.TLS.Copy()
	}

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *ConsulConfig) Merge(o *ConsulConfig) *ConsulConfig {
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

	if o.Address != nil {
		r.Address = StringCopy(o.Address)
	}

	if o.Token != nil {
		r.Token = StringCopy(o.Token)
	}

	if o.Auth != nil {
		r.Auth = r.Auth.Merge(o.Auth)
	}

	if o.TLS != nil {
		r.TLS = r.TLS.Merge(o.TLS)
	}

	if o.KVPath != nil {
		r.KVPath = StringCopy(o.KVPath)
	}

	if o.KVNamespace != nil {
		r.KVNamespace = StringCopy(o.KVNamespace)
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *ConsulConfig) Finalize() {
	if c == nil {
		return
	}

	if c.Address == nil {
		c.Address = String(DefaultConsulAddress)
	}

	if c.Token == nil {
		c.Token = stringFromEnv([]string{
			"CONSUL_TOKEN",
			"CONSUL_HTTP_TOKEN",
		}, "")
	}

	if c.Auth == nil {
		c.Auth = DefaultAuthConfig()
	}
	c.Auth.Finalize()

	if c.TLS == nil {
		c.TLS = DefaultTLSConfig()
	}
	c.TLS.Finalize()

	if c.KVPath == nil {
		c.KVPath = String(DefaultConsulKVPath)
	}

	if c.KVNamespace == nil {
		c.KVNamespace = String("")
	}
}

// GoString defines the printable version of this struct.
// Sensitive information is redacted.
func (c *ConsulConfig) GoString() string {
	if c == nil {
		return "(*ConsulConfig)(nil)"
	}

	return fmt.Sprintf("&ConsulConfig{"+
		"Address:%s, "+
		"Token:%s, "+
		"Auth:%s, "+
		"TLS:%s, "+
		"KVPath:%s, "+
		"KVNamespace:%s"+
		"}",
		StringVal(c.Address),
		senstiveGoString(c.Token),
		c.Auth.GoString(),
		c.TLS.GoString(),
		StringVal(c.KVPath),
		StringVal(c.KVNamespace),
	)
}
