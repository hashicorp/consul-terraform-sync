package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServicesMonitorConfig_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ServicesMonitorConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&ServicesMonitorConfig{},
		},
		{
			"regexp_fully_configured",
			&ServicesMonitorConfig{
				Regexp:     String("^web.*"),
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				Filter:     String("filter"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value",
				},
			},
		},
		{
			"names_fully_configured",
			&ServicesMonitorConfig{
				Names:      []string{"web", "api"},
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				Filter:     String("filter"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value",
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

func TestServicesMonitorConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ServicesMonitorConfig
		b    *ServicesMonitorConfig
		r    *ServicesMonitorConfig
	}{
		{
			"nil_a",
			nil,
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{},
		},
		{
			"nil_b",
			&ServicesMonitorConfig{},
			nil,
			&ServicesMonitorConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{},
		},
		{
			"regexp_overrides",
			&ServicesMonitorConfig{Regexp: String("same")},
			&ServicesMonitorConfig{Regexp: String("different")},
			&ServicesMonitorConfig{Regexp: String("different")},
		},
		{
			"regexp_empty_one",
			&ServicesMonitorConfig{Regexp: String("same")},
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{Regexp: String("same")},
		},
		{
			"regexp_empty_two",
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{Regexp: String("same")},
			&ServicesMonitorConfig{Regexp: String("same")},
		},
		{
			"regexp_empty_same",
			&ServicesMonitorConfig{Regexp: String("same")},
			&ServicesMonitorConfig{Regexp: String("same")},
			&ServicesMonitorConfig{Regexp: String("same")},
		},
		{
			"names_merges",
			&ServicesMonitorConfig{Names: []string{"a"}},
			&ServicesMonitorConfig{Names: []string{"b"}},
			&ServicesMonitorConfig{Names: []string{"a", "b"}},
		},
		{
			"names_empty_one",
			&ServicesMonitorConfig{Names: []string{"service"}},
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{Names: []string{"service"}},
		},
		{
			"names_empty_two",
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{Names: []string{"service"}},
			&ServicesMonitorConfig{Names: []string{"service"}},
		},
		{
			"datacenter_overrides",
			&ServicesMonitorConfig{Datacenter: String("datacenter")},
			&ServicesMonitorConfig{Datacenter: String("dc")},
			&ServicesMonitorConfig{Datacenter: String("dc")},
		},
		{
			"datacenter_empty_one",
			&ServicesMonitorConfig{Datacenter: String("datacenter")},
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{Datacenter: String("datacenter")},
		},
		{
			"datacenter_empty_two",
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{Datacenter: String("datacenter")},
			&ServicesMonitorConfig{Datacenter: String("datacenter")},
		},
		{
			"datacenter_same",
			&ServicesMonitorConfig{Datacenter: String("datacenter")},
			&ServicesMonitorConfig{Datacenter: String("datacenter")},
			&ServicesMonitorConfig{Datacenter: String("datacenter")},
		},
		{
			"namespace_overrides",
			&ServicesMonitorConfig{Namespace: String("namespace")},
			&ServicesMonitorConfig{Namespace: String("ns")},
			&ServicesMonitorConfig{Namespace: String("ns")},
		},
		{
			"namespace_empty_one",
			&ServicesMonitorConfig{Namespace: String("namespace")},
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{Namespace: String("namespace")},
		},
		{
			"namespace_empty_two",
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{Namespace: String("namespace")},
			&ServicesMonitorConfig{Namespace: String("namespace")},
		},
		{
			"namespace_same",
			&ServicesMonitorConfig{Namespace: String("namespace")},
			&ServicesMonitorConfig{Namespace: String("namespace")},
			&ServicesMonitorConfig{Namespace: String("namespace")},
		},
		{
			"filter_overrides",
			&ServicesMonitorConfig{Filter: String("filter")},
			&ServicesMonitorConfig{Filter: String("f")},
			&ServicesMonitorConfig{Filter: String("f")},
		},
		{
			"filter_empty_one",
			&ServicesMonitorConfig{Filter: String("filter")},
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{Filter: String("filter")},
		},
		{
			"filter_empty_two",
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{Filter: String("filter")},
			&ServicesMonitorConfig{Filter: String("filter")},
		},
		{
			"filter_same",
			&ServicesMonitorConfig{Filter: String("filter")},
			&ServicesMonitorConfig{Filter: String("filter")},
			&ServicesMonitorConfig{Filter: String("filter")},
		},
		{
			"cts_user_defined_meta_overrides",
			&ServicesMonitorConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
			&ServicesMonitorConfig{CTSUserDefinedMeta: map[string]string{"key": "new-value"}},
			&ServicesMonitorConfig{CTSUserDefinedMeta: map[string]string{"key": "new-value"}},
		},
		{
			"cts_user_defined_meta_empty_one",
			&ServicesMonitorConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
		},
		{
			"cts_user_defined_meta_empty_two",
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
			&ServicesMonitorConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
		},
		{
			"cts_user_defined_meta_same",
			&ServicesMonitorConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
			&ServicesMonitorConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
			&ServicesMonitorConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
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

func TestServicesMonitorConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		s    []string
		i    *ServicesMonitorConfig
		r    *ServicesMonitorConfig
	}{
		{
			"nil",
			[]string{},
			nil,
			nil,
		},
		{
			"empty",
			[]string{},
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{
				Regexp:             nil,
				Names:              []string{},
				Datacenter:         String(""),
				Namespace:          String(""),
				Filter:             String(""),
				CTSUserDefinedMeta: map[string]string{},
			},
		},
		{
			"regexp_fully_configured",
			[]string{},
			&ServicesMonitorConfig{
				Regexp:     String("^web.*"),
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				Filter:     String("filter"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value",
				},
			},
			&ServicesMonitorConfig{
				Regexp:     String("^web.*"),
				Names:      []string{},
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				Filter:     String("filter"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value",
				},
			},
		},
		{
			"names_fully_configured",
			[]string{},
			&ServicesMonitorConfig{
				Names:      []string{"api"},
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				Filter:     String("filter"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value",
				},
			},
			&ServicesMonitorConfig{
				Names:      []string{"api"},
				Regexp:     nil,
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				Filter:     String("filter"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value",
				},
			},
		},
		{
			"services_param_unused",
			[]string{"api"},
			&ServicesMonitorConfig{},
			&ServicesMonitorConfig{
				Regexp:             nil,
				Names:              []string{},
				Datacenter:         String(""),
				Namespace:          String(""),
				Filter:             String(""),
				CTSUserDefinedMeta: map[string]string{},
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

func TestServicesMonitorConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *ServicesMonitorConfig
	}{
		{
			"nil",
			false,
			nil,
		},
		{
			"valid_with_regexp",
			false,
			&ServicesMonitorConfig{
				Regexp: String(".*"),
			},
		},
		{
			"valid_with_names",
			false,
			&ServicesMonitorConfig{
				Names: []string{"api"},
			},
		},
		{
			"valid_regexp_empty_string",
			false,
			&ServicesMonitorConfig{
				Regexp: String(""),
			},
		},
		{
			"invalid_regexp",
			true,
			&ServicesMonitorConfig{
				Regexp: String("*"),
			},
		},
		{
			"invalid_both_regexp_and_names_configured",
			true,
			&ServicesMonitorConfig{
				Regexp: String(".*"),
				Names:  []string{"api"},
			},
		},
		{
			"invalid_no_regexp_no_names_configured",
			true,
			&ServicesMonitorConfig{},
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

func TestServicesMonitorConfig_GoString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		i        *ServicesMonitorConfig
		expected string
	}{
		{
			"nil",
			nil,
			"(*ServicesMonitorConfig)(nil)",
		},
		{
			"regexp_fully_configured",
			&ServicesMonitorConfig{
				Regexp:     String("^api$"),
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				Filter:     String("filter"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value",
				},
			},
			"&ServicesMonitorConfig{Regexp:^api$, Names:[], Datacenter:dc, " +
				"Namespace:namespace, Filter:filter, " +
				"CTSUserDefinedMeta:map[key:value]}",
		},
		{
			"names_fully_configured",
			&ServicesMonitorConfig{
				Names:      []string{"api", "web"},
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				Filter:     String("filter"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value",
				},
			},
			"&ServicesMonitorConfig{Regexp:, Names:[api web], Datacenter:dc, " +
				"Namespace:namespace, Filter:filter, " +
				"CTSUserDefinedMeta:map[key:value]}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.i.GoString()
			require.Equal(t, tc.expected, actual)
		})
	}
}
