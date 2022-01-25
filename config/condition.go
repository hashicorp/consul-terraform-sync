package config

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/internal/decode"
	"github.com/mitchellh/mapstructure"
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

// isConditionNil can be used to check if a ConditionConfig interface is nil by
// checking both the type and value. Not needed for checking a ConditionConfig
// implementation i.e. isConditionNil(ConditionConfig),
// servicesConditionConfig == nil
func isConditionNil(c ConditionConfig) bool {
	var result bool
	// switching on type is a performance enhancement
	switch v := c.(type) {
	case *ServicesConditionConfig:
		result = v == nil
	case *CatalogServicesConditionConfig:
		result = v == nil
	default:
		return c == nil || reflect.ValueOf(c).IsNil()
	}
	return c == nil || result
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

		if c, ok := conditions[catalogServicesConditionType]; ok {
			var config CatalogServicesConditionConfig
			return decodeConditionToType(c, &config)
		}
		if c, ok := conditions[servicesConditionType]; ok {
			var config ServicesConditionConfig
			return decodeConditionToType(c, &config)
		}

		return nil, fmt.Errorf("unsupported condition type: %v", data)
	}
}

// decodeConditionToType is used by the overall config mapstructure decode hook
// conditionToTypeFunc in order to convert ConditionConfig in the form
// of an interface into an implementation
func decodeConditionToType(data interface{}, condition ConditionConfig) (ConditionConfig, error) {
	var md mapstructure.Metadata
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			decode.HookWeakDecodeFromSlice,
		),
		WeaklyTypedInput: true,
		ErrorUnused:      false,
		Metadata:         &md,
		Result:           &condition,
	})
	if err != nil {
		log.Printf("[ERR] (config) condition mapstructure decoder creation"+
			"failed: %s", err)
		return nil, err
	}

	if err := decoder.Decode(data); err != nil {
		log.Printf("[ERR] (config) condition mapstructure decode failed: %s",
			err)
		return nil, err
	}

	if len(md.Unused) > 0 {
		sort.Strings(md.Unused)
		err := fmt.Errorf("invalid keys: %s", strings.Join(md.Unused, ", "))
		log.Printf("[ERR] (config) condition %s", err.Error())
		return nil, err
	}

	return condition, nil
}
