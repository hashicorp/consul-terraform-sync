package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTLSConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &TLSConfig{}
	finalizedConf.Finalize()

	cases := []struct {
		name string
		a    *TLSConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&TLSConfig{},
		},
		{
			"finalized",
			finalizedConf,
		},
		{
			"same_enabled",
			&TLSConfig{
				Enabled:    Bool(true),
				Verify:     Bool(true),
				CACert:     String("ca_cert"),
				CAPath:     String("ca_path"),
				Cert:       String("cert"),
				Key:        String("key"),
				ServerName: String("server_name"),
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

func TestTLSConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *TLSConfig
		b    *TLSConfig
		r    *TLSConfig
	}{
		{
			"nil_a",
			nil,
			&TLSConfig{},
			&TLSConfig{},
		},
		{
			"nil_b",
			&TLSConfig{},
			nil,
			&TLSConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&TLSConfig{},
			&TLSConfig{},
			&TLSConfig{},
		},
		{
			"enabled_overrides",
			&TLSConfig{Enabled: Bool(true)},
			&TLSConfig{Enabled: Bool(false)},
			&TLSConfig{Enabled: Bool(false)},
		},
		{
			"enabled_empty_one",
			&TLSConfig{Enabled: Bool(true)},
			&TLSConfig{},
			&TLSConfig{Enabled: Bool(true)},
		},
		{
			"enabled_empty_two",
			&TLSConfig{},
			&TLSConfig{Enabled: Bool(true)},
			&TLSConfig{Enabled: Bool(true)},
		},
		{
			"enabled_same",
			&TLSConfig{Enabled: Bool(true)},
			&TLSConfig{Enabled: Bool(true)},
			&TLSConfig{Enabled: Bool(true)},
		},
		{
			"verify_overrides",
			&TLSConfig{Verify: Bool(true)},
			&TLSConfig{Verify: Bool(false)},
			&TLSConfig{Verify: Bool(false)},
		},
		{
			"verify_empty_one",
			&TLSConfig{Verify: Bool(true)},
			&TLSConfig{},
			&TLSConfig{Verify: Bool(true)},
		},
		{
			"verify_empty_two",
			&TLSConfig{},
			&TLSConfig{Verify: Bool(true)},
			&TLSConfig{Verify: Bool(true)},
		},
		{
			"verify_same",
			&TLSConfig{Verify: Bool(true)},
			&TLSConfig{Verify: Bool(true)},
			&TLSConfig{Verify: Bool(true)},
		},
		{
			"cert_overrides",
			&TLSConfig{Cert: String("cert")},
			&TLSConfig{Cert: String("")},
			&TLSConfig{Cert: String("")},
		},
		{
			"cert_empty_one",
			&TLSConfig{Cert: String("cert")},
			&TLSConfig{},
			&TLSConfig{Cert: String("cert")},
		},
		{
			"cert_empty_two",
			&TLSConfig{},
			&TLSConfig{Cert: String("cert")},
			&TLSConfig{Cert: String("cert")},
		},
		{
			"cert_same",
			&TLSConfig{Cert: String("cert")},
			&TLSConfig{Cert: String("cert")},
			&TLSConfig{Cert: String("cert")},
		},
		{
			"key_overrides",
			&TLSConfig{Key: String("key")},
			&TLSConfig{Key: String("")},
			&TLSConfig{Key: String("")},
		},
		{
			"key_empty_one",
			&TLSConfig{Key: String("key")},
			&TLSConfig{},
			&TLSConfig{Key: String("key")},
		},
		{
			"key_empty_two",
			&TLSConfig{},
			&TLSConfig{Key: String("key")},
			&TLSConfig{Key: String("key")},
		},
		{
			"key_same",
			&TLSConfig{Key: String("key")},
			&TLSConfig{Key: String("key")},
			&TLSConfig{Key: String("key")},
		},
		{
			"ca_cert_overrides",
			&TLSConfig{CACert: String("ca_cert")},
			&TLSConfig{CACert: String("")},
			&TLSConfig{CACert: String("")},
		},
		{
			"ca_cert_empty_one",
			&TLSConfig{CACert: String("ca_cert")},
			&TLSConfig{},
			&TLSConfig{CACert: String("ca_cert")},
		},
		{
			"ca_cert_empty_two",
			&TLSConfig{},
			&TLSConfig{CACert: String("ca_cert")},
			&TLSConfig{CACert: String("ca_cert")},
		},
		{
			"ca_cert_same",
			&TLSConfig{CACert: String("ca_cert")},
			&TLSConfig{CACert: String("ca_cert")},
			&TLSConfig{CACert: String("ca_cert")},
		},
		{
			"ca_path_overrides",
			&TLSConfig{CAPath: String("ca_path")},
			&TLSConfig{CAPath: String("")},
			&TLSConfig{CAPath: String("")},
		},
		{
			"ca_path_empty_one",
			&TLSConfig{CAPath: String("ca_path")},
			&TLSConfig{},
			&TLSConfig{CAPath: String("ca_path")},
		},
		{
			"ca_path_empty_two",
			&TLSConfig{},
			&TLSConfig{CAPath: String("ca_path")},
			&TLSConfig{CAPath: String("ca_path")},
		},
		{
			"ca_path_same",
			&TLSConfig{CAPath: String("ca_path")},
			&TLSConfig{CAPath: String("ca_path")},
			&TLSConfig{CAPath: String("ca_path")},
		},
		{
			"server_name_overrides",
			&TLSConfig{ServerName: String("server_name")},
			&TLSConfig{ServerName: String("")},
			&TLSConfig{ServerName: String("")},
		},
		{
			"server_name_empty_one",
			&TLSConfig{ServerName: String("server_name")},
			&TLSConfig{},
			&TLSConfig{ServerName: String("server_name")},
		},
		{
			"server_name_empty_two",
			&TLSConfig{},
			&TLSConfig{ServerName: String("server_name")},
			&TLSConfig{ServerName: String("server_name")},
		},
		{
			"server_name_same",
			&TLSConfig{ServerName: String("server_name")},
			&TLSConfig{ServerName: String("server_name")},
			&TLSConfig{ServerName: String("server_name")},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestTLSConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *TLSConfig
		r    *TLSConfig
	}{
		{
			"empty",
			&TLSConfig{},
			&TLSConfig{
				Enabled:    Bool(false),
				Cert:       String(""),
				CACert:     String(""),
				CAPath:     String(""),
				Key:        String(""),
				ServerName: String(""),
				Verify:     Bool(true),
			},
		},
		{
			"with_cert",
			&TLSConfig{
				Cert: String("cert"),
			},
			&TLSConfig{
				Enabled:    Bool(true),
				Cert:       String("cert"),
				CACert:     String(""),
				CAPath:     String(""),
				Key:        String(""),
				ServerName: String(""),
				Verify:     Bool(true),
			},
		},
		{
			"with_ca_cert",
			&TLSConfig{
				CACert: String("ca_cert"),
			},
			&TLSConfig{
				Enabled:    Bool(true),
				Cert:       String(""),
				CACert:     String("ca_cert"),
				CAPath:     String(""),
				Key:        String(""),
				ServerName: String(""),
				Verify:     Bool(true),
			},
		},
		{
			"with_ca_path",
			&TLSConfig{
				CAPath: String("ca_path"),
			},
			&TLSConfig{
				Enabled:    Bool(true),
				Cert:       String(""),
				CACert:     String(""),
				CAPath:     String("ca_path"),
				Key:        String(""),
				ServerName: String(""),
				Verify:     Bool(true),
			},
		},
		{
			"with_key",
			&TLSConfig{
				Key: String("key"),
			},
			&TLSConfig{
				Enabled:    Bool(true),
				Cert:       String(""),
				CACert:     String(""),
				CAPath:     String(""),
				Key:        String("key"),
				ServerName: String(""),
				Verify:     Bool(true),
			},
		},
		{
			"with_server_name",
			&TLSConfig{
				ServerName: String("server_name"),
			},
			&TLSConfig{
				Enabled:    Bool(true),
				Cert:       String(""),
				CACert:     String(""),
				CAPath:     String(""),
				Key:        String(""),
				ServerName: String("server_name"),
				Verify:     Bool(true),
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
