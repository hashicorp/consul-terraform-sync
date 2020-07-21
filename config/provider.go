package config

import (
	"fmt"
	"strings"
)

type ProviderConfigs []*ProviderConfig

type ProviderConfig map[string]interface{}

// DefaultProviderConfigs returns a configuration that is populated with the
// default values.
func DefaultProviderConfigs() *ProviderConfigs {
	return &ProviderConfigs{}
}

// Len is a helper method to get the length of the underlying config list
func (c *ProviderConfigs) Len() int {
	if c == nil {
		return 0
	}

	return len(*c)
}

// Copy returns a deep copy of this configuration.
func (c *ProviderConfigs) Copy() *ProviderConfigs {
	if c == nil {
		return nil
	}

	o := make(ProviderConfigs, c.Len())
	for i, t := range *c {
		copy := make(ProviderConfig)
		for k, v := range *t {
			copy[k] = v
		}
		o[i] = &copy
	}
	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *ProviderConfigs) Merge(o *ProviderConfigs) *ProviderConfigs {
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
func (c *ProviderConfigs) Finalize() {
	if c == nil {
		*c = *DefaultProviderConfigs()
	}
}

// Validate validates the values and nested values of the configuration struct
func (c *ProviderConfigs) Validate() error {
	if c == nil {
		// Uninitialized config is invalid. Although unlikely, no providers could
		// still be valid if all of the tasks happen to not depend on a provider.
		return fmt.Errorf("missing provider configuration")
	}

	// TODO validate uniqueness by alias
	for _, s := range *c {
		if err := s.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// GoString defines the printable version of this struct.
func (c *ProviderConfigs) GoString() string {
	if c == nil {
		return "(*ProviderConfigs)(nil)"
	}

	s := make([]string, len(*c))
	for i, t := range *c {
		s[i] = fmt.Sprint(t)
	}

	return "{" + strings.Join(s, ", ") + "}"
}

func (c *ProviderConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("invalid provider configuration")
	}

	numLabels := len(*c)
	if numLabels == 0 {
		return fmt.Errorf("missing provider name for the provider block")
	} else if numLabels > 1 {
		labels := make([]string, 0, numLabels)
		for l := range *c {
			labels = append(labels, l)
		}
		return fmt.Errorf("unexpected provider block labels: %s", strings.Join(labels, ","))
	}

	return nil
}
