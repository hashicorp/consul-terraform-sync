package config

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVaultConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &VaultConfig{}
	finalizedConf.Finalize()

	cases := []struct {
		name string
		a    *VaultConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&VaultConfig{},
		},
		{
			"finalized",
			finalizedConf,
		},
		{
			"same_enabled",
			&VaultConfig{
				Address:    String("address"),
				Enabled:    Bool(true),
				Namespace:  String("foo"),
				RenewToken: Bool(true),
				TLS:        &TLSConfig{Enabled: Bool(true)},
				Token:      String("token"),
				Transport: &TransportConfig{
					DialKeepAlive: TimeDuration(20 * time.Second),
				},
				UnwrapToken:         Bool(true),
				VaultAgentTokenFile: String("/tmp/vault/agent/token"),
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

func TestVaultConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *VaultConfig
		b    *VaultConfig
		r    *VaultConfig
	}{
		{
			"nil_a",
			nil,
			&VaultConfig{},
			&VaultConfig{},
		},
		{
			"nil_b",
			&VaultConfig{},
			nil,
			&VaultConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&VaultConfig{},
			&VaultConfig{},
			&VaultConfig{},
		},
		{
			"enabled_overrides",
			&VaultConfig{Enabled: Bool(true)},
			&VaultConfig{Enabled: Bool(false)},
			&VaultConfig{Enabled: Bool(false)},
		},
		{
			"enabled_empty_one",
			&VaultConfig{Enabled: Bool(true)},
			&VaultConfig{},
			&VaultConfig{Enabled: Bool(true)},
		},
		{
			"enabled_empty_two",
			&VaultConfig{},
			&VaultConfig{Enabled: Bool(true)},
			&VaultConfig{Enabled: Bool(true)},
		},
		{
			"enabled_same",
			&VaultConfig{Enabled: Bool(true)},
			&VaultConfig{Enabled: Bool(true)},
			&VaultConfig{Enabled: Bool(true)},
		},
		{
			"address_overrides",
			&VaultConfig{Address: String("address")},
			&VaultConfig{Address: String("")},
			&VaultConfig{Address: String("")},
		},
		{
			"address_empty_one",
			&VaultConfig{Address: String("address")},
			&VaultConfig{},
			&VaultConfig{Address: String("address")},
		},
		{
			"address_empty_two",
			&VaultConfig{},
			&VaultConfig{Address: String("address")},
			&VaultConfig{Address: String("address")},
		},
		{
			"address_same",
			&VaultConfig{Address: String("address")},
			&VaultConfig{Address: String("address")},
			&VaultConfig{Address: String("address")},
		},
		{
			"namespace_overrides",
			&VaultConfig{Namespace: String("foo")},
			&VaultConfig{Namespace: String("bar")},
			&VaultConfig{Namespace: String("bar")},
		},
		{
			"namespace_empty_one",
			&VaultConfig{Namespace: String("foo")},
			&VaultConfig{},
			&VaultConfig{Namespace: String("foo")},
		},
		{
			"namespace_empty_two",
			&VaultConfig{},
			&VaultConfig{Namespace: String("bar")},
			&VaultConfig{Namespace: String("bar")},
		},
		{
			"namespace_same",
			&VaultConfig{Namespace: String("foo")},
			&VaultConfig{Namespace: String("foo")},
			&VaultConfig{Namespace: String("foo")},
		},
		{
			"token_overrides",
			&VaultConfig{Token: String("token")},
			&VaultConfig{Token: String("")},
			&VaultConfig{Token: String("")},
		},
		{
			"token_empty_one",
			&VaultConfig{Token: String("token")},
			&VaultConfig{},
			&VaultConfig{Token: String("token")},
		},
		{
			"token_empty_two",
			&VaultConfig{},
			&VaultConfig{Token: String("token")},
			&VaultConfig{Token: String("token")},
		},
		{
			"token_same",
			&VaultConfig{Token: String("token")},
			&VaultConfig{Token: String("token")},
			&VaultConfig{Token: String("token")},
		},
		{
			"unwrap_token_overrides",
			&VaultConfig{UnwrapToken: Bool(true)},
			&VaultConfig{UnwrapToken: Bool(false)},
			&VaultConfig{UnwrapToken: Bool(false)},
		},
		{
			"unwrap_token_empty_one",
			&VaultConfig{UnwrapToken: Bool(true)},
			&VaultConfig{},
			&VaultConfig{UnwrapToken: Bool(true)},
		},
		{
			"unwrap_token_empty_two",
			&VaultConfig{},
			&VaultConfig{UnwrapToken: Bool(true)},
			&VaultConfig{UnwrapToken: Bool(true)},
		},
		{
			"unwrap_token_same",
			&VaultConfig{UnwrapToken: Bool(true)},
			&VaultConfig{UnwrapToken: Bool(true)},
			&VaultConfig{UnwrapToken: Bool(true)},
		},
		{
			"renew_token_overrides",
			&VaultConfig{RenewToken: Bool(true)},
			&VaultConfig{RenewToken: Bool(false)},
			&VaultConfig{RenewToken: Bool(false)},
		},
		{
			"renew_token_empty_one",
			&VaultConfig{RenewToken: Bool(true)},
			&VaultConfig{},
			&VaultConfig{RenewToken: Bool(true)},
		},
		{
			"renew_token_empty_two",
			&VaultConfig{},
			&VaultConfig{RenewToken: Bool(true)},
			&VaultConfig{RenewToken: Bool(true)},
		},
		{
			"renew_token_same",
			&VaultConfig{RenewToken: Bool(true)},
			&VaultConfig{RenewToken: Bool(true)},
			&VaultConfig{RenewToken: Bool(true)},
		},
		{
			"TLS_overrides",
			&VaultConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
			&VaultConfig{TLS: &TLSConfig{Enabled: Bool(false)}},
			&VaultConfig{TLS: &TLSConfig{Enabled: Bool(false)}},
		},
		{
			"TLS_empty_one",
			&VaultConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
			&VaultConfig{},
			&VaultConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
		},
		{
			"TLS_empty_two",
			&VaultConfig{},
			&VaultConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
			&VaultConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
		},
		{
			"TLS_same",
			&VaultConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
			&VaultConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
			&VaultConfig{TLS: &TLSConfig{Enabled: Bool(true)}},
		},
		{
			"transport_overrides",
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(20 * time.Second)}},
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(20 * time.Second)}},
		},
		{
			"transport_empty_one",
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
			&VaultConfig{},
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
		},
		{
			"transport_empty_two",
			&VaultConfig{},
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
		},
		{
			"transport_same",
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
			&VaultConfig{Transport: &TransportConfig{DialKeepAlive: TimeDuration(10 * time.Second)}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestVaultConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *VaultConfig
		r    *VaultConfig
	}{
		{
			"empty",
			&VaultConfig{},
			&VaultConfig{
				Address:    String(""),
				Enabled:    Bool(false),
				Namespace:  String(""),
				RenewToken: Bool(false),
				TLS: &TLSConfig{
					CACert:     String(""),
					CAPath:     String(""),
					Cert:       String(""),
					Enabled:    Bool(true),
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
				UnwrapToken: Bool(DefaultVaultUnwrapToken),
			},
		},
		{
			"with_address",
			&VaultConfig{
				Address: String("address"),
			},
			&VaultConfig{
				Address:    String("address"),
				Enabled:    Bool(true),
				Namespace:  String(""),
				RenewToken: Bool(false),
				TLS: &TLSConfig{
					CACert:     String(""),
					CAPath:     String(""),
					Cert:       String(""),
					Enabled:    Bool(true),
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
				UnwrapToken: Bool(DefaultVaultUnwrapToken),
			},
		},
		{
			"with_TLS_config",
			&VaultConfig{
				Address: String("address"),
			},
			&VaultConfig{
				Address:    String("address"),
				Enabled:    Bool(true),
				Namespace:  String(""),
				RenewToken: Bool(false),
				TLS: &TLSConfig{
					CACert:     String(""),
					CAPath:     String(""),
					Cert:       String(""),
					Enabled:    Bool(true),
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
				UnwrapToken: Bool(DefaultVaultUnwrapToken),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.r.test = true
			tc.i.test = true
			tc.i.Finalize()
			assert.Equal(t, tc.r, tc.i)
		})
	}
}

func TestVaultConfig_TokenRenew(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		act    *VaultConfig
		exp    *VaultConfig
		fields []string
	}{
		{
			"base_renew",
			&VaultConfig{test: true},
			&VaultConfig{
				RenewToken: Bool(false),
			},
			[]string{"RenewToken"},
		},
		{
			"base_renew_w_token",
			&VaultConfig{
				Token: String("a-token"),
			},
			&VaultConfig{
				RenewToken: Bool(true),
			},
			[]string{"RenewToken"},
		},
		{
			"token_file_w_no_renew",
			&VaultConfig{
				VaultAgentTokenFile: String("foo"),
			},
			&VaultConfig{
				VaultAgentTokenFile: String("foo"),
				RenewToken:          Bool(false),
			},
			[]string{"RenewToken"},
		},
		{
			"token_file_w_renew",
			&VaultConfig{
				VaultAgentTokenFile: String("foo"),
				RenewToken:          Bool(true),
			},
			&VaultConfig{
				VaultAgentTokenFile: String("foo"),
				RenewToken:          Bool(true),
			},
			[]string{"RenewToken"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			tc.act.Finalize()
			for _, f := range tc.fields {
				av := reflect.Indirect(reflect.ValueOf(*tc.act).FieldByName(f))
				ev := reflect.Indirect(reflect.ValueOf(*tc.exp).FieldByName(f))
				switch av.Kind() {
				case reflect.Bool:
					if ev.Bool() != av.Bool() {
						t.Errorf("\nfield:%s\nexp: %#v\nact: %#v", f, ev, av)
					}
				}
			}
		})
	}
}
