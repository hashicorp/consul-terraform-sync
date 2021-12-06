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

// EmptySourceInputConfig sets un-configured source inputs with a non-null
// value
func EmptySourceInputConfig() SourceInputConfig {
	return &NoSourceInputConfig{}
}

// isSourceInputEmpty returns true if the provided SourceInputConfig `c` is
// of type NoSourceInputConfig
func isSourceInputEmpty(c SourceInputConfig) bool {
	_, ok := c.(*NoSourceInputConfig)
	return ok
}

// sourceInputToTypeFunc is a decode hook function to decode a SourceInputConfig
// into a specific source var implementation structures. Used when decoding
// cts config overall.
func sourceInputToTypeFunc() mapstructure.DecodeHookFunc {
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
			return decodeSourceInputToType(c, &config)
		}

		if c, ok := sourceInputs[consulKVType]; ok {
			var config ConsulKVSourceInputConfig
			return decodeSourceInputToType(c, &config)
		}

		return nil, fmt.Errorf("unsupported source_input type: %v", data)
	}
}

// decodeSourceInputToType is used by the overall config mapstructure decode hook
// SourceInputToTypeFunc in order to convert SourceInputConfig in the form
// of an interface into an implementation
func decodeSourceInputToType(data interface{}, sourceInput SourceInputConfig) (SourceInputConfig, error) {
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
		logger.Error("source_input mapstructure decoder create failed", "error", err)
		return nil, err
	}

	if err := decoder.Decode(data); err != nil {
		logger.Error("source_input mapstructure decode failed", "error", err)
		return nil, err
	}

	if len(md.Unused) > 0 {
		sort.Strings(md.Unused)
		err := fmt.Errorf("invalid keys: %s", strings.Join(md.Unused, ", "))
		logger.Error("source_input invalid keys", "error", err)
		return nil, err
	}

	return sourceInput, nil
}

// isSourceInputNil returns true if the condition is Nil and false otherwise
func isSourceInputNil(si SourceInputConfig) bool {
	return isMonitorNil(si)
}
