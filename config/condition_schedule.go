package config

import (
	"fmt"

	"github.com/hashicorp/cronexpr"
)

const scheduleType = "schedule"

var _ ConditionConfig = (*ScheduleConditionConfig)(nil)

// ScheduleConditionConfig configures a condition configuration block of type
// 'schedule'. A schedule condition is triggered by a configured cron schedule
type ScheduleConditionConfig struct {
	Cron *string `mapstructure:"cron"`
}

func (c *ScheduleConditionConfig) VariableType() string {
	return ""
}

// Copy returns a deep copy of this configuration.
func (c *ScheduleConditionConfig) Copy() MonitorConfig {
	if c == nil {
		return nil
	}

	var o ScheduleConditionConfig
	o.Cron = StringCopy(c.Cron)

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
func (c *ScheduleConditionConfig) Merge(o MonitorConfig) MonitorConfig {
	if c == nil {
		if isConditionNil(o) { // o is interface, use isConditionNil()
			return nil
		}
		return o.Copy()
	}

	if isConditionNil(o) {
		return c.Copy()
	}

	r := c.Copy()
	o2, ok := o.(*ScheduleConditionConfig)
	if !ok {
		return r
	}

	r2 := r.(*ScheduleConditionConfig)

	if o2.Cron != nil {
		r2.Cron = StringCopy(o2.Cron)
	}

	return r2
}

// Finalize ensures there no nil pointers.
func (c *ScheduleConditionConfig) Finalize() {
	if c == nil { // config not required, return early
		return
	}

	if c.Cron == nil {
		c.Cron = String("")
	}
}

// Validate validates the values and required options. This method is recommended
// to run after Finalize() to ensure the configuration is safe to proceed.
func (c *ScheduleConditionConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}

	if c.Cron == nil || len(*c.Cron) == 0 {
		return fmt.Errorf("cron config is required for schedule condition")
	}

	if _, err := cronexpr.Parse(*c.Cron); err != nil {
		return fmt.Errorf("unable to parse schedule condition's cron config "+
			"%q: %s. for more information on writing cron expressions, see %s",
			StringVal(c.Cron), err, "https://github.com/hashicorp/cronexpr")
	}

	return nil
}

// GoString defines the printable version of this struct.
func (c *ScheduleConditionConfig) GoString() string {
	if c == nil {
		return "(*ScheduleConditionConfig)(nil)"
	}

	return fmt.Sprintf("&ScheduleConditionConfig{"+
		"Cron:%s, "+
		"}",
		StringVal(c.Cron),
	)
}
