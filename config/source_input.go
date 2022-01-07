package config

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul/lib/decode"
	"github.com/mitchellh/mapstructure"
)

// ModuleInputConfig configures a module_input on a task. The module input
// defines the Consul object(s) to monitor (e.g. services, kv). The object
// values as passed to the task module's input variable
type ModuleInputConfig interface {
	MonitorConfig
}

// EmptyModuleInputConfig sets un-configured module inputs with a non-null
// value
func EmptyModuleInputConfig() ModuleInputConfig {
	return &NoMonitorConfig{}
}

// isModuleInputEmpty returns true if the provided ModuleInputConfig `c` is
// of type NoMonitorConfig
func isModuleInputEmpty(c ModuleInputConfig) bool {
	_, ok := c.(*NoMonitorConfig)
	return ok
}

// moduleInputToTypeFunc is a decode hook function to decode a ModuleInputConfig
// into a specific module var implementation structures. Used when decoding
// cts config overall.
func moduleInputToTypeFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		// identify if parsing a ModuleInputConfig
		var i ModuleInputConfig
		if t != reflect.TypeOf(&i).Elem() {
			return data, nil
		}

		// abstract moduleInputs map out depending on hcl vs. json formatting
		// data hcl ex: [map[services:[map[regexp:.*]]]]
		// data json ex: map[services:map[regexp:.*]]
		var moduleInputs map[string]interface{}
		if hcl, ok := data.([]map[string]interface{}); ok {
			if len(hcl) != 1 {
				return nil, fmt.Errorf("expected only one item in hcl "+
					"module_input but got %d: %v", len(hcl), data)
			}
			moduleInputs = hcl[0]
		}
		if json, ok := data.(map[string]interface{}); ok {
			moduleInputs = json
		}

		if c, ok := moduleInputs[servicesType]; ok {
			var config ServicesModuleInputConfig
			return decodeModuleInputToType(c, &config)
		}

		if c, ok := moduleInputs[consulKVType]; ok {
			var config ConsulKVModuleInputConfig
			return decodeModuleInputToType(c, &config)
		}

		return nil, fmt.Errorf("unsupported module_input type: %v", data)
	}
}

// decodeModuleInputToType is used by the overall config mapstructure decode hook
// ModuleInputToTypeFunc in order to convert ModuleInputConfig in the form
// of an interface into an implementation
func decodeModuleInputToType(data interface{}, moduleInput ModuleInputConfig) (ModuleInputConfig, error) {
	var md mapstructure.Metadata
	logger := logging.Global().Named(logSystemName)
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			decode.HookWeakDecodeFromSlice,
		),
		WeaklyTypedInput: true,
		ErrorUnused:      false,
		Metadata:         &md,
		Result:           &moduleInput,
	})
	if err != nil {
		logger.Error("module_input mapstructure decoder create failed", "error", err)
		return nil, err
	}

	if err := decoder.Decode(data); err != nil {
		logger.Error("module_input mapstructure decode failed", "error", err)
		return nil, err
	}

	if len(md.Unused) > 0 {
		sort.Strings(md.Unused)
		err := fmt.Errorf("invalid keys: %s", strings.Join(md.Unused, ", "))
		logger.Error("module_input invalid keys", "error", err)
		return nil, err
	}

	return moduleInput, nil
}

// isModuleInputNil returns true if the module input is nil and false otherwise
func isModuleInputNil(si ModuleInputConfig) bool {
	return isMonitorNil(si)
}
