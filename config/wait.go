package config

import (
	"fmt"
	"time"
)

var (
	DefaultWaitMin = time.Duration(5 * time.Second)
	DefaultWaitMax = time.Duration(4 * DefaultWaitMin)
)

// WaitConfig is the Min/Max duration used by the Watcher
type WaitConfig struct {
	// Enabled determines if this wait is enabled.
	Enabled *bool `mapstructure:"enabled"`

	// Min and Max are the minimum and maximum time, respectively, to wait for
	// data changes before rendering a new template to disk.
	Min *time.Duration `mapstructure:"min"`
	Max *time.Duration `mapstructure:"max"`
}

// DefaultWaitConfig is the global default configuration for all tasks.
func DefaultWaitConfig() *WaitConfig {
	return &WaitConfig{
		Enabled: Bool(true),
		Min:     &DefaultWaitMin,
		Max:     &DefaultWaitMax,
	}
}

// DefaultTaskWaitConfig is the default configuration for a task.
func DefaultTaskWaitConfig() *WaitConfig {
	return &WaitConfig{
		Enabled: Bool(false),
		Min:     &DefaultWaitMin,
		Max:     &DefaultWaitMax,
	}
}

// Copy returns a deep copy of this configuration.
func (c *WaitConfig) Copy() *WaitConfig {
	if c == nil {
		return nil
	}

	var o WaitConfig
	o.Enabled = BoolCopy(c.Enabled)
	o.Min = TimeDurationCopy(c.Min)
	o.Max = TimeDurationCopy(c.Max)
	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *WaitConfig) Merge(o *WaitConfig) *WaitConfig {
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

// Finalize ensures there no nil pointers.
func (c *WaitConfig) Finalize() {
	if c.Enabled == nil {
		c.Enabled = Bool(TimeDurationPresent(c.Min))
	}

	if c.Min == nil {
		c.Min = TimeDuration(DefaultWaitMin)
	}

	if c.Max == nil {
		c.Max = TimeDuration(4 * *c.Min)
	}
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *WaitConfig) Validate() error {
	if c == nil {
		// Wait config is not required, return early
		return nil
	}

	if !BoolVal(c.Enabled) {
		return nil
	}

	if c.Min.Seconds() < 0 || c.Max.Seconds() < 0 {
		return fmt.Errorf("wait: cannot be negative")
	}

	if *c.Max < *c.Min {
		return fmt.Errorf("wait: min must be less than max")
	}

	return nil
}

// GoString defines the printable version of this struct.
func (c *WaitConfig) GoString() string {
	if c == nil {
		return "(*WaitConfig)(nil)"
	}

	return fmt.Sprintf("&WaitConfig{"+
		"Enabled:%v, "+
		"Min:%s, "+
		"Max:%s"+
		"}",
		BoolVal(c.Enabled),
		TimeDurationVal(c.Min),
		TimeDurationVal(c.Max),
	)
}
