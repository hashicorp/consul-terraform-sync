package driver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBaseControllerInit(t *testing.T) {
	t.Parallel()

	conf := singleTaskConfig()

	cases := []struct {
		name        string
		expectError bool
		initTaskErr error
		config      *config.Config
	}{
		{
			"error on InitTask()",
			true,
			errors.New("error on InitTask()"),
			conf,
		},
		{
			"happy path",
			false,
			nil,
			conf,
		},
	}

	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := new(mocksD.Driver)
			d.On("TemplateIDs").Return(nil)
			d.On("InitTask", mock.Anything).Return(tc.initTaskErr).Once()

			baseCtrl := Factory{
				newDriver: func(*config.Config, *Task, templates.Watcher) (Driver, error) {
					return d, nil
				},
				initConf: tc.config,
				logger:   logging.NewNullLogger(),
			}
			err := baseCtrl.drivers.Add("task", d)
			require.NoError(t, err)

			err = baseCtrl.init(ctx)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.name)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewDriverTask(t *testing.T) {
	// newDriverTask function reorganizes various user-defined configuration
	// blocks into a task object with all the information for the driver to
	// execute on.
	testCases := []struct {
		name  string
		conf  *config.Config
		tasks []*Task
	}{
		{
			"no config",
			nil,
			[]*Task{},
		}, {
			"no tasks",
			&config.Config{Tasks: &config.TaskConfigs{}},
			[]*Task{},
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
					TFVersion:    config.String("1.0.0"),
					TFCWorkspace: config.DefaultTerraformCloudWorkspaceConfig(),
				},
			}},
			[]*Task{newTestTask(t, TaskConfig{
				Description: "description",
				Name:        "name",
				Enabled:     true,
				Module:      "path",
				Version:     "version",
				BufferPeriod: &BufferPeriod{
					Min: 5 * time.Second,
					Max: 20 * time.Second,
				},
				Condition:    config.EmptyConditionConfig(),
				ModuleInputs: *config.DefaultModuleInputConfigs(),
				WorkingDir:   "working-dir/name",

				// Enterprise
				TFVersion:    "1.0.0",
				TFCWorkspace: *config.DefaultTerraformCloudWorkspaceConfig(),

				Env: map[string]string{
					"CONSUL_HTTP_ADDR": "localhost:8500",
				},
				Providers:    TerraformProviderBlocks{},
				ProviderInfo: map[string]interface{}{},
				Services:     []Service{},
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
			[]*Task{newTestTask(t, TaskConfig{
				Name:    "name",
				Enabled: true,
				Env: map[string]string{
					"CONSUL_HTTP_ADDR": "localhost:8500",
				},
				Providers: NewTerraformProviderBlocks(
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
				Services:     []Service{},
				Module:       "path",
				VarFiles:     []string{},
				Condition:    config.EmptyConditionConfig(),
				ModuleInputs: *config.DefaultModuleInputConfigs(),
				BufferPeriod: &BufferPeriod{
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
			[]*Task{newTestTask(t, TaskConfig{
				Name:    "name",
				Enabled: true,
				Env: map[string]string{
					"CONSUL_HTTP_ADDR": "localhost:8500",
				},
				Providers: NewTerraformProviderBlocks(
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
				Services:     []Service{},
				Module:       "path",
				VarFiles:     []string{},
				Condition:    config.EmptyConditionConfig(),
				ModuleInputs: *config.DefaultModuleInputConfigs(),
				BufferPeriod: &BufferPeriod{
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
			[]*Task{newTestTask(t, TaskConfig{
				Name:    "name",
				Enabled: true,
				Env: map[string]string{
					"CONSUL_HTTP_ADDR":  "my.consul.address",
					"CONSUL_HTTP_TOKEN": "TEST_CONSUL_TOKEN",
					"PROVIDER_TOKEN":    "TEST_PROVIDER_TOKEN",
				},
				Providers: NewTerraformProviderBlocks(
					hcltmpl.NewNamedBlocksTest([]map[string]interface{}{
						{"providerA": map[string]interface{}{
							"task_env": map[string]interface{}{
								"PROVIDER_TOKEN": "TEST_PROVIDER_TOKEN",
							},
						}},
					})),
				ProviderInfo: map[string]interface{}{},
				Services:     []Service{},
				Module:       "path",
				VarFiles:     []string{},
				Condition:    config.EmptyConditionConfig(),
				ModuleInputs: *config.DefaultModuleInputConfigs(),
				BufferPeriod: &BufferPeriod{
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
			tc.conf.Finalize()

			var providerConfigs []TerraformProviderBlock
			if tc.conf != nil && tc.conf.TerraformProviders != nil {
				for _, pconf := range *tc.conf.TerraformProviders {
					providerBlock := NewTerraformProviderBlock(hcltmpl.NewNamedBlockTest(*pconf))
					providerConfigs = append(providerConfigs, providerBlock)
				}
			}

			tasks, err := newTestDriverTasks(tc.conf, providerConfigs)
			assert.NoError(t, err)
			assert.Equal(t, tc.tasks, tasks)
		})
	}
}

func newTestDriverTasks(conf *config.Config, providerConfigs TerraformProviderBlocks) ([]*Task, error) {
	if conf == nil {
		return []*Task{}, nil
	}

	tasks := make([]*Task, len(*conf.Tasks))
	for i, t := range *conf.Tasks {
		var err error
		tasks[i], err = newDriverTask(conf, t, providerConfigs)
		if err != nil {
			return nil, err
		}
	}

	return tasks, nil
}

func newTestTask(tb testing.TB, conf TaskConfig) *Task {
	task, err := NewTask(conf)
	require.NoError(tb, err)
	return task
}
