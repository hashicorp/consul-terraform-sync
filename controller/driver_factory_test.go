// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/logging"
	mocksD "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	mocksTmpl "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_driverFactory_Init(t *testing.T) {
	t.Parallel()

	conf := singleTaskConfig(t)

	cases := []struct {
		name        string
		expectError bool
		config      *config.Config
		// Note: if driverFactory is refactored to the drivers package, we
		// can add a check for expected providers. driver.TerraformProviderBlock
		// fields are private so currently cannot easily check
	}{
		{
			"happy path",
			false,
			conf,
		},
	}

	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := driverFactory{
				initConf: tc.config,
				logger:   logging.NewNullLogger(),
			}

			err := f.Init(ctx)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.name)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_driverFactory_Make(t *testing.T) {
	t.Parallel()

	conf := singleTaskConfig(t)
	taskConf := *(*conf.Tasks)[0]

	// Mock watcher
	w := new(mocksTmpl.Watcher)
	w.On("Clients").Return(nil).Once()
	w.On("Register", mock.Anything).Return(nil).Once()

	// Setup for driver factory
	f, err := NewDriverFactory(conf, w)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("happy path", func(t *testing.T) {
		// Mock returned driver
		d := new(mocksD.Driver)
		f.newDriver = func(context.Context, *config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			d.On("InitTask", mock.Anything).Return(nil).Once()
			return d, nil
		}

		// Test Make
		actualD, err := f.Make(ctx, conf, taskConf)
		assert.NoError(t, err)
		assert.NotNil(t, actualD)
		d.AssertExpectations(t)
	})

	t.Run("driver init error", func(t *testing.T) {
		// Mock returned driver
		errStr := "init error"
		d := new(mocksD.Driver)
		f.newDriver = func(context.Context, *config.Config, *driver.Task, templates.Watcher) (driver.Driver, error) {
			d.On("InitTask", mock.Anything).Return(errors.New(errStr)).Once()
			d.On("DestroyTask", mock.Anything).Return().Once()
			return d, nil
		}

		// Test Make
		_, err = f.Make(ctx, conf, taskConf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), errStr)
		d.AssertExpectations(t)
	})
}

