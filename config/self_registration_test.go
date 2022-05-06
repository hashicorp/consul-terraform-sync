package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelfRegistrationConfig_DefaultSelfRegistrationConfig(t *testing.T) {
	t.Parallel()
	r := DefaultSelfRegistrationConfig()
	expected := &SelfRegistrationConfig{
		Enabled:     Bool(true),
		Namespace:   String(""),
		ServiceName: String(DefaultServiceName),
		DefaultCheck: &DefaultCheckConfig{
			Enabled: Bool(true),
			Address: String(""),
		},
	}
	assert.Equal(t, expected, r)
}

func TestSelfRegistrationConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &SelfRegistrationConfig{}
	finalizedConf.Finalize()

	cases := []struct {
		name string
		a    *SelfRegistrationConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&SelfRegistrationConfig{},
		},
		{
			"finalized",
			finalizedConf,
		},
		{
			"fully_configured",
			&SelfRegistrationConfig{
				Enabled:     Bool(false),
				ServiceName: String("cts-service"),
				Namespace:   String("test"),
				DefaultCheck: &DefaultCheckConfig{
					Enabled: Bool(false),
					Address: String("test"),
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.a.Copy()
			assert.Equal(t, tc.a, r)
		})
	}
}

func TestSelfRegistrationConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *SelfRegistrationConfig
		b    *SelfRegistrationConfig
		r    *SelfRegistrationConfig
	}{
		{
			"nil_a",
			nil,
			&SelfRegistrationConfig{},
			&SelfRegistrationConfig{},
		},
		{
			"nil_b",
			&SelfRegistrationConfig{},
			nil,
			&SelfRegistrationConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&SelfRegistrationConfig{},
			&SelfRegistrationConfig{},
			&SelfRegistrationConfig{},
		},
		{
			"enabled_overrides",
			&SelfRegistrationConfig{Enabled: Bool(false)},
			&SelfRegistrationConfig{Enabled: Bool(true)},
			&SelfRegistrationConfig{Enabled: Bool(true)},
		},
		{
			"enabled_empty_one",
			&SelfRegistrationConfig{Enabled: Bool(false)},
			&SelfRegistrationConfig{},
			&SelfRegistrationConfig{Enabled: Bool(false)},
		},
		{
			"enabled_empty_two",
			&SelfRegistrationConfig{},
			&SelfRegistrationConfig{Enabled: Bool(false)},
			&SelfRegistrationConfig{Enabled: Bool(false)},
		},
		{
			"enabled_same",
			&SelfRegistrationConfig{Enabled: Bool(false)},
			&SelfRegistrationConfig{Enabled: Bool(false)},
			&SelfRegistrationConfig{Enabled: Bool(false)},
		},
		{
			"namespace_overrides",
			&SelfRegistrationConfig{Namespace: String("ns_a")},
			&SelfRegistrationConfig{Namespace: String("ns_b")},
			&SelfRegistrationConfig{Namespace: String("ns_b")},
		},
		{
			"namespace_empty_one",
			&SelfRegistrationConfig{Namespace: String("ns_a")},
			&SelfRegistrationConfig{},
			&SelfRegistrationConfig{Namespace: String("ns_a")},
		},
		{
			"namespace_empty_two",
			&SelfRegistrationConfig{},
			&SelfRegistrationConfig{Namespace: String("ns_a")},
			&SelfRegistrationConfig{Namespace: String("ns_a")},
		},
		{
			"namespace_same",
			&SelfRegistrationConfig{Namespace: String("ns_a")},
			&SelfRegistrationConfig{Namespace: String("ns_a")},
			&SelfRegistrationConfig{Namespace: String("ns_a")},
		},
		{
			"service_name_overrides",
			&SelfRegistrationConfig{ServiceName: String("service_a")},
			&SelfRegistrationConfig{ServiceName: String("service_b")},
			&SelfRegistrationConfig{ServiceName: String("service_b")},
		},
		{
			"service_name_empty_one",
			&SelfRegistrationConfig{ServiceName: String("service_a")},
			&SelfRegistrationConfig{},
			&SelfRegistrationConfig{ServiceName: String("service_a")},
		},
		{
			"service_name_empty_two",
			&SelfRegistrationConfig{},
			&SelfRegistrationConfig{ServiceName: String("service_a")},
			&SelfRegistrationConfig{ServiceName: String("service_a")},
		},
		{
			"service_name_same",
			&SelfRegistrationConfig{ServiceName: String("service_a")},
			&SelfRegistrationConfig{ServiceName: String("service_a")},
			&SelfRegistrationConfig{ServiceName: String("service_a")},
		},
		{
			"default_check_overrides",
			&SelfRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(true)}},
			&SelfRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
			&SelfRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
		},
		{
			"default_check_empty_one",
			&SelfRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
			&SelfRegistrationConfig{},
			&SelfRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
		},
		{
			"default_check_empty_two",
			&SelfRegistrationConfig{},
			&SelfRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
			&SelfRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
		},
		{
			"default_check_same",
			&SelfRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
			&SelfRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
			&SelfRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestSelfRegistrationConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *SelfRegistrationConfig
		r    *SelfRegistrationConfig
	}{
		{
			"empty",
			&SelfRegistrationConfig{},
			&SelfRegistrationConfig{
				Enabled:     Bool(true),
				ServiceName: String(DefaultServiceName),
				Namespace:   String(""),
				DefaultCheck: &DefaultCheckConfig{
					Enabled: Bool(true),
					Address: String(""),
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

func TestSelfRegistrationConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		c         *SelfRegistrationConfig
		expectErr bool
	}{
		{
			"nil",
			nil,
			false,
		},
		{
			"empty",
			&SelfRegistrationConfig{},
			false,
		},
		{
			"configured",
			&SelfRegistrationConfig{
				Enabled:     Bool(true),
				ServiceName: String("cts-service"),
				Namespace:   String("ns-1"),
				DefaultCheck: &DefaultCheckConfig{
					Enabled: Bool(true),
					Address: String("http://172.0.0.8:5000"),
				},
			},
			false,
		},
		{
			"invalid",
			&SelfRegistrationConfig{
				DefaultCheck: &DefaultCheckConfig{
					Address: String("172.0.0.8:5000"),
				},
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.c.Validate()
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSelfRegistrationConfig_GoString(t *testing.T) {
	cases := []struct {
		name     string
		c        *SelfRegistrationConfig
		expected string
	}{
		{
			"nil",
			nil,
			"(*SelfRegistrationConfig)(nil)",
		},
		{
			"fully_configured",
			&SelfRegistrationConfig{
				Enabled:     Bool(true),
				ServiceName: String("cts-service"),
				Namespace:   String("test"),
				DefaultCheck: &DefaultCheckConfig{
					Enabled: Bool(false),
					Address: String("test"),
				},
			},
			"&SelfRegistrationConfig{" +
				"Enabled:true, " +
				"ServiceName:cts-service, " +
				"Namespace:test, " +
				"DefaultCheck: &DefaultCheckConfig{Enabled:false, Address:test}" +
				"}",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.c.GoString()
			assert.Equal(t, tc.expected, r)
		})
	}
}
