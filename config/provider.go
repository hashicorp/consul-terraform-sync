// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"fmt"
	"strings"
)

// TerraformProviderConfigs is an array of configuration for each provider.
type TerraformProviderConfigs []*TerraformProviderConfig

// TerraformProviderConfig is a map representing the configuration for a single
// provider where the key is the name of provider and value is the configuration.
type TerraformProviderConfig map[string]interface{}

// DefaultTerraformProviderConfigs returns a configuration that is populated
// with the default values.
func DefaultTerraformProviderConfigs() *TerraformProviderConfigs {
	return &TerraformProviderConfigs{}
}

// Len is a helper method to get the length of the underlying config list
func (c *TerraformProviderConfigs) Len() int {
	if c == nil {
		return 0
	}

	return len(*c)
}

// Copy returns a deep copy of this configuration.
func (c *TerraformProviderConfigs) Copy() *TerraformProviderConfigs {
	if c == nil {
		return nil
	}

	o := make(TerraformProviderConfigs, c.Len())
	for i, t := range *c {
		tmp := make(TerraformProviderConfig)
		for k, v := range *t {
			tmp[k] = v
		}
		o[i] = &tmp
	}
	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *TerraformProviderConfigs) Merge(o *TerraformProviderConfigs) *TerraformProviderConfigs {
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
func (c *TerraformProviderConfigs) Finalize() {
	if c == nil {
		*c = *DefaultTerraformProviderConfigs()
	}
}

// Validate validates the values and nested values of the configuration struct
func (c *TerraformProviderConfigs) Validate() error {
	if c == nil {
		// Uninitialized config is invalid. Although unlikely, no providers could
		// still be valid if all of the tasks happen to not depend on a provider.
		return fmt.Errorf("missing provider configuration")
	}

	m := make(map[string]bool)
	for _, s := range *c {
		if err := s.Validate(); err != nil {
			return err
		}

		// Require providers to be unique by name and alias.
		id := s.id()
		if ok := m[id]; ok {
			return fmt.Errorf("duplicate provider configuration: %s", id)
		}
		m[id] = true
	}

	return nil
}

// GoString defines the printable version of this struct. Provider configuration
// is completely redacted since providers will have varying arguments containing
// secrets
func (c *TerraformProviderConfigs) GoString() string {
	if c == nil {
		return "(*TerraformProviderConfigs)(nil)"
	}

	s := make([]string, len(*c))
	for i, provider := range *c {
		for name := range *provider {
			s[i] = fmt.Sprintf("&map[%s:%s]", name, redactMessage)
		}
	}

	return "{" + strings.Join(s, ", ") + "}"
}

// Validate validates the values and nested values of the configuration struct.
func (c *TerraformProviderConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("invalid provider configuration")
	}

	numLabels := len(*c)
	if numLabels == 0 {
		return fmt.Errorf("missing provider name for the terraform_provider block")
	} else if numLabels > 1 {
		labels := make([]string, 0, numLabels)
		for l := range *c {
			labels = append(labels, l)
		}
		return fmt.Errorf("unexpected terraform_provider block labels: %s", strings.Join(labels, ","))
	}

	for label := range *c {
		// Validate block format
		rawBlock := (*c)[label]
		block, ok := rawBlock.(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected terraform_provider block format")
		}

		// Validate task_env format if exists
		taskEnv, exists := block["task_env"]
		if exists {
			taskEnvMap, ok := taskEnv.(map[string]interface{})
			if !ok {
				return fmt.Errorf("unexpected task_env block format: task_env should be a map of strings")
			}
			for k, v := range taskEnvMap {
				if _, ok := v.(string); !ok {
					return fmt.Errorf("unexpected task_env block format: value for %q should be a string", k)
				}
			}
		}
	}

	return nil
}

// id returns the unique name to represent the provider configuration. If alias is set,
// the ID is <name>.<alias>. Otherwise, the name is used as the ID.
func (c *TerraformProviderConfig) id() string {
	if c == nil || len(*c) == 0 {
		return ""
	}

	var name string
	var rawConf interface{}
	for k, v := range *c {
		name = k
		rawConf = v
	}
	pConf, ok := rawConf.(map[string]interface{})
	if !ok {
		return name
	}

	alias, ok := pConf["alias"].(string)
	if !ok {
		return name
	}

	return fmt.Sprintf("%s.%s", name, alias)
}
