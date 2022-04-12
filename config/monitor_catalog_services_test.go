package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCatalogServicesMonitorConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &CatalogServicesConditionConfig{}
	finalizedConf.Finalize()

	cases := []struct {
		name string
		a    *CatalogServicesConditionConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&CatalogServicesConditionConfig{},
		},
		{
			"finalized",
			finalizedConf,
		},
		{
			"fully_configured",
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{
					Regexp:           String(".*"),
					UseAsModuleInput: Bool(true),
					Datacenter:       String("dc2"),
					Namespace:        String("ns2"),
					NodeMeta: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
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

func TestCatalogServicesMonitorConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *CatalogServicesConditionConfig
		b    *CatalogServicesConditionConfig
		r    *CatalogServicesConditionConfig
	}{
		{
			"nil_a",
			nil,
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{},
		},
		{
			"nil_b",
			&CatalogServicesConditionConfig{},
			nil,
			&CatalogServicesConditionConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{},
		},
		{
			"regexp_overrides",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String("same")}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String("different")}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String("different")}},
		},
		{
			"regexp_empty_one",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String("same")}},
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String("same")}},
		},
		{
			"regexp_empty_two",
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String("same")}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String("same")}},
		},
		{
			"regexp_empty_same",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String("same")}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String("same")}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String("same")}},
		},
		{
			"source_includes_var_overrides",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{DeprecatedSourceIncludesVar: Bool(true)}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{DeprecatedSourceIncludesVar: Bool(false)}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{DeprecatedSourceIncludesVar: Bool(false)}},
		},
		{
			"source_includes_var_empty_one",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{DeprecatedSourceIncludesVar: Bool(true)}},
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{DeprecatedSourceIncludesVar: Bool(true)}},
		},
		{
			"source_includes_var_empty_two",
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{DeprecatedSourceIncludesVar: Bool(true)}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{DeprecatedSourceIncludesVar: Bool(true)}},
		},
		{
			"source_includes_var_empty_same",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{DeprecatedSourceIncludesVar: Bool(true)}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{DeprecatedSourceIncludesVar: Bool(true)}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{DeprecatedSourceIncludesVar: Bool(true)}},
		},
		{
			"use_as_module_input_overrides",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{UseAsModuleInput: Bool(true)}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{UseAsModuleInput: Bool(false)}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{UseAsModuleInput: Bool(false)}},
		},
		{
			"use_as_module_input_empty_one",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{UseAsModuleInput: Bool(true)}},
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{UseAsModuleInput: Bool(true)}},
		},
		{
			"use_as_module_input_empty_two",
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{UseAsModuleInput: Bool(true)}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{UseAsModuleInput: Bool(true)}},
		},
		{
			"use_as_module_input_empty_same",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{UseAsModuleInput: Bool(true)}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{UseAsModuleInput: Bool(true)}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{UseAsModuleInput: Bool(true)}},
		},
		{
			"datacenter_overrides",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Datacenter: String("same")}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Datacenter: String("different")}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Datacenter: String("different")}},
		},
		{
			"datacenter_empty_one",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Datacenter: String("same")}},
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Datacenter: String("same")}},
		},
		{
			"datacenter_empty_two",
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Datacenter: String("same")}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Datacenter: String("same")}},
		},
		{
			"datacenter_empty_same",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Datacenter: String("same")}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Datacenter: String("same")}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Datacenter: String("same")}},
		},
		{
			"namespace_overrides",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Namespace: String("same")}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Namespace: String("different")}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Namespace: String("different")}},
		},
		{
			"namespace_empty_one",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Namespace: String("same")}},
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Namespace: String("same")}},
		},
		{
			"namespace_empty_two",
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Namespace: String("same")}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Namespace: String("same")}},
		},
		{
			"namespace_empty_same",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Namespace: String("same")}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Namespace: String("same")}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Namespace: String("same")}},
		},
		{
			"node_meta_overrides",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{NodeMeta: map[string]string{"key": "value"}}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{NodeMeta: map[string]string{"key": "new-value"}}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{NodeMeta: map[string]string{"key": "new-value"}}},
		},
		{
			"node_meta_empty_one",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{NodeMeta: map[string]string{"key": "value"}}},
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{NodeMeta: map[string]string{"key": "value"}}},
		},
		{
			"node_meta_empty_two",
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{NodeMeta: map[string]string{"key": "value"}}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{NodeMeta: map[string]string{"key": "value"}}},
		},
		{
			"node_meta_same",
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{NodeMeta: map[string]string{"key": "value"}}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{NodeMeta: map[string]string{"key": "value"}}},
			&CatalogServicesConditionConfig{CatalogServicesMonitorConfig{NodeMeta: map[string]string{"key": "value"}}},
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

func TestCatalogServicesMonitorConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *CatalogServicesConditionConfig
		r    *CatalogServicesConditionConfig
	}{
		{
			"nil",
			nil,
			nil,
		},
		{
			"empty",
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{
					Regexp:           nil,
					UseAsModuleInput: Bool(true),
					Datacenter:       String(""),
					Namespace:        String(""),
					NodeMeta:         map[string]string{},
				},
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

func TestCatalogServicesConditionConfig_Finalize_DeprecatedSourceIncludesVar(t *testing.T) {
	cases := []struct {
		name     string
		i        *CatalogServicesConditionConfig
		expected bool
	}{
		{
			"use_as_module_input_configured",
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{
					UseAsModuleInput: Bool(false),
				},
			},
			false,
		},
		{
			"source_includes_var_configured",
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{
					DeprecatedSourceIncludesVar: Bool(false),
				},
			},
			false,
		},
		{
			"both_configured",
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{
					UseAsModuleInput:            Bool(false),
					DeprecatedSourceIncludesVar: Bool(true),
				},
			},
			false,
		},

		{
			"neither_configured",
			&CatalogServicesConditionConfig{},
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

func TestCatalogServicesMonitorConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *CatalogServicesConditionConfig
	}{
		{
			"nil",
			false,
			nil,
		},
		{
			"nil_regexp",
			true,
			&CatalogServicesConditionConfig{},
		},
		{
			"invalid_regexp",
			true,
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{
					Regexp: String("*"),
				},
			},
		},
		{
			"valid_empty_regexp",
			false,
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{
					Regexp: String(""),
				},
			},
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
