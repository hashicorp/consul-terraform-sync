package config

import "fmt"

const (
	// DefaultConsulAddress is the default address to connect with Consul
	DefaultConsulAddress = "localhost:8500"

	// DefaultConsulKVPath is the default Consul KV path to use for Sync
	// KV operations.
	DefaultConsulKVPath = "consul-terraform-sync/"
)

// ConsulConfig is the configuration for Consul client.
type ConsulConfig struct {
	// Address is the address of the Consul server. It may be an IP or FQDN.
	Address *string `mapstructure:"address"`

	// Auth is the HTTP basic authentication for communicating with Consul.
	Auth *AuthConfig `mapstructure:"auth"`

	// KVNamespace is the optional namespace for Sync to use for Consul KV
	// queries and operations.
	KVNamespace *string `mapstructure:"kv_namespace"`

	// KVPath is the directory in the Consul KV store to use for storing run time
	// data
	KVPath *string `mapstructure:"kv_path"`

	// TLS indicates we should use a secure connection while talking to
	// Consul. This requires Consul to be configured to serve HTTPS.
	TLS *TLSConfig `mapstructure:"tls"`

	// Token is the token to communicate with Consul securely.
	Token *string `mapstructure:"token"`

	// Transport configures the low-level network connection details.
	Transport *TransportConfig `mapstructure:"transport"`
}

// DefaultConsulConfig returns the default configuration struct
func DefaultConsulConfig() *ConsulConfig {
	return &ConsulConfig{
		Address:   String(DefaultConsulAddress),
		Auth:      DefaultAuthConfig(),
		KVPath:    String(DefaultConsulKVPath),
		TLS:       DefaultTLSConfig(),
		Transport: DefaultTransportConfig(),
	}
}

// Copy returns a deep copy of this configuration.
func (c *ConsulConfig) Copy() *ConsulConfig {
	if c == nil {
		return nil
	}

	var o ConsulConfig

	o.Address = StringCopy(c.Address)

	if c.Auth != nil {
		o.Auth = c.Auth.Copy()
	}

	o.KVNamespace = StringCopy(c.KVNamespace)

	o.KVPath = StringCopy(c.KVPath)

	if c.TLS != nil {
		o.TLS = c.TLS.Copy()
	}

	o.Token = StringCopy(c.Token)

	if c.Transport != nil {
		o.Transport = c.Transport.Copy()
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

	if o.Auth != nil {
		r.Auth = r.Auth.Merge(o.Auth)
	}

	if o.KVNamespace != nil {
		r.KVNamespace = StringCopy(o.KVNamespace)
	}

	if o.KVPath != nil {
		r.KVPath = StringCopy(o.KVPath)
	}

	if o.TLS != nil {
		r.TLS = r.TLS.Merge(o.TLS)
	}

	if o.Token != nil {
		r.Token = StringCopy(o.Token)
	}

	if o.Transport != nil {
		r.Transport = r.Transport.Merge(o.Transport)
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *ConsulConfig) Finalize() {
	if c == nil {
		return
	}

	if c.Address == nil {
		c.Address = stringFromEnv([]string{
			"CONSUL_HTTP_ADDR",
		}, DefaultConsulAddress)
	}

	if c.Auth == nil {
		c.Auth = DefaultAuthConfig()
	}
	c.Auth.Finalize()

	if c.KVNamespace == nil {
		c.KVNamespace = String("")
	}

	if c.KVPath == nil {
		c.KVPath = String(DefaultConsulKVPath)
	}

	if c.TLS == nil {
		c.TLS = DefaultTLSConfig()
	}
	c.TLS.Finalize()

	if c.Token == nil {
		c.Token = stringFromEnv([]string{
			"CONSUL_TOKEN",
			"CONSUL_HTTP_TOKEN",
		}, "")
	}

	if c.Transport == nil {
		c.Transport = DefaultTransportConfig()
	}
	c.Transport.Finalize()
}

// GoString defines the printable version of this struct.
// Sensitive information is redacted.
func (c *ConsulConfig) GoString() string {
	if c == nil {
		return "(*ConsulConfig)(nil)"
	}

	return fmt.Sprintf("&ConsulConfig{"+
		"Address:%s, "+
		"Auth:%s, "+
		"KVNamespace:%s, "+
		"KVPath:%s, "+
		"TLS:%s, "+
		"Token:%s, "+
		"Transport:%s"+
		"}",
		StringVal(c.Address),
		c.Auth.GoString(),
		StringVal(c.KVNamespace),
		StringVal(c.KVPath),
		c.TLS.GoString(),
		sensitiveGoString(c.Token),
		c.Transport.GoString(),
	)
}

// Env returns an environment map of supported Consul configuration
func (c *ConsulConfig) Env() map[string]string {
	if c == nil {
		return nil
	}

	env := make(map[string]string)
	if val := StringVal(c.Address); val != "" {
		env["CONSUL_HTTP_ADDR"] = val
	}
	if val := StringVal(c.Token); val != "" {
		env["CONSUL_HTTP_TOKEN"] = val
	}
	if c.TLS != nil {
		for k, v := range c.TLS.ConsulEnv() {
			env[k] = v
		}
	}

	return env
}
