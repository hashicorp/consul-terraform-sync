package config

import (
	"reflect"
)

// MonitorConfig represents the base object for objects like source_input and condition, both of which
// "monitor" some entity or key-store in order to perform some action
type MonitorConfig interface {
	Copy() MonitorConfig
	Merge(MonitorConfig) MonitorConfig
	Finalize()
	Validate() error
	GoString() string
}

// isMonitorNil can be used to check if a MonitorConfig interface is nil by
// checking both the type and value. Not needed for checking a MonitorConfig
// implementation i.e. isMonitorNil(MonitorConfig),
// ServicesMonitorConfig == nil
func isMonitorNil(c MonitorConfig) bool {
	var result bool
	// switching on type is a performance enhancement
	switch v := c.(type) {
	case *ServicesConditionConfig:
		result = v == nil
	case *CatalogServicesConditionConfig:
		result = v == nil
	case *ConsulKVConditionConfig:
		result = v == nil
	case *ScheduleConditionConfig:
		result = v == nil
	case *ServicesSourceInputConfig:
		result = v == nil
	default:
		return c == nil || reflect.ValueOf(c).IsNil()
	}
	return c == nil || result
}
