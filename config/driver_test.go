package config

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriverConfig_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *DriverConfig
	}{
		{
			"nil",
			nil,
		}, {
			"empty",
			&DriverConfig{},
		}, {
			"same_enabled",
			&DriverConfig{
				consul:    &ConsulConfig{Address: String("localhost:8500")},
				Terraform: &TerraformConfig{LogLevel: String("debug")},
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

func TestDriverConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *DriverConfig
		b    *DriverConfig
		r    *DriverConfig
	}{
		{
			"nil_a",
			nil,
			&DriverConfig{},
			&DriverConfig{},
		},
		{
			"nil_b",
			&DriverConfig{},
			nil,
			&DriverConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&DriverConfig{},
			&DriverConfig{},
			&DriverConfig{},
		},
		{
			"consul_overrides",
			&DriverConfig{consul: &ConsulConfig{Address: String("127.0.0.1:8500")}},
			&DriverConfig{consul: &ConsulConfig{Address: String("localhost:8500")}},
			&DriverConfig{consul: &ConsulConfig{Address: String("localhost:8500")}},
		},
		{
			"consul_empty_one",
			&DriverConfig{consul: &ConsulConfig{Address: String("localhost:8500")}},
			&DriverConfig{},
			&DriverConfig{consul: &ConsulConfig{Address: String("localhost:8500")}},
		},
		{
			"consul_empty_two",
			&DriverConfig{},
			&DriverConfig{consul: &ConsulConfig{Address: String("localhost:8500")}},
			&DriverConfig{consul: &ConsulConfig{Address: String("localhost:8500")}},
		},
		{
			"consul_same",
			&DriverConfig{consul: &ConsulConfig{Address: String("localhost:8500")}},
			&DriverConfig{consul: &ConsulConfig{Address: String("localhost:8500")}},
			&DriverConfig{consul: &ConsulConfig{Address: String("localhost:8500")}},
		},
		{
			"terraform_overrides",
			&DriverConfig{Terraform: &TerraformConfig{LogLevel: String("info")}},
			&DriverConfig{Terraform: &TerraformConfig{LogLevel: String("")}},
			&DriverConfig{Terraform: &TerraformConfig{LogLevel: String("")}},
		},
		{
			"terraform_empty_one",
			&DriverConfig{Terraform: &TerraformConfig{LogLevel: String("debug")}},
			&DriverConfig{},
			&DriverConfig{Terraform: &TerraformConfig{LogLevel: String("debug")}},
		},
		{
			"terraform_empty_two",
			&DriverConfig{},
			&DriverConfig{Terraform: &TerraformConfig{LogLevel: String("debug")}},
			&DriverConfig{Terraform: &TerraformConfig{LogLevel: String("debug")}},
		},
		{
			"terraform_same",
			&DriverConfig{Terraform: &TerraformConfig{LogLevel: String("debug")}},
			&DriverConfig{Terraform: &TerraformConfig{LogLevel: String("debug")}},
			&DriverConfig{Terraform: &TerraformConfig{LogLevel: String("debug")}},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestDriverConfig_Finalize(t *testing.T) {
	t.Parallel()
	wd, err := os.Getwd()
	require.NoError(t, err)

	cases := []struct {
		name string
		i    *DriverConfig
		r    *DriverConfig
	}{
		{
			"nil",
			nil,
			nil,
		},
		{
			"empty",
			&DriverConfig{},
			&DriverConfig{
				Terraform: &TerraformConfig{
					LogLevel:          String("info"),
					Path:              String(wd),
					DataDir:           String(path.Join(wd, DefaultTFDataDir)),
					WorkingDir:        String(path.Join(wd, DefaultTFWorkingDir)),
					SkipVerify:        Bool(false),
					Backend:           map[string]interface{}{},
					RequiredProviders: map[string]interface{}{},
				},
			},
		},
		{
			"with_terraform",
			&DriverConfig{
				Terraform: &TerraformConfig{
					LogLevel:   String("debug"),
					SkipVerify: Bool(true),
				},
			},
			&DriverConfig{
				Terraform: &TerraformConfig{
					LogLevel:          String("debug"),
					Path:              String(wd),
					DataDir:           String(path.Join(wd, DefaultTFDataDir)),
					WorkingDir:        String(path.Join(wd, DefaultTFWorkingDir)),
					SkipVerify:        Bool(true),
					Backend:           map[string]interface{}{},
					RequiredProviders: map[string]interface{}{},
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

func TestDriverConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		i       *DriverConfig
		isValid bool
	}{
		{
			"nil",
			nil,
			false,
		}, {
			"empty",
			&DriverConfig{},
			false,
		}, {
			"valid",
			&DriverConfig{Terraform: &TerraformConfig{Backend: map[string]interface{}{"consul": nil}}},
			true,
		}, {
			"terraform_invalid",
			&DriverConfig{Terraform: &TerraformConfig{}},
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
