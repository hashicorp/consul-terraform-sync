package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestConsulKVModuleInputConfig_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ConsulKVModuleInputConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&ConsulKVModuleInputConfig{},
		},
		{
			"fully_configured",
			&ConsulKVModuleInputConfig{
				ConsulKVMonitorConfig{
					Path:       String("key-path"),
					Recurse:    Bool(true),
					Datacenter: String("dc2"),
					Namespace:  String("ns2"),
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

func TestConsulKVModuleInputConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ConsulKVModuleInputConfig
		b    *ConsulKVModuleInputConfig
		r    *ConsulKVModuleInputConfig
	}{
		{
			"nil_a",
			nil,
			&ConsulKVModuleInputConfig{},
			&ConsulKVModuleInputConfig{},
		},
		{
			"nil_b",
			&ConsulKVModuleInputConfig{},
			nil,
			&ConsulKVModuleInputConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&ConsulKVModuleInputConfig{},
			&ConsulKVModuleInputConfig{},
			&ConsulKVModuleInputConfig{},
		},
		{
			"path_overrides",
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Path: String("different")}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Path: String("different")}},
		},
		{
			"path_empty_one",
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVModuleInputConfig{},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
		},
		{
			"path_empty_two",
			&ConsulKVModuleInputConfig{},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
		},
		{
			"path_empty_same",
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
		},
		{
			"recurse_overrides",
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Recurse: Bool(false)}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Recurse: Bool(false)}},
		},
		{
			"recurse_empty_one",
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVModuleInputConfig{},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
		},
		{
			"recurse_empty_two",
			&ConsulKVModuleInputConfig{},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
		},
		{
			"recurse_empty_same",
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
		},
		{
			"datacenter_overrides",
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Datacenter: String("different")}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Datacenter: String("different")}},
		},
		{
			"datacenter_empty_one",
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVModuleInputConfig{},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
		},
		{
			"datacenter_empty_two",
			&ConsulKVModuleInputConfig{},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
		},
		{
			"datacenter_empty_same",
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
		},
		{
			"namespace_overrides",
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Namespace: String("different")}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Namespace: String("different")}},
		},
		{
			"namespace_empty_one",
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVModuleInputConfig{},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
		},
		{
			"namespace_empty_two",
			&ConsulKVModuleInputConfig{},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
		},
		{
			"namespace_empty_same",
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVModuleInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
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

func TestConsulKVModuleInputConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *ConsulKVModuleInputConfig
		r    *ConsulKVModuleInputConfig
	}{
		{
			"empty",
			&ConsulKVModuleInputConfig{},
			&ConsulKVModuleInputConfig{
				ConsulKVMonitorConfig{
					Path:       String(""),
					Recurse:    Bool(false),
					Datacenter: String(""),
					Namespace:  String(""),
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

func TestConsulKVModuleInputConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *ConsulKVModuleInputConfig
	}{
		{
			"happy_path",
			false,
			&ConsulKVModuleInputConfig{
				ConsulKVMonitorConfig{
					Path:       String("key-path"),
					Recurse:    Bool(true),
					Datacenter: String("dc2"),
					Namespace:  String("ns2"),
				},
			},
		},
		{
			"nil_path",
			true,
			&ConsulKVModuleInputConfig{},
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

func TestConsulKVModuleInputConfig_GoString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		ckv      *ConsulKVModuleInputConfig
		expected string
	}{
		{
			"configured services module_input",
			&ConsulKVModuleInputConfig{
				ConsulKVMonitorConfig{
					Path:       String("path"),
					Recurse:    Bool(true),
					Datacenter: String("dc"),
					Namespace:  String("ns"),
				},
			},
			"&ConsulKVModuleInputConfig{" +
				"&ConsulKVMonitorConfig{" +
				"Path:path, " +
				"Recurse:true, " +
				"Datacenter:dc, " +
				"Namespace:ns, " +
				"}" +
				"}",
		},
		{
			"nil services module_input",
			nil,
			"(*ConsulKVModuleInputConfig)(nil)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.ckv.GoString()
			require.Equal(t, actual, tc.expected)
		})
	}
}
