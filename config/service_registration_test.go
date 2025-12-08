// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceRegistrationConfig_DefaultServiceRegistrationConfig(t *testing.T) {
	t.Parallel()
	r := DefaultServiceRegistrationConfig()
	expected := &ServiceRegistrationConfig{
		Enabled:     Bool(true),
		Namespace:   String(""),
		ServiceName: String(DefaultServiceName),
		Address:     String(""),
		DefaultCheck: &DefaultCheckConfig{
			Enabled: Bool(true),
			Address: String(""),
		},
	}
	assert.Equal(t, expected, r)
}

func TestServiceRegistrationConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &ServiceRegistrationConfig{}
	finalizedConf.Finalize()

	cases := []struct {
		name string
		a    *ServiceRegistrationConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&ServiceRegistrationConfig{},
		},
		{
			"finalized",
			finalizedConf,
		},
		{
			"fully_configured",
			&ServiceRegistrationConfig{
				Enabled:     Bool(false),
				ServiceName: String("cts-service"),
				Namespace:   String("test"),
				Address:     String("172.0.0.2"),
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

func TestServiceRegistrationConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ServiceRegistrationConfig
		b    *ServiceRegistrationConfig
		r    *ServiceRegistrationConfig
	}{
		{
			"nil_a",
			nil,
			&ServiceRegistrationConfig{},
			&ServiceRegistrationConfig{},
		},
		{
			"nil_b",
			&ServiceRegistrationConfig{},
			nil,
			&ServiceRegistrationConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&ServiceRegistrationConfig{},
			&ServiceRegistrationConfig{},
			&ServiceRegistrationConfig{},
		},
		{
			"enabled_overrides",
			&ServiceRegistrationConfig{Enabled: Bool(false)},
			&ServiceRegistrationConfig{Enabled: Bool(true)},
			&ServiceRegistrationConfig{Enabled: Bool(true)},
		},
		{
			"enabled_empty_one",
			&ServiceRegistrationConfig{Enabled: Bool(false)},
			&ServiceRegistrationConfig{},
			&ServiceRegistrationConfig{Enabled: Bool(false)},
		},
		{
			"enabled_empty_two",
			&ServiceRegistrationConfig{},
			&ServiceRegistrationConfig{Enabled: Bool(false)},
			&ServiceRegistrationConfig{Enabled: Bool(false)},
		},
		{
			"enabled_same",
			&ServiceRegistrationConfig{Enabled: Bool(false)},
			&ServiceRegistrationConfig{Enabled: Bool(false)},
			&ServiceRegistrationConfig{Enabled: Bool(false)},
		},
		{
			"namespace_overrides",
			&ServiceRegistrationConfig{Namespace: String("ns_a")},
			&ServiceRegistrationConfig{Namespace: String("ns_b")},
			&ServiceRegistrationConfig{Namespace: String("ns_b")},
		},
		{
			"namespace_empty_one",
			&ServiceRegistrationConfig{Namespace: String("ns_a")},
			&ServiceRegistrationConfig{},
			&ServiceRegistrationConfig{Namespace: String("ns_a")},
		},
		{
			"namespace_empty_two",
			&ServiceRegistrationConfig{},
			&ServiceRegistrationConfig{Namespace: String("ns_a")},
			&ServiceRegistrationConfig{Namespace: String("ns_a")},
		},
		{
			"namespace_same",
			&ServiceRegistrationConfig{Namespace: String("ns_a")},
			&ServiceRegistrationConfig{Namespace: String("ns_a")},
			&ServiceRegistrationConfig{Namespace: String("ns_a")},
		},
		{
			"service_name_overrides",
			&ServiceRegistrationConfig{ServiceName: String("service_a")},
			&ServiceRegistrationConfig{ServiceName: String("service_b")},
			&ServiceRegistrationConfig{ServiceName: String("service_b")},
		},
		{
			"service_name_empty_one",
			&ServiceRegistrationConfig{ServiceName: String("service_a")},
			&ServiceRegistrationConfig{},
			&ServiceRegistrationConfig{ServiceName: String("service_a")},
		},
		{
			"service_name_empty_two",
			&ServiceRegistrationConfig{},
			&ServiceRegistrationConfig{ServiceName: String("service_a")},
			&ServiceRegistrationConfig{ServiceName: String("service_a")},
		},
		{
			"service_name_same",
			&ServiceRegistrationConfig{ServiceName: String("service_a")},
			&ServiceRegistrationConfig{ServiceName: String("service_a")},
			&ServiceRegistrationConfig{ServiceName: String("service_a")},
		},
		{
			"address_overrides",
			&ServiceRegistrationConfig{Address: String("address_a")},
			&ServiceRegistrationConfig{Address: String("address_b")},
			&ServiceRegistrationConfig{Address: String("address_b")},
		},
		{
			"address_empty_one",
			&ServiceRegistrationConfig{Address: String("address_a")},
			&ServiceRegistrationConfig{},
			&ServiceRegistrationConfig{Address: String("address_a")},
		},
		{
			"address_empty_two",
			&ServiceRegistrationConfig{},
			&ServiceRegistrationConfig{Address: String("address_a")},
			&ServiceRegistrationConfig{Address: String("address_a")},
		},
		{
			"address_same",
			&ServiceRegistrationConfig{Address: String("address_a")},
			&ServiceRegistrationConfig{Address: String("address_a")},
			&ServiceRegistrationConfig{Address: String("address_a")},
		},
		{
			"default_check_overrides",
			&ServiceRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(true)}},
			&ServiceRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
			&ServiceRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
		},
		{
			"default_check_empty_one",
			&ServiceRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
			&ServiceRegistrationConfig{},
			&ServiceRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
		},
		{
			"default_check_empty_two",
			&ServiceRegistrationConfig{},
			&ServiceRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
			&ServiceRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
		},
		{
			"default_check_same",
			&ServiceRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
			&ServiceRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
			&ServiceRegistrationConfig{DefaultCheck: &DefaultCheckConfig{Enabled: Bool(false)}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestServiceRegistrationConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *ServiceRegistrationConfig
		r    *ServiceRegistrationConfig
	}{
		{
			"empty",
			&ServiceRegistrationConfig{},
			&ServiceRegistrationConfig{
				Enabled:     Bool(true),
				ServiceName: String(DefaultServiceName),
				Address:     String(""),
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

func TestServiceRegistrationConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		c         *ServiceRegistrationConfig
		expectErr bool
	}{
		{
			"nil",
			nil,
			false,
		},
		{
			"empty",
			&ServiceRegistrationConfig{},
			false,
		},
		{
			"configured",
			&ServiceRegistrationConfig{
				Enabled:     Bool(true),
				ServiceName: String("cts-service"),
				Address:     String("172.0.0.5"),
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
			&ServiceRegistrationConfig{
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

func TestServiceRegistrationConfig_GoString(t *testing.T) {
	cases := []struct {
		name     string
		c        *ServiceRegistrationConfig
		expected string
	}{
		{
			"nil",
			nil,
			"(*ServiceRegistrationConfig)(nil)",
		},
		{
			"fully_configured",
			&ServiceRegistrationConfig{
				Enabled:     Bool(true),
				ServiceName: String("cts-service"),
				Address:     String("172.0.0.5"),
				Namespace:   String("test"),
				DefaultCheck: &DefaultCheckConfig{
					Enabled: Bool(false),
					Address: String("test"),
				},
			},
			"&ServiceRegistrationConfig{" +
				"Enabled:true, " +
				"ServiceName:cts-service, " +
				"Address:172.0.0.5, " +
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
