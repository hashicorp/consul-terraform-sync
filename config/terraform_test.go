package config

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerraformConfig_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *TerraformConfig
	}{
		{
			"nil",
			nil,
		}, {
			"empty",
			&TerraformConfig{},
		}, {
			"same_enabled",
			&TerraformConfig{
				Log:        Bool(true),
				Path:       String("path"),
				WorkingDir: String("working"),
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
			"working_dir_overrides",
			&TerraformConfig{WorkingDir: String("working")},
			&TerraformConfig{WorkingDir: String("")},
			&TerraformConfig{WorkingDir: String("")},
		},
		{
			"working_dir_empty_one",
			&TerraformConfig{WorkingDir: String("working")},
			&TerraformConfig{},
			&TerraformConfig{WorkingDir: String("working")},
		},
		{
			"working_dir_empty_two",
			&TerraformConfig{},
			&TerraformConfig{WorkingDir: String("working")},
			&TerraformConfig{WorkingDir: String("working")},
		},
		{
			"working_dir_same",
			&TerraformConfig{WorkingDir: String("working")},
			&TerraformConfig{WorkingDir: String("working")},
			&TerraformConfig{WorkingDir: String("working")},
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
				Log:               Bool(false),
				PersistLog:        Bool(false),
				Path:              String(wd),
				WorkingDir:        String(path.Join(wd, DefaultTFWorkingDir)),
				Backend:           map[string]interface{}{},
				RequiredProviders: map[string]interface{}{},
			},
		},
		{
			"default consul backend",
			&TerraformConfig{},
			consul,
			&TerraformConfig{
				Log:        Bool(false),
				PersistLog: Bool(false),
				Path:       String(wd),
				WorkingDir: String(path.Join(wd, DefaultTFWorkingDir)),
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
				Log:        Bool(false),
				PersistLog: Bool(false),
				Path:       String(wd),
				WorkingDir: String(path.Join(wd, DefaultTFWorkingDir)),
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
				Log:        Bool(false),
				PersistLog: Bool(false),
				Path:       String(wd),
				WorkingDir: String(path.Join(wd, DefaultTFWorkingDir)),
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
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			tc.consul.Finalize()
			tc.i.Finalize(tc.consul)
			assert.Equal(t, tc.r, tc.i)
		})
	}
}

func TestTerraformValidate(t *testing.T) {
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
			"valid",
			&TerraformConfig{Backend: map[string]interface{}{"consul": nil}},
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
