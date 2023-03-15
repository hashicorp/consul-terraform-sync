// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package config

import "fmt"

// DriverConfig is the configuration for the CTS driver used to execute
// infrastructure updates.
type DriverConfig struct {
	consul *ConsulConfig

	Terraform *TerraformConfig `mapstructure:"terraform"`
}

// DefaultDriverConfig returns the default configuration struct.
func DefaultDriverConfig() *DriverConfig {
	return &DriverConfig{}
}

// Copy returns a deep copy of this configuration.
func (c *DriverConfig) Copy() *DriverConfig {
	if c == nil {
		return nil
	}

	var o DriverConfig

	if c.consul != nil {
		o.consul = c.consul.Copy()
	}

	if c.Terraform != nil {
		o.Terraform = c.Terraform.Copy()
	}

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *DriverConfig) Merge(o *DriverConfig) *DriverConfig {
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

	if o.consul != nil {
		r.consul = r.consul.Merge(o.consul)
	}

	if o.Terraform != nil {
		r.Terraform = r.Terraform.Merge(o.Terraform)
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *DriverConfig) Finalize() {
	if c == nil {
		return
	}

	if c.Terraform == nil {
		c.Terraform = DefaultTerraformConfig()
	}
	c.Terraform.Finalize(c.consul)
}

// Validate validates the values and nested values of the configuration struct.
func (c *DriverConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("missing driver configuration")
	}

	return c.Terraform.Validate()
}

// GoString defines the printable version of this struct.
func (c *DriverConfig) GoString() string {
	if c == nil {
		return "(*DriverConfig)(nil)"
	}

	return fmt.Sprintf("&DriverConfig{"+
		"Terraform:%s"+
		"}",
		c.Terraform.GoString(),
	)
}
