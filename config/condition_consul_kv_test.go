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
				ConsulKVMonitorConfig{
					Path:              String("key-path"),
					Recurse:           Bool(true),
					SourceIncludesVar: Bool(true),
					Datacenter:        String("dc2"),
					Namespace:         String("ns2"),
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
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Path: String("different")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Path: String("different")}},
		},
		{
			"path_empty_one",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Path: String("same")}},
		},
		{
			"path_empty_two",
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Path: String("same")}},
		},
		{
			"path_empty_same",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Path: String("same")}},
		},
		{
			"recurse_overrides",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Recurse: Bool(false)}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Recurse: Bool(false)}},
		},
		{
			"recurse_empty_one",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
		},
		{
			"recurse_empty_two",
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
		},
		{
			"recurse_empty_same",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
		},
		{
			"source_includes_var_overrides",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{SourceIncludesVar: Bool(true)}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{SourceIncludesVar: Bool(false)}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{SourceIncludesVar: Bool(false)}},
		},
		{
			"source_includes_var_empty_one",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{SourceIncludesVar: Bool(true)}},
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{SourceIncludesVar: Bool(true)}},
		},
		{
			"source_includes_var_empty_two",
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{SourceIncludesVar: Bool(true)}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{SourceIncludesVar: Bool(true)}},
		},
		{
			"source_includes_var_empty_same",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{SourceIncludesVar: Bool(true)}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{SourceIncludesVar: Bool(true)}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{SourceIncludesVar: Bool(true)}},
		},
		{
			"datacenter_overrides",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Datacenter: String("different")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Datacenter: String("different")}},
		},
		{
			"datacenter_empty_one",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
		},
		{
			"datacenter_empty_two",
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
		},
		{
			"datacenter_empty_same",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
		},
		{
			"namespace_overrides",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Namespace: String("different")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Namespace: String("different")}},
		},
		{
			"namespace_empty_one",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
		},
		{
			"namespace_empty_two",
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
		},
		{
			"namespace_empty_same",
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVConditionConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
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
		s    []string
		i    *ConsulKVConditionConfig
		r    *ConsulKVConditionConfig
	}{
		{
			"empty",
			[]string{},
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{
				ConsulKVMonitorConfig{
					Path:              String(""),
					Recurse:           Bool(false),
					SourceIncludesVar: Bool(false),
					Datacenter:        String(""),
					Namespace:         String(""),
				},
			},
		},
		{
			"services_ignored",
			[]string{"api"},
			&ConsulKVConditionConfig{},
			&ConsulKVConditionConfig{
				ConsulKVMonitorConfig{
					Path:              String(""),
					Recurse:           Bool(false),
					SourceIncludesVar: Bool(false),
					Datacenter:        String(""),
					Namespace:         String(""),
				},
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
				ConsulKVMonitorConfig{
					Path:              String("key-path"),
					Recurse:           Bool(true),
					SourceIncludesVar: Bool(true),
					Datacenter:        String("dc2"),
					Namespace:         String("ns2"),
				},
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
