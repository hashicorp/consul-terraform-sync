package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestConsulKVSourceInputConfig_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ConsulKVSourceInputConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&ConsulKVSourceInputConfig{},
		},
		{
			"fully_configured",
			&ConsulKVSourceInputConfig{
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

func TestConsulKVSourceInputConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ConsulKVSourceInputConfig
		b    *ConsulKVSourceInputConfig
		r    *ConsulKVSourceInputConfig
	}{
		{
			"nil_a",
			nil,
			&ConsulKVSourceInputConfig{},
			&ConsulKVSourceInputConfig{},
		},
		{
			"nil_b",
			&ConsulKVSourceInputConfig{},
			nil,
			&ConsulKVSourceInputConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&ConsulKVSourceInputConfig{},
			&ConsulKVSourceInputConfig{},
			&ConsulKVSourceInputConfig{},
		},
		{
			"path_overrides",
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Path: String("different")}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Path: String("different")}},
		},
		{
			"path_empty_one",
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVSourceInputConfig{},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
		},
		{
			"path_empty_two",
			&ConsulKVSourceInputConfig{},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
		},
		{
			"path_empty_same",
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Path: String("same")}},
		},
		{
			"recurse_overrides",
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Recurse: Bool(false)}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Recurse: Bool(false)}},
		},
		{
			"recurse_empty_one",
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVSourceInputConfig{},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
		},
		{
			"recurse_empty_two",
			&ConsulKVSourceInputConfig{},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
		},
		{
			"recurse_empty_same",
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Recurse: Bool(true)}},
		},
		{
			"datacenter_overrides",
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Datacenter: String("different")}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Datacenter: String("different")}},
		},
		{
			"datacenter_empty_one",
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVSourceInputConfig{},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
		},
		{
			"datacenter_empty_two",
			&ConsulKVSourceInputConfig{},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
		},
		{
			"datacenter_empty_same",
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Datacenter: String("same")}},
		},
		{
			"namespace_overrides",
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Namespace: String("different")}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Namespace: String("different")}},
		},
		{
			"namespace_empty_one",
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVSourceInputConfig{},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
		},
		{
			"namespace_empty_two",
			&ConsulKVSourceInputConfig{},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
		},
		{
			"namespace_empty_same",
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
			&ConsulKVSourceInputConfig{ConsulKVMonitorConfig{Namespace: String("same")}},
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

func TestConsulKVSourceInputConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		s    []string
		i    *ConsulKVSourceInputConfig
		r    *ConsulKVSourceInputConfig
	}{
		{
			"empty",
			[]string{},
			&ConsulKVSourceInputConfig{},
			&ConsulKVSourceInputConfig{
				ConsulKVMonitorConfig{
					Path:       String(""),
					Recurse:    Bool(false),
					Datacenter: String(""),
					Namespace:  String(""),
				},
			},
		},
		{
			"services_ignored",
			[]string{"api"},
			&ConsulKVSourceInputConfig{},
			&ConsulKVSourceInputConfig{
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
			tc.i.Finalize(tc.s)
			assert.Equal(t, tc.r, tc.i)
		})
	}
}

func TestConsulKVSourceInputConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *ConsulKVSourceInputConfig
	}{
		{
			"happy_path",
			false,
			&ConsulKVSourceInputConfig{
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
			&ConsulKVSourceInputConfig{},
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

func TestConsulKVSourceInputConfig_GoString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		ckv      *ConsulKVSourceInputConfig
		expected string
	}{
		{
			"configured services source_input",
			&ConsulKVSourceInputConfig{
				ConsulKVMonitorConfig{
					Path:       String("path"),
					Recurse:    Bool(true),
					Datacenter: String("dc"),
					Namespace:  String("ns"),
				},
			},
			"&ConsulKVSourceInputConfig{" +
				"&ConsulKVMonitorConfig{" +
				"Path:path, " +
				"Recurse:true, " +
				"Datacenter:dc, " +
				"Namespace:ns, " +
				"}" +
				"}",
		},
		{
			"nil services source_input",
			nil,
			"(*ConsulKVSourceInputConfig)(nil)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.ckv.GoString()
			require.Equal(t, actual, tc.expected)
		})
	}
}
