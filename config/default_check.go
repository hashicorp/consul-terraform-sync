// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"fmt"
	"net/url"
	"strings"
)

// DefaultCheckConfig is a configuration that controls whether to
// create a default health check on the CTS service in Consul. It
// also allows for some modification of the health check to account
// for different setups of CTS.
type DefaultCheckConfig struct {
	Enabled *bool   `mapstructure:"enabled"`
	Address *string `mapstructure:"address"`
}

// Copy returns a deep copy of this configuration.
func (c *DefaultCheckConfig) Copy() *DefaultCheckConfig {
	if c == nil {
		return nil
	}

	var o DefaultCheckConfig
	o.Enabled = BoolCopy(c.Enabled)
	o.Address = StringCopy(c.Address)

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
func (c *DefaultCheckConfig) Merge(o *DefaultCheckConfig) *DefaultCheckConfig {
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

	if o.Address != nil {
		r.Address = StringCopy(o.Address)
	}

	return r
}

// Finalize ensures that the receiver contains no nil pointers.
func (c *DefaultCheckConfig) Finalize() {
	if c.Enabled == nil {
		c.Enabled = Bool(true)
	}

	if c.Address == nil {
		c.Address = String("")
	}
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *DefaultCheckConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}

	if c.Address != nil && *c.Address != "" {
		if !strings.HasPrefix(strings.ToLower(*c.Address), "http") {
			// check specifically for the scheme since this can be a common error
			return fmt.Errorf("default check address must include scheme (http or https)")
		}

		_, err := url.ParseRequestURI(*c.Address)
		if err != nil {
			return fmt.Errorf("error with address for default check: %s", err)
		}
	}

	return nil
}

// GoString defines the printable version of this struct.
func (c *DefaultCheckConfig) GoString() string {
	if c == nil {
		return "(*DefaultCheckConfig)(nil)"
	}

	return fmt.Sprintf("&DefaultCheckConfig{"+
		"Enabled:%v, "+
		"Address:%s"+
		"}",
		BoolVal(c.Enabled),
		StringVal(c.Address),
	)
}
