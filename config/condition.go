package config

import (
	"reflect"
)

// ConditionConfig configures a condition on a task to define the condition on
// which to execute a task.
type ConditionConfig interface {
	Copy() ConditionConfig
	Merge(ConditionConfig) ConditionConfig
	Finalize([]string)
	Validate() error
	GoString() string
}

// DefaultConditionConfig returns the default conditions which is an unconfigured
// 'services' type condition.
func DefaultConditionConfig() ConditionConfig {
	return &ServicesConditionConfig{}
}

// isNil can be used to check if a ConditionConfig interface is nil by checking
// both the type and value. Not needed for checking a ConditionConfig
// implementation i.e. isNil(ConditionConfig), servicesConditionConfig == nil
func isNil(c ConditionConfig) bool {
	return c == nil || reflect.ValueOf(c).IsNil()
}
