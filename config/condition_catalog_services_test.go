package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCatalogServicesConditionConfig_Copy(t *testing.T) {
	t.Parallel()

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
			"fully_configured",
			&CatalogServicesConditionConfig{
				Regexp:      String(".*"),
				EnableTfVar: Bool(true),
				Datacenter:  String("dc2"),
				Namespace:   String("ns2"),
				NodeMeta: map[string]string{
					"key1": "value1",
					"key2": "value2",
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

func TestCatalogServicesConditionConfig_Merge(t *testing.T) {
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
			&CatalogServicesConditionConfig{Regexp: String("same")},
			&CatalogServicesConditionConfig{Regexp: String("different")},
			&CatalogServicesConditionConfig{Regexp: String("different")},
		},
		{
			"regexp_empty_one",
			&CatalogServicesConditionConfig{Regexp: String("same")},
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{Regexp: String("same")},
		},
		{
			"regexp_empty_two",
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{Regexp: String("same")},
			&CatalogServicesConditionConfig{Regexp: String("same")},
		},
		{
			"regexp_empty_same",
			&CatalogServicesConditionConfig{Regexp: String("same")},
			&CatalogServicesConditionConfig{Regexp: String("same")},
			&CatalogServicesConditionConfig{Regexp: String("same")},
		},
		{
			"enable_tf_var_overrides",
			&CatalogServicesConditionConfig{EnableTfVar: Bool(true)},
			&CatalogServicesConditionConfig{EnableTfVar: Bool(false)},
			&CatalogServicesConditionConfig{EnableTfVar: Bool(false)},
		},
		{
			"enable_tf_var_empty_one",
			&CatalogServicesConditionConfig{EnableTfVar: Bool(true)},
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{EnableTfVar: Bool(true)},
		},
		{
			"enable_tf_var_empty_two",
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{EnableTfVar: Bool(true)},
			&CatalogServicesConditionConfig{EnableTfVar: Bool(true)},
		},
		{
			"enable_tf_var_empty_same",
			&CatalogServicesConditionConfig{EnableTfVar: Bool(true)},
			&CatalogServicesConditionConfig{EnableTfVar: Bool(true)},
			&CatalogServicesConditionConfig{EnableTfVar: Bool(true)},
		},
		{
			"datacenter_overrides",
			&CatalogServicesConditionConfig{Datacenter: String("same")},
			&CatalogServicesConditionConfig{Datacenter: String("different")},
			&CatalogServicesConditionConfig{Datacenter: String("different")},
		},
		{
			"datacenter_empty_one",
			&CatalogServicesConditionConfig{Datacenter: String("same")},
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{Datacenter: String("same")},
		},
		{
			"datacenter_empty_two",
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{Datacenter: String("same")},
			&CatalogServicesConditionConfig{Datacenter: String("same")},
		},
		{
			"datacenter_empty_same",
			&CatalogServicesConditionConfig{Datacenter: String("same")},
			&CatalogServicesConditionConfig{Datacenter: String("same")},
			&CatalogServicesConditionConfig{Datacenter: String("same")},
		},
		{
			"namespace_overrides",
			&CatalogServicesConditionConfig{Namespace: String("same")},
			&CatalogServicesConditionConfig{Namespace: String("different")},
			&CatalogServicesConditionConfig{Namespace: String("different")},
		},
		{
			"namespace_empty_one",
			&CatalogServicesConditionConfig{Namespace: String("same")},
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{Namespace: String("same")},
		},
		{
			"namespace_empty_two",
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{Namespace: String("same")},
			&CatalogServicesConditionConfig{Namespace: String("same")},
		},
		{
			"namespace_empty_same",
			&CatalogServicesConditionConfig{Namespace: String("same")},
			&CatalogServicesConditionConfig{Namespace: String("same")},
			&CatalogServicesConditionConfig{Namespace: String("same")},
		},
		{
			"node_meta_overrides",
			&CatalogServicesConditionConfig{NodeMeta: map[string]string{"key": "value"}},
			&CatalogServicesConditionConfig{NodeMeta: map[string]string{"key": "new-value"}},
			&CatalogServicesConditionConfig{NodeMeta: map[string]string{"key": "new-value"}},
		},
		{
			"node_meta_empty_one",
			&CatalogServicesConditionConfig{NodeMeta: map[string]string{"key": "value"}},
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{NodeMeta: map[string]string{"key": "value"}},
		},
		{
			"node_meta_empty_two",
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{NodeMeta: map[string]string{"key": "value"}},
			&CatalogServicesConditionConfig{NodeMeta: map[string]string{"key": "value"}},
		},
		{
			"node_meta_same",
			&CatalogServicesConditionConfig{NodeMeta: map[string]string{"key": "value"}},
			&CatalogServicesConditionConfig{NodeMeta: map[string]string{"key": "value"}},
			&CatalogServicesConditionConfig{NodeMeta: map[string]string{"key": "value"}},
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

func TestCatalogServicesConditionConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		s    []string
		i    *CatalogServicesConditionConfig
		r    *CatalogServicesConditionConfig
	}{
		{
			"empty",
			[]string{},
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{
				Regexp:      String(""),
				EnableTfVar: Bool(false),
				Datacenter:  String(""),
				Namespace:   String(""),
				NodeMeta:    map[string]string{},
			},
		},
		{
			"pass_in_services",
			[]string{"api"},
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{
				Regexp:      String("^api$"),
				EnableTfVar: Bool(false),
				Datacenter:  String(""),
				Namespace:   String(""),
				NodeMeta:    map[string]string{},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.i.Finalize(tc.s)
			assert.Equal(t, tc.r, tc.i)
		})
	}
}

func TestCatalogServicesConditionConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *CatalogServicesConditionConfig
	}{
		{
			"happy_path",
			false,
			&CatalogServicesConditionConfig{
				Regexp:      String(".*"),
				EnableTfVar: Bool(true),
				Datacenter:  String("dc2"),
				Namespace:   String("ns2"),
				NodeMeta: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
		},
		{
			"invalid_regexp",
			true,
			&CatalogServicesConditionConfig{
				Regexp: String("*"),
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
