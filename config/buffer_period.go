// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"fmt"
	"time"
)

var (
	DefaultBufferPeriodMin = 5 * time.Second
	DefaultBufferPeriodMax = 4 * DefaultBufferPeriodMin
)

// BufferPeriodConfig is the min and max duration to buffer changes for tasks
// before executing.
type BufferPeriodConfig struct {
	// Enabled determines if this buffer period is enabled.
	Enabled *bool `mapstructure:"enabled" json:"enabled"`

	// Min and Max are the minimum and maximum time, respectively, to wait for
	// data changes before rendering a new template to disk.
	Min *time.Duration `mapstructure:"min" json:"min"`
	Max *time.Duration `mapstructure:"max" json:"max"`
}

// DefaultBufferPeriodConfig is the global default configuration for all tasks.
func DefaultBufferPeriodConfig() *BufferPeriodConfig {
	return &BufferPeriodConfig{
		Enabled: Bool(true),
		Min:     &DefaultBufferPeriodMin,
		Max:     &DefaultBufferPeriodMax,
	}
}

// Copy returns a deep copy of this configuration.
func (c *BufferPeriodConfig) Copy() *BufferPeriodConfig {
	if c == nil {
		return nil
	}

	var o BufferPeriodConfig
	o.Enabled = BoolCopy(c.Enabled)
	o.Min = TimeDurationCopy(c.Min)
	o.Max = TimeDurationCopy(c.Max)
	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *BufferPeriodConfig) Merge(o *BufferPeriodConfig) *BufferPeriodConfig {
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

	if o.Min != nil {
		r.Min = TimeDurationCopy(o.Min)
	}

	if o.Max != nil {
		r.Max = TimeDurationCopy(o.Max)
	}

	return r
}

// Finalize ensures that the receiver contains no nil pointers. For nil pointers,
// Finalize sets default values where necessary
func (c *BufferPeriodConfig) Finalize() {
	c.inheritParentConfig(DefaultBufferPeriodConfig())
}

func (c *BufferPeriodConfig) inheritParentConfig(parent *BufferPeriodConfig) {
	// if disabled, fill in zero values
	if c.Enabled != nil && !*c.Enabled {
		c.Min = TimeDuration(0 * time.Second)
		c.Max = TimeDuration(0 * time.Second)
		return
	}

	// if nothing configured, default to parent
	if c.Enabled == nil && c.Min == nil && c.Max == nil {
		c.Min = parent.Min
		c.Max = parent.Max
		c.Enabled = parent.Enabled
		return
	}

	// some Min/Max info configured, assume user intention is enabled
	c.Enabled = Bool(true)

	if c.Min == nil {
		if parent.Enabled != nil && *parent.Enabled {
			// parent is enabled: use parent Min
			c.Min = parent.Min
		} else {
			// parent is disabled: use default Min
			c.Min = TimeDuration(DefaultBufferPeriodMin)
		}
	}

	if c.Max == nil {
		c.Max = TimeDuration(4 * *c.Min)
	}

	if *c.Min > *c.Max {
		*c.Max = *c.Min
	}
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *BufferPeriodConfig) Validate() error {
	if c == nil {
		// config is not required, return early
		return nil
	}

	if !BoolVal(c.Enabled) {
		return nil
	}

	if c.Min.Seconds() < 0 || c.Max.Seconds() < 0 {
		return fmt.Errorf("buffer_period: cannot be negative")
	}

	if *c.Max < *c.Min {
		return fmt.Errorf("buffer_period: min must be less than max")
	}

	return nil
}

// GoString defines the printable version of this struct.
func (c *BufferPeriodConfig) GoString() string {
	if c == nil {
		return "(*BufferPeriodConfig)(nil)"
	}

	return fmt.Sprintf("&BufferPeriodConfig{"+
		"Enabled:%v, "+
		"Min:%s, "+
		"Max:%s"+
		"}",
		BoolVal(c.Enabled),
		TimeDurationVal(c.Min),
		TimeDurationVal(c.Max),
	)
}
