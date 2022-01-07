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

// SourceInputConfig configures a source_input on a task. This Source Input defines which Consul objects to monitor
// (e.g. services, kv) whose values are then provided as the task source’s input variables
type SourceInputConfig interface {
	MonitorConfig
}

// EmptyModuleInputConfig sets un-configured module inputs with a non-null
// value
func EmptyModuleInputConfig() SourceInputConfig {
	return &NoMonitorConfig{}
}

// isModuleInputEmpty returns true if the provided SourceInputConfig `c` is
// of type NoMonitorConfig
func isModuleInputEmpty(c SourceInputConfig) bool {
	_, ok := c.(*NoMonitorConfig)
	return ok
}

// moduleInputToTypeFunc is a decode hook function to decode a SourceInputConfig
// into a specific module var implementation structures. Used when decoding
// cts config overall.
func moduleInputToTypeFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		// identify if parsing a SourceInputConfig
		var i SourceInputConfig
		if t != reflect.TypeOf(&i).Elem() {
			return data, nil
		}

		// abstract sourceInputs map out depending on hcl vs. json formatting
		// data hcl ex: [map[services:[map[regexp:.*]]]]
		// data json ex: map[services:map[regexp:.*]]
		var sourceInputs map[string]interface{}
		if hcl, ok := data.([]map[string]interface{}); ok {
			if len(hcl) != 1 {
				return nil, fmt.Errorf("expected only one item in hcl "+
					"sourceInput but got %d: %v", len(hcl), data)
			}
			sourceInputs = hcl[0]
		}
		if json, ok := data.(map[string]interface{}); ok {
			sourceInputs = json
		}

		if c, ok := sourceInputs[servicesType]; ok {
			var config ServicesSourceInputConfig
			return decodeModuleInputToType(c, &config)
		}

		if c, ok := sourceInputs[consulKVType]; ok {
			var config ConsulKVSourceInputConfig
			return decodeModuleInputToType(c, &config)
		}

		return nil, fmt.Errorf("unsupported module_input type: %v", data)
	}
}

// decodeModuleInputToType is used by the overall config mapstructure decode hook
// ModuleInputToTypeFunc in order to convert SourceInputConfig in the form
// of an interface into an implementation
func decodeModuleInputToType(data interface{}, sourceInput SourceInputConfig) (SourceInputConfig, error) {
	var md mapstructure.Metadata
	logger := logging.Global().Named(logSystemName)
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			decode.HookWeakDecodeFromSlice,
		),
		WeaklyTypedInput: true,
		ErrorUnused:      false,
		Metadata:         &md,
		Result:           &sourceInput,
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

	return sourceInput, nil
}

// isModuleInputNil returns true if the module input is nil and false otherwise
func isModuleInputNil(si SourceInputConfig) bool {
	return isMonitorNil(si)
}
