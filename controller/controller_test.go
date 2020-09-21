package controller

import (
	"testing"

	"github.com/hashicorp/consul-nia/config"
	"github.com/hashicorp/consul-nia/driver"
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

func TestNewDriverTasks(t *testing.T) {
	// newDriverTasks function reorganizes various user-defined configuration
	// blocks into a task object with all the information for the driver to
	// execute on.
	testCases := []struct {
		name  string
		conf  *config.Config
		tasks []driver.Task
	}{
		{
			"no config",
			nil,
			[]driver.Task{},
		}, {
			"no tasks",
			&config.Config{Tasks: &config.TaskConfigs{}},
			[]driver.Task{},
		}, {
			// Fetches correct provider and required_providers blocks from config
			"providers",
			&config.Config{
				Tasks: &config.TaskConfigs{
					{
						Name:      config.String("name"),
						Providers: []string{"providerA", "providerB"},
						Source:    config.String("source"),
					},
				},
				Driver: &config.DriverConfig{
					Terraform: &config.TerraformConfig{
						RequiredProviders: map[string]interface{}{
							"providerA": map[string]string{
								"source": "source/providerA",
							},
						},
					},
				},
				Providers: &config.ProviderConfigs{
					{"providerB": map[string]interface{}{
						"var": "val",
					}},
				},
			},
			[]driver.Task{{
				Name: "name",
				Providers: []map[string]interface{}{
					{"providerA": map[string]interface{}{}},
					{"providerB": map[string]interface{}{
						"var": "val",
					}},
				},
				ProviderInfo: map[string]interface{}{
					"providerA": map[string]string{
						"source": "source/providerA",
					},
				},
				Services: []driver.Service{},
				Source:   "source",
				VarFiles: []string{},
			}},
		}, {
			// Fetches correct provider and required_providers blocks from config
			// with context of alias
			"provider instance",
			&config.Config{
				Tasks: &config.TaskConfigs{
					{
						Name:      config.String("name"),
						Providers: []string{"providerA.alias1", "providerB"},
						Source:    config.String("source"),
					},
				},
				Driver: &config.DriverConfig{
					Terraform: &config.TerraformConfig{
						RequiredProviders: map[string]interface{}{
							"providerA": map[string]string{
								"source": "source/providerA",
							},
						},
					},
				},
				Providers: &config.ProviderConfigs{
					{"providerA": map[string]interface{}{
						"alias": "alias1",
						"foo":   "bar",
					}},
					{"providerA": map[string]interface{}{
						"alias": "alias2",
						"baz":   "baz",
					}},
					{"providerB": map[string]interface{}{
						"var": "val",
					}},
				},
			},
			[]driver.Task{{
				Name: "name",
				Providers: []map[string]interface{}{
					{"providerA": map[string]interface{}{
						"alias": "alias1",
						"foo":   "bar",
					}},
					{"providerB": map[string]interface{}{
						"var": "val",
					}},
				},
				ProviderInfo: map[string]interface{}{
					"providerA": map[string]string{
						"source": "source/providerA",
					},
				},
				Services: []driver.Service{},
				Source:   "source",
				VarFiles: []string{},
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.conf.Finalize()
			tasks := newDriverTasks(tc.conf)
			assert.Equal(t, tc.tasks, tasks)
		})
	}
}
