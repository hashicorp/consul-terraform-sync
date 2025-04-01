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
			assert.Equal(t, len(tc.tasks), len(tasks))
			for i := range tasks {
				assert.Equal(t, tc.tasks[i].Description(), tasks[i].Description())
				assert.Equal(t, tc.tasks[i].Name(), tasks[i].Name())
				assert.Equal(t, tc.tasks[i].IsEnabled(), tasks[i].IsEnabled())
				assert.Equal(t, tc.tasks[i].Module(), tasks[i].Module())
				assert.Equal(t, tc.tasks[i].Version(), tasks[i].Version())
				bufferPeriod, _ := tc.tasks[i].BufferPeriod()
				tasksBufferPeriod, _ := tasks[i].BufferPeriod()
				assert.Equal(t, bufferPeriod, tasksBufferPeriod)
				assert.Equal(t, tc.tasks[i].Condition(), tasks[i].Condition())
				assert.Equal(t, tc.tasks[i].ModuleInputs(), tasks[i].ModuleInputs())
				assert.Equal(t, tc.tasks[i].WorkingDir(), tasks[i].WorkingDir())
				assert.Equal(t, tc.tasks[i].DeprecatedTFVersion(), tasks[i].DeprecatedTFVersion())
				assert.Equal(t, tc.tasks[i].TFCWorkspace(), tasks[i].TFCWorkspace())
				assert.Equal(t, tc.tasks[i].Env(), tasks[i].Env())
				assert.Equal(t, tc.tasks[i].Providers(), tasks[i].Providers())
				assert.Equal(t, tc.tasks[i].Services(), tasks[i].Services())
			}
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
