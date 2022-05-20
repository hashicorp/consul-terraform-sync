package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsulConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &ConsulConfig{}
	finalizedConf.Finalize()

	cases := []struct {
		name string
		a    *ConsulConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&ConsulConfig{},
		},
		{
			"finalized",
			finalizedConf,
		},
		{
			"same_enabled",
			&ConsulConfig{
				Address:     String("1.2.3.4"),
				Auth:        &AuthConfig{Enabled: Bool(true)},
				KVPath:      String("consul-terraform-sync/"),
				KVNamespace: String("org"),
				TLS:         &TLSConfig{Enabled: Bool(true)},
				Token:       String("abcd1234"),
				ServiceRegistration: &ServiceRegistrationConfig{
					Enabled:     Bool(false),
					ServiceName: String("test-service"),
					Address:     String("10.2.3.4"),
					Namespace:   String("test-ns"),
					DefaultCheck: &DefaultCheckConfig{
						Enabled: Bool(true),
						Address: String("test"),
					},
				},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Copy()
			assert.Equal(t, tc.a, r)
		})
	}
}

func TestConsulConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ConsulConfig
		b    *ConsulConfig
		r    *ConsulConfig
	}{
		{
			"nil_a",
			nil,
			&ConsulConfig{},
			&ConsulConfig{},
		},
		{
			"nil_b",
			&ConsulConfig{},
			nil,
			&ConsulConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&ConsulConfig{},
			&ConsulConfig{},
			&ConsulConfig{},
		},
		{
			"address_overrides",
			&ConsulConfig{Address: String("same")},
			&ConsulConfig{Address: String("different")},
			&ConsulConfig{Address: String("different")},
		},
		{
			"address_empty_one",
			&ConsulConfig{Address: String("same")},
			&ConsulConfig{},
			&ConsulConfig{Address: String("same")},
		},
		{
			"address_empty_two",
			&ConsulConfig{},
			&ConsulConfig{Address: String("same")},
			&ConsulConfig{Address: String("same")},
		},
		{
			"address_same",
			&ConsulConfig{Address: String("same")},
			&ConsulConfig{Address: String("same")},
			&ConsulConfig{Address: String("same")},
		},
		{
			"auth_overrides",
			&ConsulConfig{Auth: &AuthConfig{Enabled: Bool(true)}},
			&ConsulConfig{Auth: &AuthConfig{Enabled: Bool(false)}},
			&ConsulConfig{Auth: &AuthConfig{Enabled: Bool(false)}},
		},
		{
			"auth_empty_one",
			&ConsulConfig{Auth: &AuthConfig{Enabled: Bool(true)}},
			&ConsulConfig{},
			&ConsulConfig{Auth: &AuthConfig{Enabled: Bool(true)}},
		},
		{
			"auth_empty_two",
			&ConsulConfig{},
			&ConsulConfig{Auth: &AuthConfig{Enabled: Bool(true)}},
			&ConsulConfig{Auth: &AuthConfig{Enabled: Bool(true)}},
		},
		{
			"auth_same",
			&ConsulConfig{Auth: &AuthConfig{Enabled: Bool(true)}},
			&ConsulConfig{Auth: &AuthConfig{Enabled: Bool(true)}},
			&ConsulConfig{Auth: &AuthConfig{Enabled: Bool(true)}},
		},
		{
			"tls_overrides",
			&ConsulConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
			&ConsulConfig{TLS: &TLSConfig{Enabled: Bool(false)}},
			&ConsulConfig{TLS: &TLSConfig{Enabled: Bool(false)}},
		},
		{
			"tls_empty_one",
			&ConsulConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
			&ConsulConfig{},
			&ConsulConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
		},
		{
			"tls_empty_two",
			&ConsulConfig{},
			&ConsulConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
			&ConsulConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
		},
		{
			"tls_same",
			&ConsulConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
			&ConsulConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
			&ConsulConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
		},
		{
			"token_overrides",
			&ConsulConfig{Token: String("same")},
			&ConsulConfig{Token: String("different")},
			&ConsulConfig{Token: String("different")},
		},
		{
			"token_empty_one",
			&ConsulConfig{Token: String("same")},
			&ConsulConfig{},
			&ConsulConfig{Token: String("same")},
		},
		{
			"token_empty_two",
			&ConsulConfig{},
			&ConsulConfig{Token: String("same")},
			&ConsulConfig{Token: String("same")},
		},
		{
			"token_same",
			&ConsulConfig{Token: String("same")},
			&ConsulConfig{Token: String("same")},
			&ConsulConfig{Token: String("same")},
		},
		{
			"transport_overrides",
			&ConsulConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
			&ConsulConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(20 * time.Second)}},
			&ConsulConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(20 * time.Second)}},
		},
		{
			"transport_empty_one",
			&ConsulConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
			&ConsulConfig{},
			&ConsulConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
		},
		{
			"transport_empty_two",
			&ConsulConfig{},
			&ConsulConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
			&ConsulConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
		},
		{
			"transport_same",
			&ConsulConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
			&ConsulConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
			&ConsulConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
		},
		{
			"service_registration_overrides",
			&ConsulConfig{ServiceRegistration: &ServiceRegistrationConfig{Enabled: Bool(true)}},
			&ConsulConfig{ServiceRegistration: &ServiceRegistrationConfig{Enabled: Bool(false)}},
			&ConsulConfig{ServiceRegistration: &ServiceRegistrationConfig{Enabled: Bool(false)}},
		},
		{
			"service_registration_empty_one",
			&ConsulConfig{ServiceRegistration: &ServiceRegistrationConfig{Enabled: Bool(true)}},
			&ConsulConfig{},
			&ConsulConfig{ServiceRegistration: &ServiceRegistrationConfig{Enabled: Bool(true)}},
		},
		{
			"service_registration_empty_two",
			&ConsulConfig{},
			&ConsulConfig{ServiceRegistration: &ServiceRegistrationConfig{Enabled: Bool(true)}},
			&ConsulConfig{ServiceRegistration: &ServiceRegistrationConfig{Enabled: Bool(true)}},
		},
		{
			"service_registration_same",
			&ConsulConfig{ServiceRegistration: &ServiceRegistrationConfig{Enabled: Bool(true)}},
			&ConsulConfig{ServiceRegistration: &ServiceRegistrationConfig{Enabled: Bool(true)}},
			&ConsulConfig{ServiceRegistration: &ServiceRegistrationConfig{Enabled: Bool(true)}},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestConsulConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *ConsulConfig
		r    *ConsulConfig
	}{
		{
			"empty",
			&ConsulConfig{},
			&ConsulConfig{
				Address: String("localhost:8500"),
				Auth: &AuthConfig{
					Enabled:  Bool(false),
					Username: String(""),
					Password: String(""),
				},
				KVNamespace: String(""),
				KVPath:      String(DefaultConsulKVPath),
				TLS: &TLSConfig{
					CACert:     String(""),
					CAPath:     String(""),
					Cert:       String(""),
					Enabled:    Bool(false),
					Key:        String(""),
					ServerName: String(""),
					Verify:     Bool(true),
				},
				Token: String(""),
				Transport: &TransportConfig{
					DialKeepAlive:       TimeDuration(DefaultDialKeepAlive),
					DialTimeout:         TimeDuration(DefaultDialTimeout),
					DisableKeepAlives:   Bool(false),
					IdleConnTimeout:     TimeDuration(DefaultIdleConnTimeout),
					MaxIdleConns:        Int(DefaultMaxIdleConns),
					MaxIdleConnsPerHost: Int(DefaultMaxIdleConnsPerHost),
					TLSHandshakeTimeout: TimeDuration(DefaultTLSHandshakeTimeout),
				},
				ServiceRegistration: &ServiceRegistrationConfig{
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
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			tc.i.Finalize()
			assert.Equal(t, tc.r, tc.i)
		})
	}
}

func TestConsulConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		c         *ConsulConfig
		expectErr bool
	}{
		{
			"nil",
			nil,
			false,
		},
		{
			"empty",
			&ConsulConfig{},
			false,
		},
		{
			"invalid",
			&ConsulConfig{
				ServiceRegistration: &ServiceRegistrationConfig{
					DefaultCheck: &DefaultCheckConfig{
						Address: String("172.0.0.8:5000"),
					},
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
