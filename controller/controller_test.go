package controller

import (
	"testing"

	"github.com/hashicorp/consul-nia/config"
	"github.com/hashicorp/consul-nia/handler"
	"github.com/stretchr/testify/assert"
)

func TestGetTerraformHandlers(t *testing.T) {
	cases := []struct {
		name        string
		expectError bool
		nilHandler  bool
		provConfig  *config.ProviderConfigs
	}{
		{
			"no provider",
			false,
			true,
			&config.ProviderConfigs{},
		},
		{
			"error getting handler for provider",
			true,
			true,
			&config.ProviderConfigs{
				&config.ProviderConfig{
					"some-provider": "malformed-config",
				},
			},
		},
		{
			"provider without handler (no error)",
			false,
			true,
			&config.ProviderConfigs{
				&config.ProviderConfig{
					"provider-no-handler": map[string]interface{}{},
				},
			},
		},
		{
			"happy path - provider with handler",
			false,
			false,
			&config.ProviderConfigs{
				&config.ProviderConfig{
					handler.TerraformProviderFake: map[string]interface{}{
						"name": "happy-path",
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &config.Config{
				Providers: tc.provConfig,
			}
			h, err := getTerraformHandlers(c)
			if tc.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tc.nilHandler {
				assert.Nil(t, h)
				return
			}
			assert.NotNil(t, h)
		})
	}
}

func TestGetPostApplyHandlers(t *testing.T) {
	cases := []struct {
		name        string
		expectError bool
		nilHandler  bool
		config      *config.Config
	}{
		{
			"unsupported driver",
			true,
			true,
			&config.Config{
				Driver: &config.DriverConfig{},
			},
		},
		{
			"terraform driver w handler",
			false,
			false,
			&config.Config{
				Driver: &config.DriverConfig{
					Terraform: &config.TerraformConfig{},
				},
				Providers: &config.ProviderConfigs{
					&config.ProviderConfig{
						handler.TerraformProviderFake: map[string]interface{}{
							"name": "happy-path",
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, err := getPostApplyHandlers(tc.config)
			if tc.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tc.nilHandler {
				assert.Nil(t, h)
				return
			}
			assert.NotNil(t, h)
		})
	}
}
