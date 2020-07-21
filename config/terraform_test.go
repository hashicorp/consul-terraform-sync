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
				LogLevel:   String("err"),
				Path:       String("path"),
				DataDir:    String("data"),
				WorkingDir: String("working"),
				SkipVerify: Bool(false),
				Backend: map[string]interface{}{"consul": map[string]interface{}{
					"path": "consul-nia/terraform",
				}},
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
			"log_level_overrides",
			&TerraformConfig{LogLevel: String("info")},
			&TerraformConfig{LogLevel: String("debug")},
			&TerraformConfig{LogLevel: String("debug")},
		},
		{
			"log_level_empty_one",
			&TerraformConfig{LogLevel: String("debug")},
			&TerraformConfig{},
			&TerraformConfig{LogLevel: String("debug")},
		},
		{
			"log_level_empty_two",
			&TerraformConfig{},
			&TerraformConfig{LogLevel: String("debug")},
			&TerraformConfig{LogLevel: String("debug")},
		},
		{
			"log_level_same",
			&TerraformConfig{LogLevel: String("debug")},
			&TerraformConfig{LogLevel: String("debug")},
			&TerraformConfig{LogLevel: String("debug")},
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
			"data_dir_overrides",
			&TerraformConfig{DataDir: String("data")},
			&TerraformConfig{DataDir: String("")},
			&TerraformConfig{DataDir: String("")},
		},
		{
			"data_dir_empty_one",
			&TerraformConfig{DataDir: String("data")},
			&TerraformConfig{},
			&TerraformConfig{DataDir: String("data")},
		},
		{
			"data_dir_empty_two",
			&TerraformConfig{},
			&TerraformConfig{DataDir: String("data")},
			&TerraformConfig{DataDir: String("data")},
		},
		{
			"data_dir_same",
			&TerraformConfig{DataDir: String("data")},
			&TerraformConfig{DataDir: String("data")},
			&TerraformConfig{DataDir: String("data")},
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
			"skip_verify_overrides",
			&TerraformConfig{SkipVerify: Bool(true)},
			&TerraformConfig{SkipVerify: Bool(false)},
			&TerraformConfig{SkipVerify: Bool(false)},
		},
		{
			"skip_verify_empty_one",
			&TerraformConfig{SkipVerify: Bool(true)},
			&TerraformConfig{},
			&TerraformConfig{SkipVerify: Bool(true)},
		},
		{
			"skip_verify_empty_two",
			&TerraformConfig{},
			&TerraformConfig{SkipVerify: Bool(true)},
			&TerraformConfig{SkipVerify: Bool(true)},
		},
		{
			"skip_verify_same",
			&TerraformConfig{SkipVerify: Bool(true)},
			&TerraformConfig{SkipVerify: Bool(true)},
			&TerraformConfig{SkipVerify: Bool(true)},
		},
		{
			"backend_overrides",
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-nia/terraform",
					},
				},
			},
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-nia/override",
					},
				},
			},
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-nia/override",
					},
				},
			},
		},
		{
			"backend_empty_one",
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-nia/terraform",
					},
				},
			},
			&TerraformConfig{},
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-nia/terraform",
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
						"path": "consul-nia/terraform",
					},
				},
			},
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-nia/terraform",
					},
				},
			},
		},
		{
			"backend_same",
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-nia/terraform",
					},
				},
			},
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-nia/terraform",
					},
				},
			},
			&TerraformConfig{
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"path": "consul-nia/terraform",
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
				LogLevel:   String(DefaultTFLogLevel),
				Path:       String(wd),
				DataDir:    String(path.Join(wd, DefaultTFDataDir)),
				WorkingDir: String(path.Join(wd, DefaultTFWorkingDir)),
				SkipVerify: Bool(false),
				Backend:    map[string]interface{}{},
			},
		},
		{
			"default consul backend",
			&TerraformConfig{},
			consul,
			&TerraformConfig{
				LogLevel:   String(DefaultTFLogLevel),
				Path:       String(wd),
				DataDir:    String(path.Join(wd, DefaultTFDataDir)),
				WorkingDir: String(path.Join(wd, DefaultTFWorkingDir)),
				SkipVerify: Bool(false),
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"address": *consul.Address,
						"path":    DefaultTFBackendKVPath,
						"gzip":    true,
					},
				},
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
				LogLevel:   String(DefaultTFLogLevel),
				Path:       String(wd),
				DataDir:    String(path.Join(wd, DefaultTFDataDir)),
				WorkingDir: String(path.Join(wd, DefaultTFWorkingDir)),
				SkipVerify: Bool(false),
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"address": "127.0.0.1:8080",
						"path":    "custom-path/terraform",
						"gzip":    true,
					},
				},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
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
