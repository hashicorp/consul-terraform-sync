package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServicesConditionConfig_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ServicesConditionConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&ServicesConditionConfig{},
		},
		{
			"happy_path",
			&ServicesConditionConfig{
				ServicesMonitorConfig: ServicesMonitorConfig{
					Regexp:     String("^web.*"),
					Datacenter: String("dc"),
					Namespace:  String("namespace"),
					Filter:     String("filter"),
					CTSUserDefinedMeta: map[string]string{
						"key": "value",
					},
				},
				SourceIncludesVar: Bool(false),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.a.Copy()
			if tc.a == nil {
				// returned nil interface has nil type, which is unequal to tc.a
				assert.Nil(t, r)
			} else {
				assert.Equal(t, tc.a, r)
			}
		})
	}
}

func TestServicesConditionConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ServicesConditionConfig
		b    *ServicesConditionConfig
		r    *ServicesConditionConfig
	}{
		{
			"nil_a",
			nil,
			&ServicesConditionConfig{},
			&ServicesConditionConfig{},
		},
		{
			"nil_b",
			&ServicesConditionConfig{},
			nil,
			&ServicesConditionConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&ServicesConditionConfig{},
			&ServicesConditionConfig{},
			&ServicesConditionConfig{},
		},
		{
			"source_includes_var_overrides",
			&ServicesConditionConfig{SourceIncludesVar: Bool(true)},
			&ServicesConditionConfig{SourceIncludesVar: Bool(false)},
			&ServicesConditionConfig{SourceIncludesVar: Bool(false)},
		},
		{
			"source_includes_var_empty_one",
			&ServicesConditionConfig{SourceIncludesVar: Bool(true)},
			&ServicesConditionConfig{},
			&ServicesConditionConfig{SourceIncludesVar: Bool(true)},
		},
		{
			"source_includes_var_empty_two",
			&ServicesConditionConfig{},
			&ServicesConditionConfig{SourceIncludesVar: Bool(true)},
			&ServicesConditionConfig{SourceIncludesVar: Bool(true)},
		},
		{
			"source_includes_var_empty_same",
			&ServicesConditionConfig{SourceIncludesVar: Bool(true)},
			&ServicesConditionConfig{SourceIncludesVar: Bool(true)},
			&ServicesConditionConfig{SourceIncludesVar: Bool(true)},
		},
		{
			"happy_path",
			&ServicesConditionConfig{
				ServicesMonitorConfig: ServicesMonitorConfig{
					Regexp:             String("regexp"),
					Datacenter:         String("datacenter_overridden"),
					Namespace:          nil,
					Filter:             nil,
					CTSUserDefinedMeta: map[string]string{},
				},
				SourceIncludesVar: Bool(true),
			},
			&ServicesConditionConfig{
				ServicesMonitorConfig: ServicesMonitorConfig{
					Regexp:             nil,
					Datacenter:         String("datacenter"),
					Namespace:          String("namespace"),
					Filter:             nil,
					CTSUserDefinedMeta: map[string]string{},
				},
				SourceIncludesVar: Bool(false),
			},
			&ServicesConditionConfig{
				ServicesMonitorConfig: ServicesMonitorConfig{
					Regexp:             String("regexp"),
					Datacenter:         String("datacenter"),
					Namespace:          String("namespace"),
					Filter:             nil,
					CTSUserDefinedMeta: map[string]string{},
				},
				SourceIncludesVar: Bool(false),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			if tc.r == nil {
				// returned nil interface has nil type, which is unequal to tc.r
				assert.Nil(t, r)
			} else {
				assert.Equal(t, tc.r, r)
			}
		})
	}
}

func TestServicesConditionConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *ServicesConditionConfig
		r    *ServicesConditionConfig
	}{
		{
			"nil",
			nil,
			nil,
		},
		{
			"happy_path",
			&ServicesConditionConfig{},
			&ServicesConditionConfig{
				ServicesMonitorConfig: ServicesMonitorConfig{
					Regexp:             nil,
					Names:              []string{},
					Datacenter:         String(""),
					Namespace:          String(""),
					Filter:             String(""),
					CTSUserDefinedMeta: map[string]string{},
				},
				SourceIncludesVar: Bool(true),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.i.Finalize()
			assert.Equal(t, tc.r, tc.i)
		})
	}
}

func TestServicesConditionConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *ServicesConditionConfig
	}{
		{
			"valid",
			false,
			&ServicesConditionConfig{
				ServicesMonitorConfig: ServicesMonitorConfig{
					Regexp: String(".*"),
				},
			},
		},
		{
			"invalid",
			true,
			&ServicesConditionConfig{
				ServicesMonitorConfig: ServicesMonitorConfig{
					Regexp: String("*"),
				},
			},
		},
		{
			"nil",
			false,
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.c.Validate()
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServicesCondition_GoString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		i        *ServicesConditionConfig
		expected string
	}{
		{
			"nil",
			nil,
			"(*ServicesConditionConfig)(nil)",
		},
		{
			"fully_configured",
			&ServicesConditionConfig{
				ServicesMonitorConfig: ServicesMonitorConfig{
					Regexp:     String("^api$"),
					Datacenter: String("dc"),
					Namespace:  String("namespace"),
					Filter:     String("filter"),
					CTSUserDefinedMeta: map[string]string{
						"key": "value",
					},
				},
			},
			"&ServicesConditionConfig{&ServicesMonitorConfig{Regexp:^api$, Names:[], " +
				"Datacenter:dc, Namespace:namespace, Filter:filter, " +
				"CTSUserDefinedMeta:map[key:value]}, SourceIncludesVar:false}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.i.GoString()
			assert.Equal(t, tc.expected, actual)
		})
	}
}
