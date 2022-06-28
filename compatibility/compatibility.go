package compatibility

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
)

const (
	fieldIgnoreKey = "cts-ha"
	fieldIgnoreTag = "unshared"
)

// IsCompatibleConfig is a convenience method for calling IsCompatible with
// config.Config configurations
func IsCompatibleConfig(ctx context.Context, baseConf *config.Config, conf *config.Config) bool {
	return IsCompatible(ctx, reflect.ValueOf(baseConf), reflect.ValueOf(conf))
}

// IsCompatible compares a pointer to a base configuration with a pointer to a configuration and returns true
// if the two configurations are compatible with each other, and false otherwise
func IsCompatible(ctx context.Context, baseConf reflect.Value, conf reflect.Value) bool {
	// Make sure we are comparing the same type and that we are only comparing pointers
	if baseConf.Type() != conf.Type() {
		panic("types being compared are not the same")
	}

	// Only check if non-nil pointers, otherwise this is not being
	// used as intended
	if baseConf.Kind() != reflect.Pointer || baseConf.IsNil() {
		panic("cannot check compatibility of non pointers")
	}

	rc := compatabilityChecker{
		logger:         logging.FromContext(ctx),
		fieldIgnoreKey: fieldIgnoreKey,
		fieldIgnoreTag: fieldIgnoreTag,
	}
	return rc.check(baseConf, conf, "")
}

type compatabilityChecker struct {
	logger         logging.Logger
	fieldIgnoreKey string
	fieldIgnoreTag string
}

func (rc compatabilityChecker) check(valueBase reflect.Value, value reflect.Value, fieldName string) bool {
	// Only check into non-nil pointers
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return true
	}

	// Dereference the pointer to the struct
	structValueBase := valueBase.Elem()
	structValue := value.Elem()
	structType := value.Type().Elem()

	initName := fieldName
	isCompatible := true

	// Iterate through fields on the struct type
	for i, f := range reflect.VisibleFields(structType) {
		// Get the field on the struct value
		vf := structValue.Field(i)
		vb := structValueBase.Field(i)

		// Ignore fields if they contain the ignore key/tag and return true
		tag, ok := f.Tag.Lookup(rc.fieldIgnoreKey)
		if ok && tag == rc.fieldIgnoreTag {
			continue
		}

		// Concatenate field-names if necessary to give a
		// full path in context of the struct
		if fieldName != "" {
			fieldName = fmt.Sprintf("%s.%s", fieldName, f.Name)
		} else {
			fieldName = f.Name
		}

		// If struct is shared check, do not check the root struct
		if (f.Type.Kind() == reflect.Pointer && f.Type.Elem().Kind() == reflect.Struct) || f.Type.Kind() == reflect.Struct {
			rc.check(vb, vf, fieldName)
			fieldName = initName
			continue
		}

		if vb.Interface() != vf.Interface() {
			// Set isCompatible to false only on first incompatibility
			// once incompatible once, it will always be incompatible
			if isCompatible {
				isCompatible = false
			}
			rc.logger.Error(fmt.Sprintf("Config %v does not equal %v for %v",
				getPrintableValue(vb), getPrintableValue(vf), fieldName))
		}

		// Reset field name so that we do not concatenate unnecessarily
		// in the future
		fieldName = initName
	}

	return isCompatible
}

func getPrintableValue(v reflect.Value) interface{} {
	var vp interface{}
	if (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) && v.IsNil() {
		vp = "<nil>"
	} else {
		vp = reflect.Indirect(v)
	}

	return vp
}
