package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntentionsMonitorConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &IntentionsMonitorConfig{}
	finalizedConf.Finalize()

	cases := []struct {
		name string
		a    *IntentionsMonitorConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&IntentionsMonitorConfig{},
		},
		{
			"finalized",
			finalizedConf,
		},
		{
			"regexp_fully_configured",
			&IntentionsMonitorConfig{
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				SourceServices: &IntentionsServicesConfig{
					Regexp: String("^web.*"),
				},
				DestinationServices: &IntentionsServicesConfig{
					Regexp: String("^api.*"),
				},
			},
		},
		{
			"names_fully_configured",
			&IntentionsMonitorConfig{
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				SourceServices: &IntentionsServicesConfig{
					Names: []string{"web", "api"},
				},
				DestinationServices: &IntentionsServicesConfig{
					Names: []string{"web", "api"},
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

func TestIntentionsMonitorConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *IntentionsMonitorConfig
		b    *IntentionsMonitorConfig
		r    *IntentionsMonitorConfig
	}{
		{
			"nil_a",
			nil,
			&IntentionsMonitorConfig{},
			&IntentionsMonitorConfig{},
		},
		{
			"nil_b",
			&IntentionsMonitorConfig{},
			nil,
			&IntentionsMonitorConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&IntentionsMonitorConfig{},
			&IntentionsMonitorConfig{},
			&IntentionsMonitorConfig{},
		},
		{
			"datacenter_overrides",
			&IntentionsMonitorConfig{Datacenter: String("datacenter")},
			&IntentionsMonitorConfig{Datacenter: String("dc")},
			&IntentionsMonitorConfig{Datacenter: String("dc")},
		},
		{
			"datacenter_empty_one",
			&IntentionsMonitorConfig{Datacenter: String("datacenter")},
			&IntentionsMonitorConfig{},
			&IntentionsMonitorConfig{Datacenter: String("datacenter")},
		},
		{
			"datacenter_empty_two",
			&IntentionsMonitorConfig{},
			&IntentionsMonitorConfig{Datacenter: String("datacenter")},
			&IntentionsMonitorConfig{Datacenter: String("datacenter")},
		},
		{
			"datacenter_same",
			&IntentionsMonitorConfig{Datacenter: String("datacenter")},
			&IntentionsMonitorConfig{Datacenter: String("datacenter")},
			&IntentionsMonitorConfig{Datacenter: String("datacenter")},
		},
		{
			"namespace_overrides",
			&IntentionsMonitorConfig{Namespace: String("namespace")},
			&IntentionsMonitorConfig{Namespace: String("ns")},
			&IntentionsMonitorConfig{Namespace: String("ns")},
		},
		{
			"namespace_empty_one",
			&IntentionsMonitorConfig{Namespace: String("namespace")},
			&IntentionsMonitorConfig{},
			&IntentionsMonitorConfig{Namespace: String("namespace")},
		},
		{
			"namespace_empty_two",
			&IntentionsMonitorConfig{},
			&IntentionsMonitorConfig{Namespace: String("namespace")},
			&IntentionsMonitorConfig{Namespace: String("namespace")},
		},
		{
			"namespace_same",
			&IntentionsMonitorConfig{Namespace: String("namespace")},
			&IntentionsMonitorConfig{Namespace: String("namespace")},
			&IntentionsMonitorConfig{Namespace: String("namespace")},
		},
		{
			"regexp_overrides",
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Regexp: String("same")}},
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Regexp: String("different")}},
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Regexp: String("different")}},
		},
		{
			"regexp_empty_one",
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Regexp: String("same")}},
			&IntentionsMonitorConfig{},
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Regexp: String("same")}},
		},
		{
			"regexp_empty_two",
			&IntentionsMonitorConfig{},
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Regexp: String("same")}},
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Regexp: String("same")}},
		},
		{
			"regexp_same",
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Regexp: String("same")}},
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Regexp: String("same")}},
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Regexp: String("same")}},
		},
		{
			"names_merges",
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Names: []string{"a"}}},
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Names: []string{"b"}}},
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Names: []string{"a", "b"}}},
		},
		{
			"names_same_merges",
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Names: []string{"a"}}},
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Names: []string{"a"}}},
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Names: []string{"a"}}},
		},
		{
			"names_empty_one",
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Names: []string{"service"}}},
			&IntentionsMonitorConfig{},
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Names: []string{"service"}}},
		},
		{
			"names_empty_two",
			&IntentionsMonitorConfig{},
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Names: []string{"service"}}},
			&IntentionsMonitorConfig{SourceServices: &IntentionsServicesConfig{Names: []string{"service"}}},
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

func TestIntentionsMonitorConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *IntentionsMonitorConfig
		r    *IntentionsMonitorConfig
	}{
		{
			"nil",
			nil,
			nil,
		},
		{
			"empty",
			&IntentionsMonitorConfig{},
			&IntentionsMonitorConfig{
				Datacenter:          String(""),
				Namespace:           String(""),
				SourceServices:      nil,
				DestinationServices: nil,
			},
		},
		{
			"regexp_fully_configured",
			&IntentionsMonitorConfig{
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				SourceServices: &IntentionsServicesConfig{
					Regexp: String("^web.*"),
				},
				DestinationServices: &IntentionsServicesConfig{
					Regexp: String("^api.*"),
				},
			},
			&IntentionsMonitorConfig{
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				SourceServices: &IntentionsServicesConfig{
					Regexp: String("^web.*"),
					Names:  []string{},
				},
				DestinationServices: &IntentionsServicesConfig{
					Regexp: String("^api.*"),
					Names:  []string{},
				},
			},
		},
		{
			"names_fully_configured",
			&IntentionsMonitorConfig{
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				SourceServices: &IntentionsServicesConfig{
					Names: []string{"service"},
				},
				DestinationServices: &IntentionsServicesConfig{
					Names: []string{"service"},
				},
			},
			&IntentionsMonitorConfig{
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				SourceServices: &IntentionsServicesConfig{
					Names: []string{"service"},
				},
				DestinationServices: &IntentionsServicesConfig{
					Names: []string{"service"},
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

func TestIntentionsMonitorConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *IntentionsMonitorConfig
	}{
		{
			"nil",
			false,
			nil,
		},
		{
			"valid_with_regexp",
			false,
			&IntentionsMonitorConfig{
				SourceServices: &IntentionsServicesConfig{
					Regexp: String(".*"),
				},
				DestinationServices: &IntentionsServicesConfig{
					Regexp: String(".*"),
				},
			},
		},
		{
			"valid_with_names",
			false,
			&IntentionsMonitorConfig{
				SourceServices: &IntentionsServicesConfig{
					Names: []string{"api"},
				},
				DestinationServices: &IntentionsServicesConfig{
					Names: []string{"web"},
				},
			},
		},
		{
			"valid_regexp_empty_string",
			false,
			&IntentionsMonitorConfig{
				SourceServices: &IntentionsServicesConfig{
					Regexp: String(""),
				},
				DestinationServices: &IntentionsServicesConfig{
					Regexp: String(""),
				},
			},
		},
		{
			"invalid_source_service_not_configured",
			true,
			&IntentionsMonitorConfig{
				DestinationServices: &IntentionsServicesConfig{
					Regexp: String(".*"),
				},
				SourceServices: &IntentionsServicesConfig{},
			},
		},
		{
			"invalid_regexp",
			true,
			&IntentionsMonitorConfig{
				SourceServices: &IntentionsServicesConfig{
					Regexp: String("*"),
				},
				DestinationServices: &IntentionsServicesConfig{
					Regexp: String(".*"),
				},
			},
		},
		{
			"invalid_empty_string_names",
			true,
			&IntentionsMonitorConfig{
				SourceServices: &IntentionsServicesConfig{
					Names: []string{"api", ""},
				},
				DestinationServices: &IntentionsServicesConfig{
					Names: []string{"", "web"},
				},
			},
		},
		{
			"invalid_both_regexp_and_names_configured",
			true,
			&IntentionsMonitorConfig{
				SourceServices: &IntentionsServicesConfig{
					Regexp: String(".*"),
					Names:  []string{"api"},
				},
				DestinationServices: &IntentionsServicesConfig{
					Regexp: String(".*"),
					Names:  []string{"web"},
				},
			},
		},
		{
			"invalid_no_regexp_no_names_configured",
			true,
			&IntentionsMonitorConfig{
				SourceServices:      &IntentionsServicesConfig{},
				DestinationServices: &IntentionsServicesConfig{},
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

func TestIntentionsMonitorConfig_GoString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		i        *IntentionsMonitorConfig
		expected string
	}{
		{
			"nil",
			nil,
			"(*IntentionsMonitorConfig)(nil)",
		},
		{
			"regexp_fully_configured",
			&IntentionsMonitorConfig{
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				SourceServices: &IntentionsServicesConfig{
					Regexp: String("^web.*"),
				},
				DestinationServices: &IntentionsServicesConfig{
					Regexp: String("^api.*"),
				},
			},
			"&IntentionsMonitorConfig{Datacenter:dc, Namespace:namespace, " +
				"Source Services: Regexp:^web.*, Destination Services: Regexp:^api.*}",
		},
		{
			"names_fully_configured",
			&IntentionsMonitorConfig{
				Datacenter: String("dc"),
				Namespace:  String("namespace"),
				SourceServices: &IntentionsServicesConfig{
					Names: []string{"servicea"},
				},
				DestinationServices: &IntentionsServicesConfig{
					Names: []string{"serviceb"},
				},
			},
			"&IntentionsMonitorConfig{Datacenter:dc, Namespace:namespace, " +
				"Source Services: Names:[servicea], Destination Services: Names:[serviceb]}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.i.GoString()
			require.Equal(t, tc.expected, actual)
		})
	}
}

// TestIntentionsMonitorConfig_RegexpNil tests the exception that when `Regexp` is
// unset, it retains nil value after Finalize() and Validate(). Tests it is
// idempotent
func TestIntentionsMonitorConfig_RegexpNil(t *testing.T) {
	t.Parallel()

	conf :=
		&IntentionsServicesConfig{
			Names: []string{"api"},
			// Regexp unset
		}

	// Confirm `Regexp` nil
	conf.Finalize()
	err := conf.Validate()
	assert.NoError(t, err)
	assert.Nil(t, conf.Regexp)

	// Confirm idempotent - Validate() doesn't error and `Regexp` still nil
	conf.Finalize()
	err = conf.Validate()
	assert.NoError(t, err)
	assert.Nil(t, conf.Regexp)
}
