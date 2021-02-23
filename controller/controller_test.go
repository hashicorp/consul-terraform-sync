package controller

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	mocksD "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewControllers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		conf        *config.Config
	}{
		{
			"happy path",
			false,
			singleTaskConfig(),
		},
		{
			"unreachable consul server", // can take >63s locally
			true,
			singleTaskConfig(),
		},
		{
			"unsupported driver error",
			true,
			&config.Config{
				Driver: &config.DriverConfig{},
			},
		},
	}
	// fake consul server
	ts := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `"test"`) }))
	var err error
	ts.Listener, err = net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := ts.Listener.Addr().(*net.TCPAddr).Port
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ts.Start()
	defer ts.Close()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectError == false {
				tc.conf.Consul.Address = &addr
				tc.conf.Finalize()
			}

			t.Run("readwrite", func(t *testing.T) {
				controller, err := NewReadWrite(tc.conf, event.NewStore())
				if tc.expectError {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
				assert.NotNil(t, controller)
			})
			t.Run("readonly", func(t *testing.T) {
				controller, err := NewReadOnly(tc.conf)
				if tc.expectError {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
				assert.NotNil(t, controller)
			})
		})
	}
}

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
			"error on driver.InitTask()",
			true,
			errors.New("error on driver.InitTask()"),
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
			d.On("InitTask", mock.Anything, mock.Anything).Return(tc.initTaskErr).Once()

			baseCtrl := baseController{
				newDriver: func(*config.Config, driver.Task, templates.Watcher) (driver.Driver, error) {
					return d, nil
				},
				conf: tc.config,
			}

			_, err := baseCtrl.init(ctx)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.name)
			} else {
				assert.NoError(t, err)
			}
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
				TerraformProviders: &config.TerraformProviderConfigs{
					{"providerB": map[string]interface{}{
						"var": "val",
					}},
				},
			},
			[]driver.Task{{
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
				Services:        []driver.Service{},
				Source:          "source",
				VarFiles:        []string{},
				UserDefinedMeta: map[string]map[string]string{},
				BufferPeriod: &driver.BufferPeriod{
					Min: 5 * time.Second,
					Max: 20 * time.Second,
				},
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
			[]driver.Task{{
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
				Services:        []driver.Service{},
				Source:          "source",
				VarFiles:        []string{},
				UserDefinedMeta: map[string]map[string]string{},
				BufferPeriod: &driver.BufferPeriod{
					Min: 5 * time.Second,
					Max: 20 * time.Second,
				},
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.conf.Finalize()

			var providerConfigs []driver.TerraformProviderBlock
			if tc.conf != nil && tc.conf.TerraformProviders != nil {
				for _, pconf := range *tc.conf.TerraformProviders {
					providerBlock := driver.NewTerraformProviderBlock(hcltmpl.NewNamedBlockTest(*pconf))
					providerConfigs = append(providerConfigs, providerBlock)
				}
			}

			tasks := newDriverTasks(tc.conf, providerConfigs)
			assert.Equal(t, tc.tasks, tasks)
		})
	}
}

func TestGetTemplateBufferPeriods(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		conf     *config.Config
		task     *config.TaskConfig
		expected *driver.BufferPeriod
	}{
		{
			"task config override default values",
			&config.Config{
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(true),
					Min:     config.TimeDuration(2 * time.Second),
					Max:     config.TimeDuration(10 * time.Second),
				},
			},
			&config.TaskConfig{
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(true),
					Min:     config.TimeDuration(5 * time.Second),
					Max:     config.TimeDuration(7 * time.Second),
				},
			},
			&driver.BufferPeriod{
				Min: 5 * time.Second,
				Max: 7 * time.Second,
			},
		},
		{
			"use default values",
			&config.Config{
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(true),
					Min:     config.TimeDuration(2 * time.Second),
					Max:     config.TimeDuration(10 * time.Second),
				},
			},
			&config.TaskConfig{
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(false),
				},
			},
			&driver.BufferPeriod{
				Min: 2 * time.Second,
				Max: 10 * time.Second,
			},
		},
		{
			"nil default config",
			nil,
			&config.TaskConfig{
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(false),
				},
			},
			nil,
		},
		{
			"both disabled",
			&config.Config{
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(false),
				},
			},
			&config.TaskConfig{
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(false),
				},
			},
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := getTemplateBufferPeriod(tc.conf, tc.task)
			assert.Equal(t, tc.expected, actual)
		})
	}

}
