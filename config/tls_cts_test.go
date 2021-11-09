package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCTSTLSConfig_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *CTSTLSConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&CTSTLSConfig{},
		},
		{
			"same_enabled",
			&CTSTLSConfig{
				Enabled:        Bool(true),
				VerifyIncoming: Bool(true),
				CACert:         String("ca_cert"),
				CAPath:         String("ca_path"),
				Cert:           String("cert"),
				Key:            String("key"),
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

func TestCTSTLSConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *CTSTLSConfig
		b    *CTSTLSConfig
		r    *CTSTLSConfig
	}{
		{
			"nil_a",
			nil,
			&CTSTLSConfig{},
			&CTSTLSConfig{},
		},
		{
			"nil_b",
			&CTSTLSConfig{},
			nil,
			&CTSTLSConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&CTSTLSConfig{},
			&CTSTLSConfig{},
			&CTSTLSConfig{},
		},
		{
			"enabled_overrides",
			&CTSTLSConfig{Enabled: Bool(true)},
			&CTSTLSConfig{Enabled: Bool(false)},
			&CTSTLSConfig{Enabled: Bool(false)},
		},
		{
			"enabled_empty_one",
			&CTSTLSConfig{Enabled: Bool(true)},
			&CTSTLSConfig{},
			&CTSTLSConfig{Enabled: Bool(true)},
		},
		{
			"enabled_empty_two",
			&CTSTLSConfig{},
			&CTSTLSConfig{Enabled: Bool(true)},
			&CTSTLSConfig{Enabled: Bool(true)},
		},
		{
			"enabled_same",
			&CTSTLSConfig{Enabled: Bool(true)},
			&CTSTLSConfig{Enabled: Bool(true)},
			&CTSTLSConfig{Enabled: Bool(true)},
		},
		{
			"verify_incoming_overrides",
			&CTSTLSConfig{VerifyIncoming: Bool(true)},
			&CTSTLSConfig{VerifyIncoming: Bool(false)},
			&CTSTLSConfig{VerifyIncoming: Bool(false)},
		},
		{
			"verify_incoming_empty_one",
			&CTSTLSConfig{VerifyIncoming: Bool(true)},
			&CTSTLSConfig{},
			&CTSTLSConfig{VerifyIncoming: Bool(true)},
		},
		{
			"verify_incoming_empty_two",
			&CTSTLSConfig{},
			&CTSTLSConfig{VerifyIncoming: Bool(true)},
			&CTSTLSConfig{VerifyIncoming: Bool(true)},
		},
		{
			"verify_incoming_same",
			&CTSTLSConfig{VerifyIncoming: Bool(true)},
			&CTSTLSConfig{VerifyIncoming: Bool(true)},
			&CTSTLSConfig{VerifyIncoming: Bool(true)},
		},
		{
			"cert_overrides",
			&CTSTLSConfig{Cert: String("cert")},
			&CTSTLSConfig{Cert: String("")},
			&CTSTLSConfig{Cert: String("")},
		},
		{
			"cert_empty_one",
			&CTSTLSConfig{Cert: String("cert")},
			&CTSTLSConfig{},
			&CTSTLSConfig{Cert: String("cert")},
		},
		{
			"cert_empty_two",
			&CTSTLSConfig{},
			&CTSTLSConfig{Cert: String("cert")},
			&CTSTLSConfig{Cert: String("cert")},
		},
		{
			"cert_same",
			&CTSTLSConfig{Cert: String("cert")},
			&CTSTLSConfig{Cert: String("cert")},
			&CTSTLSConfig{Cert: String("cert")},
		},
		{
			"key_overrides",
			&CTSTLSConfig{Key: String("key")},
			&CTSTLSConfig{Key: String("")},
			&CTSTLSConfig{Key: String("")},
		},
		{
			"key_empty_one",
			&CTSTLSConfig{Key: String("key")},
			&CTSTLSConfig{},
			&CTSTLSConfig{Key: String("key")},
		},
		{
			"key_empty_two",
			&CTSTLSConfig{},
			&CTSTLSConfig{Key: String("key")},
			&CTSTLSConfig{Key: String("key")},
		},
		{
			"key_same",
			&CTSTLSConfig{Key: String("key")},
			&CTSTLSConfig{Key: String("key")},
			&CTSTLSConfig{Key: String("key")},
		},
		{
			"ca_cert_overrides",
			&CTSTLSConfig{CACert: String("ca_cert")},
			&CTSTLSConfig{CACert: String("")},
			&CTSTLSConfig{CACert: String("")},
		},
		{
			"ca_cert_empty_one",
			&CTSTLSConfig{CACert: String("ca_cert")},
			&CTSTLSConfig{},
			&CTSTLSConfig{CACert: String("ca_cert")},
		},
		{
			"ca_cert_empty_two",
			&CTSTLSConfig{},
			&CTSTLSConfig{CACert: String("ca_cert")},
			&CTSTLSConfig{CACert: String("ca_cert")},
		},
		{
			"ca_cert_same",
			&CTSTLSConfig{CACert: String("ca_cert")},
			&CTSTLSConfig{CACert: String("ca_cert")},
			&CTSTLSConfig{CACert: String("ca_cert")},
		},
		{
			"ca_path_overrides",
			&CTSTLSConfig{CAPath: String("ca_path")},
			&CTSTLSConfig{CAPath: String("")},
			&CTSTLSConfig{CAPath: String("")},
		},
		{
			"ca_path_empty_one",
			&CTSTLSConfig{CAPath: String("ca_path")},
			&CTSTLSConfig{},
			&CTSTLSConfig{CAPath: String("ca_path")},
		},
		{
			"ca_path_empty_two",
			&CTSTLSConfig{},
			&CTSTLSConfig{CAPath: String("ca_path")},
			&CTSTLSConfig{CAPath: String("ca_path")},
		},
		{
			"ca_path_same",
			&CTSTLSConfig{CAPath: String("ca_path")},
			&CTSTLSConfig{CAPath: String("ca_path")},
			&CTSTLSConfig{CAPath: String("ca_path")},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestCTSTLSConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *CTSTLSConfig
		r    *CTSTLSConfig
	}{
		{
			"empty",
			&CTSTLSConfig{},
			&CTSTLSConfig{
				Enabled:        Bool(false),
				Cert:           String(""),
				CACert:         String(""),
				CAPath:         String(""),
				Key:            String(""),
				VerifyIncoming: Bool(false),
			},
		},
		{
			"with_cert",
			&CTSTLSConfig{
				Cert: String("cert"),
			},
			&CTSTLSConfig{
				Enabled:        Bool(true),
				Cert:           String("cert"),
				CACert:         String(""),
				CAPath:         String(""),
				Key:            String(""),
				VerifyIncoming: Bool(false),
			},
		},
		{
			"with_ca_cert",
			&CTSTLSConfig{
				CACert: String("ca_cert"),
			},
			&CTSTLSConfig{
				Enabled:        Bool(true),
				Cert:           String(""),
				CACert:         String("ca_cert"),
				CAPath:         String(""),
				Key:            String(""),
				VerifyIncoming: Bool(false),
			},
		},
		{
			"with_ca_path",
			&CTSTLSConfig{
				CAPath: String("ca_path"),
			},
			&CTSTLSConfig{
				Enabled:        Bool(true),
				Cert:           String(""),
				CACert:         String(""),
				CAPath:         String("ca_path"),
				Key:            String(""),
				VerifyIncoming: Bool(false),
			},
		},
		{
			"with_key",
			&CTSTLSConfig{
				Key: String("key"),
			},
			&CTSTLSConfig{
				Enabled:        Bool(true),
				Cert:           String(""),
				CACert:         String(""),
				CAPath:         String(""),
				Key:            String("key"),
				VerifyIncoming: Bool(false),
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

func TestCTSTLSConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		valid bool
		c     *CTSTLSConfig
	}{
		{
			"nil",
			true,
			nil,
		},
		{
			"empty",
			true,
			&CTSTLSConfig{},
		},
		{
			"valid_fully_configured",
			true,
			&CTSTLSConfig{
				Enabled:        Bool(true),
				Cert:           String("../testutils/cert.pem"),
				CACert:         String("ca_cert.pem"),
				CAPath:         String("ca_path"),
				Key:            String("../testutils/key.pem"),
				VerifyIncoming: Bool(true),
			},
		},
		{
			"enabled_with_no_cert_or_key",
			false,
			&CTSTLSConfig{
				Enabled: Bool(true),
			},
		},
		{
			"cert_no_key",
			false,
			&CTSTLSConfig{
				Cert: String("cert"),
			},
		},
		{
			"key_and_cert_swapped",
			false,
			&CTSTLSConfig{
				Cert: String("../testutils/key.pem"),
				Key:  String("../testutils/cert.pem"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.c.Validate()
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
