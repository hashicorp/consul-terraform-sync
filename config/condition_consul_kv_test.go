package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsulKVConditionConfig_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ConsulKVConditionConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&ConsulKVConditionConfig{},
		},
		{
			"fully_configured",
			&ConsulKVConditionConfig{
				ConsulKVMonitorConfig: ConsulKVMonitorConfig{
					Path:       String("key-path"),
					Recurse:    Bool(true),
					Datacenter: String("dc2"),
					Namespace:  String("ns2"),
				},
				SourceIncludesVar: Bool(true),
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

func TestConsulKVConditionConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ConsulKVConditionConfig
		b    *ConsulKVConditionConfig
		r    *ConsulKVConditionConfig
	}{
		{
			"nil_a",
			nil,
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{},
		},
		{
			"nil_b",
			&ConsulKVConditionConfig{},
			nil,
			&ConsulKVConditionConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{},
		},
		{
			"path_overrides",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Path: String("different")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Path: String("different")}},
		},
		{
			"path_empty_one",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Path: String("same")}},
		},
		{
			"path_empty_two",
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Path: String("same")}},
		},
		{
			"path_empty_same",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Path: String("same")}},
		},
		{
			"recurse_overrides",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Recurse: Bool(false)}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Recurse: Bool(false)}},
		},
		{
			"recurse_empty_one",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Recurse: Bool(true)}},
		},
		{
			"recurse_empty_two",
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Recurse: Bool(true)}},
		},
		{
			"recurse_empty_same",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Recurse: Bool(true)}},
		},
		{
			"source_includes_var_overrides",
			&ConsulKVConditionConfig{SourceIncludesVar: Bool(true)},
			&ConsulKVConditionConfig{SourceIncludesVar: Bool(false)},
			&ConsulKVConditionConfig{SourceIncludesVar: Bool(false)},
		},
		{
			"source_includes_var_empty_one",
			&ConsulKVConditionConfig{SourceIncludesVar: Bool(true)},
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{SourceIncludesVar: Bool(true)},
		},
		{
			"source_includes_var_empty_two",
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{SourceIncludesVar: Bool(true)},
			&ConsulKVConditionConfig{SourceIncludesVar: Bool(true)},
		},
		{
			"source_includes_var_empty_same",
			&ConsulKVConditionConfig{SourceIncludesVar: Bool(true)},
			&ConsulKVConditionConfig{SourceIncludesVar: Bool(true)},
			&ConsulKVConditionConfig{SourceIncludesVar: Bool(true)},
		},
		{
			"datacenter_overrides",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Datacenter: String("different")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Datacenter: String("different")}},
		},
		{
			"datacenter_empty_one",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Datacenter: String("same")}},
		},
		{
			"datacenter_empty_two",
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Datacenter: String("same")}},
		},
		{
			"datacenter_empty_same",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Datacenter: String("same")}},
		},
		{
			"namespace_overrides",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Namespace: String("different")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Namespace: String("different")}},
		},
		{
			"namespace_empty_one",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Namespace: String("same")}},
		},
		{
			"namespace_empty_two",
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Namespace: String("same")}},
		},
		{
			"namespace_empty_same",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig: ConsulKVMonitorConfig{Namespace: String("same")}},
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

func TestConsulKVConditionConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *ConsulKVConditionConfig
		r    *ConsulKVConditionConfig
	}{
		{
			"empty",
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{
				ConsulKVMonitorConfig: ConsulKVMonitorConfig{
					Path:       String(""),
					Recurse:    Bool(false),
					Datacenter: String(""),
					Namespace:  String(""),
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

func TestConsulKVConditionConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *ConsulKVConditionConfig
	}{
		{
			"happy_path",
			false,
			&ConsulKVConditionConfig{
				ConsulKVMonitorConfig: ConsulKVMonitorConfig{
					Path:       String("key-path"),
					Recurse:    Bool(true),
					Datacenter: String("dc2"),
					Namespace:  String("ns2"),
				},
				SourceIncludesVar: Bool(true),
			},
		},
		{
			"nil_path",
			true,
			&ConsulKVConditionConfig{},
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
