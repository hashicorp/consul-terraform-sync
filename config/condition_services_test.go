// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServicesConditionConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &ServicesConditionConfig{}
	finalizedConf.Finalize()

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
			"finalized",
			finalizedConf,
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
				UseAsModuleInput:            Bool(false),
				DeprecatedSourceIncludesVar: Bool(false),
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
			&ServicesConditionConfig{DeprecatedSourceIncludesVar: Bool(true)},
			&ServicesConditionConfig{DeprecatedSourceIncludesVar: Bool(false)},
			&ServicesConditionConfig{DeprecatedSourceIncludesVar: Bool(false)},
		},
		{
			"source_includes_var_empty_one",
			&ServicesConditionConfig{DeprecatedSourceIncludesVar: Bool(true)},
			&ServicesConditionConfig{},
			&ServicesConditionConfig{DeprecatedSourceIncludesVar: Bool(true)},
		},
		{
			"source_includes_var_empty_two",
			&ServicesConditionConfig{},
			&ServicesConditionConfig{DeprecatedSourceIncludesVar: Bool(true)},
			&ServicesConditionConfig{DeprecatedSourceIncludesVar: Bool(true)},
		},
		{
			"source_includes_var_empty_same",
			&ServicesConditionConfig{DeprecatedSourceIncludesVar: Bool(true)},
			&ServicesConditionConfig{DeprecatedSourceIncludesVar: Bool(true)},
			&ServicesConditionConfig{DeprecatedSourceIncludesVar: Bool(true)},
		},
		{
			"use_as_module_input_overrides",
			&ServicesConditionConfig{UseAsModuleInput: Bool(true)},
			&ServicesConditionConfig{UseAsModuleInput: Bool(false)},
			&ServicesConditionConfig{UseAsModuleInput: Bool(false)},
		},
		{
			"use_as_module_input_empty_one",
			&ServicesConditionConfig{UseAsModuleInput: Bool(true)},
			&ServicesConditionConfig{},
			&ServicesConditionConfig{UseAsModuleInput: Bool(true)},
		},
		{
			"use_as_module_input_empty_two",
			&ServicesConditionConfig{},
			&ServicesConditionConfig{UseAsModuleInput: Bool(true)},
			&ServicesConditionConfig{UseAsModuleInput: Bool(true)},
		},
		{
			"use_as_module_input_empty_same",
			&ServicesConditionConfig{UseAsModuleInput: Bool(true)},
			&ServicesConditionConfig{UseAsModuleInput: Bool(true)},
			&ServicesConditionConfig{UseAsModuleInput: Bool(true)},
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
				DeprecatedSourceIncludesVar: Bool(true),
				UseAsModuleInput:            Bool(true),
			},
			&ServicesConditionConfig{
				ServicesMonitorConfig: ServicesMonitorConfig{
					Regexp:             nil,
					Datacenter:         String("datacenter"),
					Namespace:          String("namespace"),
					Filter:             nil,
					CTSUserDefinedMeta: map[string]string{},
				},
				DeprecatedSourceIncludesVar: Bool(false),
				UseAsModuleInput:            Bool(false),
			},
			&ServicesConditionConfig{
				ServicesMonitorConfig: ServicesMonitorConfig{
					Regexp:             String("regexp"),
					Datacenter:         String("datacenter"),
					Namespace:          String("namespace"),
					Filter:             nil,
					CTSUserDefinedMeta: map[string]string{},
				},
				DeprecatedSourceIncludesVar: Bool(false),
				UseAsModuleInput:            Bool(false),
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
			"empty",
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
				UseAsModuleInput: Bool(true),
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

func TestServicesConditionConfig_Finalize_DeprecatedSourceIncludesVar(t *testing.T) {
	cases := []struct {
		name     string
		i        *ServicesConditionConfig
		expected bool
	}{
		{
			"use_as_module_input_configured",
			&ServicesConditionConfig{
				UseAsModuleInput: Bool(false),
			},
			false,
		},
		{
			"source_includes_var_configured",
			&ServicesConditionConfig{
				DeprecatedSourceIncludesVar: Bool(false),
			},
			false,
		},
		{
			"both_configured",
			&ServicesConditionConfig{
				UseAsModuleInput:            Bool(false),
				DeprecatedSourceIncludesVar: Bool(true),
			},
			false,
		},

		{
			"neither_configured",
			&ServicesConditionConfig{},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.i.Finalize()
			assert.Equal(t, tc.expected, *tc.i.UseAsModuleInput)
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
				UseAsModuleInput: Bool(false),
			},
			"&ServicesConditionConfig{&ServicesMonitorConfig{Regexp:^api$, Names:[], " +
				"Datacenter:dc, Namespace:namespace, Filter:filter, " +
				"CTSUserDefinedMeta:map[key:value]}, UseAsModuleInput:false}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.i.GoString()
			assert.Equal(t, tc.expected, actual)
		})
	}
}
