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

// ModuleInputConfig configures a module_input on a task. The module input
// defines the Consul object(s) to monitor (e.g. services, kv). The object
// values as passed to the task module's input variable
type ModuleInputConfig interface {
	MonitorConfig
}

// ModuleInputConfigs is a collection of ModuleInputConfig
type ModuleInputConfigs []ModuleInputConfig

func DefaultModuleInputConfigs() *ModuleInputConfigs {
	return &ModuleInputConfigs{}
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

// Len is a helper method to get the length of the underlying config list
func (c *ModuleInputConfigs) Len() int {
	if c == nil {
		return 0
	}

	return len(*c)
}

// Copy returns a deep copy of this configuration.
func (c *ModuleInputConfigs) Copy() *ModuleInputConfigs {
	if c == nil {
		return nil
	}

	o := make(ModuleInputConfigs, c.Len())
	for i, t := range *c {
		o[i] = t.Copy()
	}
	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *ModuleInputConfigs) Merge(o *ModuleInputConfigs) *ModuleInputConfigs {
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

	*r = append(*r, *o...)

	return r
}

// Finalize ensures the configuration has no nil pointers and sets default
// values.
func (c *ModuleInputConfigs) Finalize() {
	if c == nil {
		return
	}

	for _, t := range *c {
		t.Finalize()
	}
}

// Validate validates the values and nested values of the configuration struct.
// It validates a task's module inputs while taking into account the task's
// services and condition
func (c *ModuleInputConfigs) Validate(services []string, condition ConditionConfig) error {
	if c == nil || c.Len() == 0 {
		// config is not required, return early
		return nil
	}

	logger := logging.Global().Named(logSystemName).Named(taskSubsystemName)

	// Confirm module_inputs's type is unique across module_inputs
	varTypes := make(map[string]bool)
	for _, input := range *c {
		varType := input.VariableType()
		if ok := varTypes[varType]; ok {
			return fmt.Errorf("more than one 'module_input' block for the %q "+
				"variable. variable types must be unique", varType)
		}
		varTypes[varType] = true
	}

	// Confirm module_input types are different from task.services variable type
	if len(services) > 0 {

		// ServicesModuleInput is the only module_input with the same variable
		// type as task.services
		servicesType := &ServicesModuleInputConfig{}

		if ok := varTypes[servicesType.VariableType()]; ok {
			err := fmt.Errorf("task's `services` field and `module_input "+
				"'services'` block both monitor %q variable type. only one of "+
				"these can be configured per task", servicesType.VariableType())
			logger.Error("list of `services` and `module_input 'services'` "+
				"block cannot both be configured. Consider combining the list "+
				"into the module_input block or creating separate tasks",
				"error", err)
			return err
		}
	}

	// Confirm module_input's type is different from condition
	if condition == nil {
		return nil
	}
	if ok := varTypes[condition.VariableType()]; ok {
		err := fmt.Errorf("task's condition block and module_input block "+
			"both monitor %q variable type. condition and module_input "+
			"variable type must be unique", condition.VariableType())
		logger.Error("condition and module_input block cannot monitor same "+
			"variable type. If both are needed, consider combining the "+
			"module_input with the condition block or creating separate tasks",
			"error", err)
		return err
	}

	return nil
}

// GoString defines the printable version of this struct.
func (c *ModuleInputConfigs) GoString() string {
	if c == nil {
		return "(*ModuleInputConfigs)(nil)"
	}

	s := make([]string, len(*c))
	for i, t := range *c {
		s[i] = t.GoString()
	}

	return "{" + strings.Join(s, ", ") + "}"
}
