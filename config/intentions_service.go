package config

import (
	"fmt"
	"regexp"
)

// IntentionsServicesConfig configures regexp/list of names of services.
type IntentionsServicesConfig struct {
	// Regexp configures the source services to monitor by matching on the service name.
	// Either Regexp or Names must be configured, not both.
	Regexp *string `mapstructure:"regexp"`

	// Names configures the services to monitor by listing the service name.
	// Either Regexp or Names must be configured, not both.
	Names []string `mapstructure:"names"`
}

// Copy returns a deep copy of IntentionsServicesConfig
func (c *IntentionsServicesConfig) Copy() *IntentionsServicesConfig {
	if c == nil {
		return nil
	}

	var o IntentionsServicesConfig
	o.Regexp = StringCopy(c.Regexp)

	if c.Names != nil {
		o.Names = make([]string, 0, len(c.Names))
		o.Names = append(o.Names, c.Names...)
	}
	return &o
}

// Merge combines all values in this IntentionsServicesConfig with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *IntentionsServicesConfig) Merge(o *IntentionsServicesConfig) *IntentionsServicesConfig {
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

	if o.Regexp != nil {
		r.Regexp = StringCopy(o.Regexp)
	}

	r.Names = mergeSlices(c.Names, o.Names)

	return r
}

// Finalize ensures there no nil pointers.
// Exception: `Regexp` is never finalized and can potentially be nil.
func (c *IntentionsServicesConfig) Finalize() {
	if c == nil { // config not required, return early
		return
	}

	if c.Names == nil {
		c.Names = []string{}
	}
}

func (c *IntentionsServicesConfig) Validate() error {
	if c == nil { // config not required, return early
		return nil
	}

	// Check that either regex or names is configured but not both
	namesConfigured := c.Names != nil && len(c.Names) > 0
	regexConfigured := c.Regexp != nil

	if namesConfigured && regexConfigured {
		return fmt.Errorf("regexp and names fields cannot both be " +
			"configured. If both are needed, consider including the list of " +
			"names as part of the regex or creating separate tasks")
	}

	if !namesConfigured && !regexConfigured {
		return fmt.Errorf("either the regexp or names field must be configured")
	}

	// Validate regex
	if regexConfigured {
		if _, err := regexp.Compile(StringVal(c.Regexp)); err != nil {
			return fmt.Errorf("unable to compile intentions service regexp: %s", err)
		}
	}

	// Check that names does not contain empty strings
	if namesConfigured {
		for _, name := range c.Names {
			if name == "" {
				return fmt.Errorf("names field includes empty string(s). " +
					"intentions service names cannot be empty")
			}
		}
	}

	return nil
}

func (c *IntentionsServicesConfig) GoString() string {
	if c == nil {
		return ": (*IntentionsServicesConfig)(nil)"
	}

	if len(c.Names) > 0 {
		return fmt.Sprintf("Names:%s",c.Names,)
	} else {
		return fmt.Sprintf("Regexp:%s",StringVal(c.Regexp),)
	}
}
