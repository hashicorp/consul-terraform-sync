package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerraformConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &TerraformConfig{}
	consulConf := DefaultConsulConfig()
	consulConf.Finalize()
	finalizedConf.Finalize(consulConf)

	cases := []struct {
		name string
		a    *TerraformConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&TerraformConfig{},
		},
		{
			"finalized",
			finalizedConf,
		},
		{
			"same_enabled",
			&TerraformConfig{
				Log:  Bool(true),
				Path: String("path"),
				Backend: map[string]interface{}{"consul": map[string]interface{}{
					"path": "consul-terraform-sync/terraform",
				}},
				RequiredProviders: map[string]interface{}{
					"pName1": "v0.0.0",
					"pName2": map[string]string{
						"version": "v0.0.0",
						"source":  "namespace/pName2",
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

func TestTerraformConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *TerraformConfig
		b    *TerraformConfig
		r    *TerraformConfig
	}{
		{
			"nil_a",
			nil,
			&TerraformConfig{},
			&TerraformConfig{},
		},
		{
			"nil_b",
			&TerraformConfig{},
			nil,
			&TerraformConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&TerraformConfig{},
			&TerraformConfig{},
			&TerraformConfig{},
		},
		{
			"version_overrides",
			&TerraformConfig{Version: String("version")},
			&TerraformConfig{Version: String("")},
			&TerraformConfig{Version: String("")},
		},
		{
			"version_empty_one",
			&TerraformConfig{Version: String("version")},
			&TerraformConfig{},
			&TerraformConfig{Version: String("version")},
		},
		{
			"version_empty_two",
			&TerraformConfig{},
			&TerraformConfig{Version: String("version")},
			&TerraformConfig{Version: String("version")},
		},
		{
			"version_same",
			&TerraformConfig{Version: String("version")},
			&TerraformConfig{Version: String("version")},
			&TerraformConfig{Version: String("version")},
		},
		{
			"log_overrides",
			&TerraformConfig{Log: Bool(false)},
			&TerraformConfig{Log: Bool(true)},
			&TerraformConfig{Log: Bool(true)},
		},
		{
			"log_empty_one",
			&TerraformConfig{Log: Bool(true)},
			&TerraformConfig{},
			&TerraformConfig{Log: Bool(true)},
		},
		{
			"log_empty_two",
			&TerraformConfig{},
			&TerraformConfig{Log: Bool(true)},
			&TerraformConfig{Log: Bool(true)},
		},
		{
			"persist_log_overrides",
			&TerraformConfig{PersistLog: Bool(false)},
			&TerraformConfig{PersistLog: Bool(true)},
			&TerraformConfig{PersistLog: Bool(true)},
		},
		{
			"persist_log_empty_one",
			&TerraformConfig{PersistLog: Bool(true)},
			&TerraformConfig{},
			&TerraformConfig{PersistLog: Bool(true)},
		},
		{
			"persist_log_empty_two",
			&TerraformConfig{},
			&TerraformConfig{PersistLog: Bool(true)},
			&TerraformConfig{PersistLog: Bool(true)},
		},
		{
			"persist_log_same",
			&TerraformConfig{PersistLog: Bool(true)},
			&TerraformConfig{PersistLog: Bool(true)},
			&TerraformConfig{PersistLog: Bool(true)},
		},
		{
			"path_overrides",
			&TerraformConfig{Path: String("path")},
			&TerraformConfig{Path: String("")},
			&TerraformConfig{Path: String("")},
		},
		{
			"path_empty_one",
			&TerraformConfig{Path: String("path")},
			&TerraformConfig{},
			&TerraformConfig{Path: String("path")},
		},
		{
			"path_empty_two",
			&TerraformConfig{},
			&TerraformConfig{Path: String("path")},
			&TerraformConfig{Path: String("path")},
		},
		{
			"path_same",
			&TerraformConfig{Path: String("path")},
			&TerraformConfig{Path: String("path")},
			&TerraformConfig{Path: String("path")},
		},
		{
			"backend_overrides",
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-terraform-sync/terraform",
					},
				},
			},
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path":   "consul-terraform-sync/override",
						"scheme": "http",
					},
				},
			},
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path":   "consul-terraform-sync/override",
						"scheme": "http",
					},
				},
			},
		},
		{
			"backend_empty_one",
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-terraform-sync/terraform",
					},
				},
			},
			&TerraformConfig{},
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-terraform-sync/terraform",
					},
				},
			},
		},
		{
			"backend_empty_two",
			&TerraformConfig{},
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-terraform-sync/terraform",
					},
				},
			},
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-terraform-sync/terraform",
					},
				},
			},
		},
		{
			"backend_same",
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-terraform-sync/terraform",
					},
				},
			},
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-terraform-sync/terraform",
					},
				},
			},
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-terraform-sync/terraform",
					},
				},
			},
		},
		{
			"required_providers_overrides",
			&TerraformConfig{
				RequiredProviders: map[string]interface{}{
					"pName1": "v0.0.0",
					"pName2": map[string]string{
						"version": "v0.0.1",
						"source":  "namespace/pName2",
					},
				},
			},
			&TerraformConfig{
				RequiredProviders: map[string]interface{}{
					"pName1": map[string]string{
						"version": "v0.0.0",
						"source":  "namespace/pName1",
					},
				},
			},
			&TerraformConfig{
				RequiredProviders: map[string]interface{}{
					"pName1": map[string]string{
						"version": "v0.0.0",
						"source":  "namespace/pName1",
					},
					"pName2": map[string]string{
						"version": "v0.0.1",
						"source":  "namespace/pName2",
					},
				},
			},
		},
		{
			"required_providers_empty_one",
			&TerraformConfig{
				RequiredProviders: map[string]interface{}{
					"pName1": map[string]string{
						"version": "v0.0.0",
						"source":  "namespace/pName1",
					},
				},
			},
			&TerraformConfig{},
			&TerraformConfig{
				RequiredProviders: map[string]interface{}{
					"pName1": map[string]string{
						"version": "v0.0.0",
						"source":  "namespace/pName1",
					},
				},
			},
		},
		{
			"required_providers_empty_two",
			&TerraformConfig{},
			&TerraformConfig{
				RequiredProviders: map[string]interface{}{
					"pName1": map[string]string{
						"version": "v0.0.0",
						"source":  "namespace/pName1",
					},
				},
			},
			&TerraformConfig{
				RequiredProviders: map[string]interface{}{
					"pName1": map[string]string{
						"version": "v0.0.0",
						"source":  "namespace/pName1",
					},
				},
			},
		},
		{
			"required_providers_same",
			&TerraformConfig{
				RequiredProviders: map[string]interface{}{
					"pName1": map[string]string{
						"version": "v0.0.0",
						"source":  "namespace/pName1",
					},
				},
			},
			&TerraformConfig{
				RequiredProviders: map[string]interface{}{
					"pName1": map[string]string{
						"version": "v0.0.0",
						"source":  "namespace/pName1",
					},
				},
			},
			&TerraformConfig{
				RequiredProviders: map[string]interface{}{
					"pName1": map[string]string{
						"version": "v0.0.0",
						"source":  "namespace/pName1",
					},
				},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestTerraformConfig_Finalize(t *testing.T) {
	t.Parallel()
	wd, err := os.Getwd()
	require.NoError(t, err)

	consul := DefaultConsulConfig()
	consul.Finalize()

	cases := []struct {
		name   string
		i      *TerraformConfig
		consul *ConsulConfig
		r      *TerraformConfig
	}{
		{
			"nil",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&TerraformConfig{},
			nil,
			&TerraformConfig{
				Version:           String(""),
				Log:               Bool(false),
				PersistLog:        Bool(false),
				Path:              String(wd),
				Backend:           map[string]interface{}{},
				RequiredProviders: map[string]interface{}{},
			},
		},
		{
			"default consul backend",
			&TerraformConfig{},
			consul,
			&TerraformConfig{
				Version:    String(""),
				Log:        Bool(false),
				PersistLog: Bool(false),
				Path:       String(wd),
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"address": *consul.Address,
						"path":    DefaultTFBackendKVPath,
						"gzip":    true,
					},
				},
				RequiredProviders: map[string]interface{}{},
			},
		},
		{
			"consul backend with TLS",
			&TerraformConfig{},
			&ConsulConfig{
				TLS: &TLSConfig{
					CACert: String("ca_cert"),
					Cert:   String("client_cert"),
					Key:    String("client_key"),
				},
			},
			&TerraformConfig{
				Version:    String(""),
				Log:        Bool(false),
				PersistLog: Bool(false),
				Path:       String(wd),
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"address":   *consul.Address,
						"path":      DefaultTFBackendKVPath,
						"gzip":      true,
						"scheme":    "https",
						"ca_file":   "ca_cert",
						"cert_file": "client_cert",
						"key_file":  "client_key",
					},
				},
				RequiredProviders: map[string]interface{}{},
			},
		},
		{
			"custom consul backend",
			&TerraformConfig{},
			&ConsulConfig{
				Address: String("127.0.0.1:8080"),
				KVPath:  String("custom-path"),
			},
			&TerraformConfig{
				Version:    String(""),
				Log:        Bool(false),
				PersistLog: Bool(false),
				Path:       String(wd),
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"address": "127.0.0.1:8080",
						"path":    "custom-path/terraform",
						"gzip":    true,
					},
				},
				RequiredProviders: map[string]interface{}{},
			},
		},
		{
			"terraform path empty string",
			&TerraformConfig{
				Version:    String(""),
				Log:        Bool(false),
				PersistLog: Bool(false),
				Path:       String(""),
			},
			nil,
			&TerraformConfig{
				Version:           String(""),
				Log:               Bool(false),
				PersistLog:        Bool(false),
				Path:              String(wd),
				Backend:           map[string]interface{}{},
				RequiredProviders: map[string]interface{}{},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			tc.consul.Finalize()
			tc.i.Finalize(tc.consul)
			assert.Equal(t, tc.r, tc.i)
		})
	}
}

func TestTerraformConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		i       *TerraformConfig
		isValid bool
	}{
		{
			"nil",
			nil,
			false,
		}, {
			"empty",
			&TerraformConfig{},
			false,
		}, {
			"valid consul backend",
			&TerraformConfig{Backend: map[string]interface{}{"consul": nil}},
			true,
		}, {
			"valid local backend",
			&TerraformConfig{Backend: map[string]interface{}{"local": nil}},
			true,
		}, {
			"valid kubernetes backend",
			&TerraformConfig{Backend: map[string]interface{}{"kubernetes": map[string]interface{}{
				"secret_suffix":    "state",
				"load_config_file": true,
			}}},
			true,
		}, {
			"backend_invalid",
			&TerraformConfig{Backend: map[string]interface{}{"unsupported": nil}},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			err := tc.i.Validate()
			if tc.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestDefaultTerraformBackend(t *testing.T) {
	testCases := []struct {
		name     string
		config   *ConsulConfig
		expected map[string]interface{}
		err      bool
	}{
		{
			"default",
			&ConsulConfig{
				Address: String("127.0.0.1:8500"),
			},
			map[string]interface{}{
				"consul": map[string]interface{}{
					"address": "127.0.0.1:8500",
					"path":    DefaultTFBackendKVPath,
					"gzip":    true,
				},
			},
			false,
		}, {
			"nil",
			nil,
			nil,
			true,
		}, {
			"KV path",
			&ConsulConfig{
				Address: String("127.0.0.1:8500"),
				KVPath:  String("custom-path-for-terraform"),
			},
			map[string]interface{}{
				"consul": map[string]interface{}{
					"address": "127.0.0.1:8500",
					"path":    "custom-path-for-terraform/terraform",
					"gzip":    true,
				},
			},
			false,
		}, {
			"KV path suffix",
			&ConsulConfig{
				Address: String("127.0.0.1:8500"),
				KVPath:  String("custom-path/terraform"),
			},
			map[string]interface{}{
				"consul": map[string]interface{}{
					"address": "127.0.0.1:8500",
					"path":    "custom-path/terraform",
					"gzip":    true,
				},
			},
			false,
		}, {
			"TLS sets https",
			&ConsulConfig{
				Address: String("127.0.0.1:8500"),
				TLS: &TLSConfig{
					CACert: String("ca_cert"),
					Cert:   String("client_cert"),
					Key:    String("client_key"),
				},
			},
			map[string]interface{}{
				"consul": map[string]interface{}{
					"address":   "127.0.0.1:8500",
					"path":      DefaultTFBackendKVPath,
					"gzip":      true,
					"scheme":    "https",
					"ca_file":   "ca_cert",
					"cert_file": "client_cert",
					"key_file":  "client_key",
				},
			},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.config.Finalize()
			defaultBackend, err := DefaultTerraformBackend(tc.config)
			if tc.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expected, defaultBackend)
		})
	}
}
