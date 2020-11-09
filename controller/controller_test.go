package controller

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	mocksD "github.com/hashicorp/consul-terraform-sync/mocks/driver"
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
	addr := "127.0.0.1:8500"
	ts := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `"test"`) }))
	var err error
	ts.Listener, err = net.Listen("tcp", addr)
	require.NoError(t, err)
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
		initErr     error
		initTaskErr error
		fileReader  func(string) ([]byte, error)
		config      *config.Config
	}{
		{
			"error on driver.InitTask()",
			true,
			nil,
			errors.New("error on driver.InitTask()"),
			func(string) ([]byte, error) { return []byte{}, nil },
			conf,
		},
		{
			"error on newTaskTemplates()",
			true,
			nil,
			nil,
			func(string) ([]byte, error) {
				return []byte{}, errors.New("error on newTaskTemplates()")
			},
			conf,
		},
		{
			"happy path",
			false,
			nil,
			nil,
			func(string) ([]byte, error) { return []byte{}, nil },
			conf,
		},
	}

	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := new(mocksD.Driver)
			d.On("InitTask", mock.Anything, mock.Anything).Return(tc.initTaskErr).Once()

			baseCtrl := baseController{
				newDriver: func(*config.Config, driver.Task) (driver.Driver, error) {
					return d, nil
				},
				conf:       tc.config,
				fileReader: tc.fileReader,
			}

			err := baseCtrl.init(ctx)

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
