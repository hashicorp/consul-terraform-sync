package config

import (
	"fmt"
	"time"
)

const (
	// DefaultDialKeepAlive is the default amount of time to keep alive
	// connections.
	DefaultDialKeepAlive = 30 * time.Second

	// DefaultDialTimeout is the amount of time to attempt to dial before timing
	// out.
	DefaultDialTimeout = 30 * time.Second

	// DefaultIdleConnTimeout is the default connection timeout for idle
	// connections.
	DefaultIdleConnTimeout = 5 * time.Second

	// DefaultMaxIdleConns is the default number of maximum idle connections.
	DefaultMaxIdleConns = 0

	// DefaultMaxIdleConnsPerHost is the default number of maximum idle
	// connections per host.
	DefaultMaxIdleConnsPerHost = 100

	// DefaultTLSHandshakeTimeout is the amount of time to negotiate the TLS
	// handshake.
	DefaultTLSHandshakeTimeout = 10 * time.Second
)

// TransportConfig is the configuration to tune low-level APIs for the
// interactions on the wire.
type TransportConfig struct {
	// DialKeepAlive is the amount of time for keep-alives.
	DialKeepAlive *time.Duration `mapstructure:"dial_keep_alive"`

	// DialTimeout is the amount of time to wait to establish a connection.
	DialTimeout *time.Duration `mapstructure:"dial_timeout"`

	// DisableKeepAlives determines if keep-alives should be used. Disabling this
	// significantly decreases performance.
	DisableKeepAlives *bool `mapstructure:"disable_keep_alives"`

	// IdleConnTimeout is the timeout for idle connections.
	IdleConnTimeout *time.Duration `mapstructure:"idle_conn_timeout"`

	// MaxIdleConns is the maximum number of total idle connections.
	MaxIdleConns *int `mapstructure:"max_idle_conns"`

	// MaxIdleConnsPerHost is the maximum number of idle connections per remote
	// host.
	//
	// The majority of connections are established with one host, the Consul agent.
	// To achieve the shortest latency between a Consul service update to a task
	// execution, configure the max_idle_conns_per_host to a number proportional to
	// the number of services in automation across all tasks.
	//
	// This value must be lower than the configured http_max_conns_per_client
	// for the Consul agent. Note that requests made by Terraform subprocesses
	// or any other process on the same host as Consul-Terraform-Sync will
	// contribute to the Consul agent http_max_conns_per_client.
	//
	// If max_idle_conns_per_host or the number of services in automation is greater
	// than the Consul agent limit, Consul-Terraform-Sync may error due to
	// connection limits (429).
	MaxIdleConnsPerHost *int `mapstructure:"max_idle_conns_per_host"`

	// TLSHandshakeTimeout is the amount of time to wait to complete the TLS
	// handshake.
	TLSHandshakeTimeout *time.Duration `mapstructure:"tls_handshake_timeout"`
}

// DefaultTransportConfig returns a configuration that is populated with the
// default values.
func DefaultTransportConfig() *TransportConfig {
	return &TransportConfig{}
}

// Copy returns a deep copy of this configuration.
func (c *TransportConfig) Copy() *TransportConfig {
	if c == nil {
		return nil
	}

	var o TransportConfig

	o.DialKeepAlive = c.DialKeepAlive
	o.DialTimeout = c.DialTimeout
	o.DisableKeepAlives = c.DisableKeepAlives
	o.IdleConnTimeout = c.IdleConnTimeout
	o.MaxIdleConns = c.MaxIdleConns
	o.MaxIdleConnsPerHost = c.MaxIdleConnsPerHost
	o.TLSHandshakeTimeout = c.TLSHandshakeTimeout

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *TransportConfig) Merge(o *TransportConfig) *TransportConfig {
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

	if o.DialKeepAlive != nil {
		r.DialKeepAlive = o.DialKeepAlive
	}

	if o.DialTimeout != nil {
		r.DialTimeout = o.DialTimeout
	}

	if o.DisableKeepAlives != nil {
		r.DisableKeepAlives = o.DisableKeepAlives
	}

	if o.IdleConnTimeout != nil {
		r.IdleConnTimeout = o.IdleConnTimeout
	}

	if o.MaxIdleConns != nil {
		r.MaxIdleConns = o.MaxIdleConns
	}

	if o.MaxIdleConnsPerHost != nil {
		r.MaxIdleConnsPerHost = o.MaxIdleConnsPerHost
	}

	if o.TLSHandshakeTimeout != nil {
		r.TLSHandshakeTimeout = o.TLSHandshakeTimeout
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *TransportConfig) Finalize() {
	if c.DialKeepAlive == nil {
		c.DialKeepAlive = TimeDuration(DefaultDialKeepAlive)
	}

	if c.DialTimeout == nil {
		c.DialTimeout = TimeDuration(DefaultDialTimeout)
	}

	if c.DisableKeepAlives == nil {
		c.DisableKeepAlives = Bool(false)
	}

	if c.IdleConnTimeout == nil {
		c.IdleConnTimeout = TimeDuration(DefaultIdleConnTimeout)
	}

	if c.MaxIdleConns == nil {
		c.MaxIdleConns = Int(DefaultMaxIdleConns)
	}

	if c.MaxIdleConnsPerHost == nil {
		c.MaxIdleConnsPerHost = Int(DefaultMaxIdleConnsPerHost)
	}

	if c.TLSHandshakeTimeout == nil {
		c.TLSHandshakeTimeout = TimeDuration(DefaultTLSHandshakeTimeout)
	}
}

// GoString defines the printable version of this struct.
func (c *TransportConfig) GoString() string {
	if c == nil {
		return "(*TransportConfig)(nil)"
	}

	return fmt.Sprintf("&TransportConfig{"+
		"DialKeepAlive:%s, "+
		"DialTimeout:%s, "+
		"DisableKeepAlives:%t, "+
		"IdleConnTimeout:%d, "+
		"MaxIdleConns:%d, "+
		"MaxIdleConnsPerHost:%d, "+
		"TLSHandshakeTimeout:%s"+
		"}",
		TimeDurationVal(c.DialKeepAlive),
		TimeDurationVal(c.DialTimeout),
		BoolVal(c.DisableKeepAlives),
		TimeDurationVal(c.IdleConnTimeout),
		IntVal(c.MaxIdleConns),
		IntVal(c.MaxIdleConnsPerHost),
		TimeDurationVal(c.TLSHandshakeTimeout),
	)
}
