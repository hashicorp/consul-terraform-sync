package config

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/internal/decode"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/mitchellh/mapstructure"
)

// ConditionConfig configures a condition on a task. This Condition defines what to monitor for in order to
// trigger the execution of a task
type ConditionConfig interface {
	MonitorConfig
}

// EmptyConditionConfig sets an unconfigured condition with a non-null value
func EmptyConditionConfig() ConditionConfig {
	return &NoConditionConfig{}
}

// conditionToTypeFunc is a decode hook function to decode a ConditionConfig
// into a specific condition implementation structures. Used when decoding
// cts config overall.
func conditionToTypeFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		// identify if parsing a ConditionConfig
		var i ConditionConfig
		if t != reflect.TypeOf(&i).Elem() {
			return data, nil
		}

		// abstract conditions map out depending on hcl vs. json formatting
		// data hcl ex: [map[catalog-services:[map[regexp:.*]]]]
		// data json ex: map[catalog-services:map[regexp:.*]]
		var conditions map[string]interface{}
		if hcl, ok := data.([]map[string]interface{}); ok {
			if len(hcl) != 1 {
				return nil, fmt.Errorf("expected only one item in hcl "+
					"condition but got %d: %v", len(hcl), data)
			}
			conditions = hcl[0]
		}
		if json, ok := data.(map[string]interface{}); ok {
			conditions = json
		}

		if c, ok := conditions[catalogServicesType]; ok {
			var config CatalogServicesConditionConfig
			return decodeConditionToType(c, &config)
		}
		if c, ok := conditions[servicesType]; ok {
			var config ServicesConditionConfig
			return decodeConditionToType(c, &config)
		}
		if c, ok := conditions[consulKVType]; ok {
			var config ConsulKVConditionConfig
			return decodeConditionToType(c, &config)
		}
		if c, ok := conditions[scheduleType]; ok {
			var config ScheduleConditionConfig
			return decodeConditionToType(c, &config)
		}

		return nil, fmt.Errorf("unsupported condition type: %v", data)
	}
}

// decodeConditionToType is used by the overall config mapstructure decode hook
// ToTypeFunc in order to convert MonitorConfig in the form
// of an interface into an implementation
func decodeConditionToType(data interface{}, monitor MonitorConfig) (MonitorConfig, error) {
	var md mapstructure.Metadata
	logger := logging.Global().Named(logSystemName)
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			decode.HookWeakDecodeFromSlice,
		),
		WeaklyTypedInput: true,
		ErrorUnused:      false,
		Metadata:         &md,
		Result:           &monitor,
	})
	if err != nil {
		logger.Error("monitor mapstructure decoder create failed", "error", err)
		return nil, err
	}

	if err := decoder.Decode(data); err != nil {
		logger.Error("monitor mapstructure decode failed", "error", err)
		return nil, err
	}

	if len(md.Unused) > 0 {
		sort.Strings(md.Unused)
		err := fmt.Errorf("invalid keys: %s", strings.Join(md.Unused, ", "))
		logger.Error("monitor invalid keys", "error", err)
		return nil, err
	}

	return monitor, nil
}

// isConditionNil returns true if the condition is Nil and false otherwise
func isConditionNil(c ConditionConfig) bool {
	return isMonitorNil(c)
}

// bothConditionInputConfigLogMsg is the log message to warn when the user
// configures both `source_includes_var` and `use_as_module_input` fields.
//
// expected value for '%s' is the condition type
const bothConditionInputConfigLogMsg = `%s condition block is configured ` +
	`with both the 'source_includes_var' and the 'use_as_module_input' field. ` +
	`Defaulting to the 'use_as_module_input' value`

// sourceIncludesVarLogMsg is the log message for deprecating the
// `source_includes_var` field.
//
// expected value for both '%s' is the condition type
const sourceIncludesVarLogMsg = `the 'source_includes_var' field in ` +
	`the task's %s condition block is deprecated in v0.5.0 and will be removed in v0.8.0.

Please replace 'source_includes_var' with 'use_as_module_input' in your condition configuration.

We will be releasing a tool to help upgrade your configuration for this deprecation.

Example upgrade:
|    task {
|      condition "%s" {
|  -     source_includes_var = false
|  +     use_as_module_input = false
|      }
|      ...
|    }

For more details and examples, please see:
https://consul.io/docs/nia/release-notes/0-5-0#deprecate-source_includes_var-field
`