func TestNewDriverTask(t *testing.T) {
	// newDriverTask function reorganizes various user-defined configuration
	// blocks into a task object with all the information for the driver to
	// execute on.
	testCases := []struct {
		name  string
		conf  *config.Config
		tasks []*driver.Task
	}{
		{
			"no config",
			nil,
			[]*driver.Task{},
		}, {
			"no tasks",
			&config.Config{Tasks: &config.TaskConfigs{}},
			[]*driver.Task{},
		}, {
			"basic task fields",
			&config.Config{Tasks: &config.TaskConfigs{
				&config.TaskConfig{
					Description:  config.String("description"),
					Name:         config.String("name"),
					Enabled:      config.Bool(true),
					Module:       config.String("path"),
					Version:      config.String("version"),
					BufferPeriod: config.DefaultBufferPeriodConfig(),
					Condition:    config.EmptyConditionConfig(),
					ModuleInputs: config.DefaultModuleInputConfigs(),
					WorkingDir:   config.String("working-dir/name"),

					// Enterprise
					DeprecatedTFVersion: config.String("1.0.0"),
					TFCWorkspace:        config.DefaultTerraformCloudWorkspaceConfig(),
				},
			}},
			[]*driver.Task{newTestTask(t, driver.TaskConfig{
				Description: "description",
				Name:        "name",
				Enabled:     true,
				Module:      "path",
				Version:     "version",
				BufferPeriod: &driver.BufferPeriod{
					Min: 5 * time.Second,
					Max: 20 * time.Second,
				},
				Condition:    config.EmptyConditionConfig(),
				ModuleInputs: *config.DefaultModuleInputConfigs(),
				WorkingDir:   "working-dir/name",

				// Enterprise
				DeprecatedTFVersion: "1.0.0",
				TFCWorkspace:        *config.DefaultTerraformCloudWorkspaceConfig(),

				Env: map[string]string{
					"CONSUL_HTTP_ADDR": "localhost:8500",
				},
				Providers:    driver.TerraformProviderBlocks{},
				ProviderInfo: map[string]interface{}{},
				Services:     []driver.Service{},
			})},
		}, {
			// Fetches correct provider and required_providers blocks from config
			"providers",
			&config.Config{
				Tasks: &config.TaskConfigs{
					{
						Name:      config.String("name"),
						Providers: []string{"providerA", "providerB"},
						Module:    config.String("path"),
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
				TerraformProviders: &config.TerraformProviderConfigs{
					{"providerB": map[string]interface{}{
						"var": "val",
					}},
				},
			},
			[]*driver.Task{newTestTask(t, driver.TaskConfig{
				Name:    "name",
				Enabled: true,
				Env: map[string]string{
					"CONSUL_HTTP_ADDR": "localhost:8500",
				},
				Providers: driver.NewTerraformProviderBlocks(
					hcltmpl.NewNamedBlocksTest([]map[string]interface{}{
						{"providerA": map[string]interface{}{}},
						{"providerB": map[string]interface{}{
							"var": "val",
						}},
					})),
				ProviderInfo: map[string]interface{}{
					"providerA": map[string]string{
						"source": "source/providerA",
					},
				},
				Services:     []driver.Service{},
				Module:       "path",
				Condition:    config.EmptyConditionConfig(),
				ModuleInputs: *config.DefaultModuleInputConfigs(),
				BufferPeriod: &driver.BufferPeriod{
					Min: 5 * time.Second,
					Max: 20 * time.Second,
				},
				WorkingDir: "sync-tasks/name",

				// Enterprise
				TFCWorkspace: *config.DefaultTerraformCloudWorkspaceConfig(),
			})},
		}, {
			// Fetches correct provider and required_providers blocks from config
			// with context of alias
			"provider instance",
			&config.Config{
				Tasks: &config.TaskConfigs{
					{
						Name:      config.String("name"),
						Providers: []string{"providerA.alias1", "providerB"},
						Module:    config.String("path"),
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
				TerraformProviders: &config.TerraformProviderConfigs{
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
			[]*driver.Task{newTestTask(t, driver.TaskConfig{
				Name:    "name",
				Enabled: true,
				Env: map[string]string{
					"CONSUL_HTTP_ADDR": "localhost:8500",
				},
				Providers: driver.NewTerraformProviderBlocks(
					hcltmpl.NewNamedBlocksTest([]map[string]interface{}{
						{"providerA": map[string]interface{}{
							"alias": "alias1",
							"foo":   "bar",
						}},
						{"providerB": map[string]interface{}{
							"var": "val",
						}},
					})),
				ProviderInfo: map[string]interface{}{
					"providerA": map[string]string{
						"source": "source/providerA",
					},
				},
				Services:     []driver.Service{},
				Module:       "path",
				Condition:    config.EmptyConditionConfig(),
				ModuleInputs: *config.DefaultModuleInputConfigs(),
				BufferPeriod: &driver.BufferPeriod{
					Min: 5 * time.Second,
					Max: 20 * time.Second,
				},
				WorkingDir: "sync-tasks/name",

				// Enterprise
				TFCWorkspace: *config.DefaultTerraformCloudWorkspaceConfig(),
			})},
		}, {
			// Task env is fetched from providers and Consul config when using
			// default backend
			"task env",
			&config.Config{
				Consul: &config.ConsulConfig{
					Address: config.String("my.consul.address"),
					Token:   config.String("TEST_CONSUL_TOKEN"),
				},
				Tasks: &config.TaskConfigs{
					{
						Name:      config.String("name"),
						Providers: []string{"providerA"},
						Module:    config.String("path"),
					},
				},
				TerraformProviders: &config.TerraformProviderConfigs{
					{"providerA": map[string]interface{}{
						"task_env": map[string]interface{}{
							"PROVIDER_TOKEN": "TEST_PROVIDER_TOKEN",
						},
					}},
				},
			},
			[]*driver.Task{newTestTask(t, driver.TaskConfig{
				Name:    "name",
				Enabled: true,
				Env: map[string]string{
					"CONSUL_HTTP_ADDR":  "my.consul.address",
					"CONSUL_HTTP_TOKEN": "TEST_CONSUL_TOKEN",
					"PROVIDER_TOKEN":    "TEST_PROVIDER_TOKEN",
				},
				Providers: driver.NewTerraformProviderBlocks(
					hcltmpl.NewNamedBlocksTest([]map[string]interface{}{
						{"providerA": map[string]interface{}{
							"task_env": map[string]interface{}{
								"PROVIDER_TOKEN": "TEST_PROVIDER_TOKEN",
							},
						}},
					})),
				ProviderInfo: map[string]interface{}{},
				Services:     []driver.Service{},
				Module:       "path",
				Condition:    config.EmptyConditionConfig(),
				ModuleInputs: *config.DefaultModuleInputConfigs(),
				BufferPeriod: &driver.BufferPeriod{
					Min: 5 * time.Second,
					Max: 20 * time.Second,
				},
				WorkingDir: "sync-tasks/name",
				// Enterprise
				TFCWorkspace: *config.DefaultTerraformCloudWorkspaceConfig(),
			})},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.conf.Finalize()
			require.NoError(t, err)

			var providerConfigs []driver.TerraformProviderBlock
			if tc.conf != nil && tc.conf.TerraformProviders != nil {
				for _, pconf := range *tc.conf.TerraformProviders {
					providerBlock := driver.NewTerraformProviderBlock(hcltmpl.NewNamedBlockTest(*pconf))
					providerConfigs = append(providerConfigs, providerBlock)
				}
			}

			tasks, err := newTestDriverTasks(tc.conf, providerConfigs)
			assert.NoError(t, err)
			assert.Equal(t, tc.tasks, tasks)
		})
	}
}

func newTestDriverTasks(conf *config.Config, providerConfigs driver.TerraformProviderBlocks) ([]*driver.Task, error) {
	if conf == nil {
		return []*driver.Task{}, nil
	}

	tasks := make([]*driver.Task, len(*conf.Tasks))
	for i, t := range *conf.Tasks {
		var err error
		tasks[i], err = newDriverTask(conf, t, providerConfigs)
		if err != nil {
			return nil, err
		}
	}

	return tasks, nil
}

func newTestTask(tb testing.TB, conf driver.TaskConfig) *driver.Task {
	task, err := driver.NewTask(conf)
	require.NoError(tb, err)
	return task
}
