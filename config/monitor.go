package config

import (
	"reflect"
)

// MonitorConfig represents the base object for objects like monitor_input and
// condition, both of which "monitor" an object in order to perform some action
type MonitorConfig interface {
	Copy() MonitorConfig
	Merge(MonitorConfig) MonitorConfig
	Finalize()
	Validate() error
	GoString() string

	// VariableType returns type of variable that a module_input or a condition
	// block monitors. For example, `condition "services"` monitors the variable
	// type "services.
	//
	// Used to ensure requirement that configured monitored variable types are
	// unique for a given task. Must be unique across module_input, condition,
	// and services.
	VariableType() string
}

// isMonitorNil can be used to check if a MonitorConfig interface is nil by
// checking both the type and value. Not needed for checking a MonitorConfig
// implementation i.e. isMonitorNil(MonitorConfig),
// ServicesMonitorConfig == nil
func isMonitorNil(c MonitorConfig) bool {
	var result bool
	// switching on type is a performance enhancement
	switch v := c.(type) {
	// Conditions
	case *ServicesConditionConfig:
		result = v == nil
	case *CatalogServicesConditionConfig:
		result = v == nil
	case *ConsulKVConditionConfig:
		result = v == nil
	case *ScheduleConditionConfig:
		result = v == nil

	// Module Inputs
	case *ServicesModuleInputConfig:
		result = v == nil
	case *ConsulKVModuleInputConfig:
		result = v == nil
	default:
		return c == nil || reflect.ValueOf(c).IsNil()
	}
	return c == nil || result
}
