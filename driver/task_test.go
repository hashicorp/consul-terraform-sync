package driver

import (
	"reflect"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/client"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/client"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		clientType  string
		expectError bool
		expect      client.Client
	}{
		{
			"happy path with development client",
			developmentClient,
			false,
			&client.Printer{},
		},
		{
			"happy path with mock client",
			testClient,
			false,
			&mocks.Client{},
		},
		{
			"error when creating Terraform CLI client",
			"",
			true,
			&client.TerraformCLI{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := newClient(&clientConfig{
				clientType: tc.clientType,
			})
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, reflect.TypeOf(tc.expect), reflect.TypeOf(actual))
			}
		})
	}
}

func TestTask_ProviderNames(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		task     Task
		expected []string
	}{
		{
			"no provider",
			Task{},
			[]string{},
		},
		{
			"happy path",
			Task{
				providers: NewTerraformProviderBlocks(
					hcltmpl.NewNamedBlocksTest([]map[string]interface{}{
						{"local": map[string]interface{}{
							"configs": "stuff",
						}},
						{"null": map[string]interface{}{}},
					})),
			},
			[]string{"local", "null"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.task.ProviderNames()
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestTask_ServiceNames(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		task     Task
		expected []string
	}{
		{
			"no services",
			Task{},
			[]string{},
		},
		{
			"happy path",
			Task{
				services: []Service{
					Service{Name: "web"},
					Service{Name: "api"},
				},
			},
			[]string{"web", "api"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.task.ServiceNames()
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestTask_configureRootModuleInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name              string
		task              Task
		expectedTemplates []tftmpl.Template
	}{
		{
			name: "templates: services list",
			task: Task{
				services: []Service{
					{
						Name: "api",
					},
					{
						Name:       "web",
						Datacenter: "dc1",
						Namespace:  "ns1",
						Filter:     "filter",
					},
				},
			},
			expectedTemplates: []tftmpl.Template{
				&tftmpl.ServicesTemplate{
					Names: []string{"api", "web"},
					Services: map[string]tftmpl.Service{
						"api": {},
						"web": {
							Datacenter: "dc1",
							Namespace:  "ns1",
							Filter:     "filter",
						},
					},
					RenderVar: true,
				},
			},
		},
		{
			name: "templates: services cond regex",
			task: Task{
				condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Regexp:     config.String("^web.*"),
						Datacenter: config.String("dc1"),
						Namespace:  config.String("ns1"),
						Filter:     config.String("filter"),
					},
					UseAsModuleInput: config.Bool(false),
				},
			},
			expectedTemplates: []tftmpl.Template{
				&tftmpl.ServicesRegexTemplate{
					Regexp:     "^web.*",
					Datacenter: "dc1",
					Namespace:  "ns1",
					Filter:     "filter",
					RenderVar:  false,
				},
			},
		},
		{
			name: "templates: services cond names",
			task: Task{
				condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Names:      []string{"api"},
						Datacenter: config.String("dc1"),
						Namespace:  config.String("ns1"),
						Filter:     config.String("filter"),
					},
					UseAsModuleInput: config.Bool(false),
				},
			},
			expectedTemplates: []tftmpl.Template{
				&tftmpl.ServicesTemplate{
					Names:      []string{"api"},
					Datacenter: "dc1",
					Namespace:  "ns1",
					Filter:     "filter",
					RenderVar:  false,
				},
			},
		},
		{
			name: "templates: services module_input regex",
			task: Task{
				moduleInputs: config.ModuleInputConfigs{
					&config.ServicesModuleInputConfig{
						ServicesMonitorConfig: config.ServicesMonitorConfig{
							Regexp:     config.String("^web.*"),
							Datacenter: config.String("dc1"),
							Namespace:  config.String("ns1"),
							Filter:     config.String("filter"),
						},
					},
				},
			},
			expectedTemplates: []tftmpl.Template{
				&tftmpl.ServicesRegexTemplate{
					Regexp:     "^web.*",
					Datacenter: "dc1",
					Namespace:  "ns1",
					Filter:     "filter",
					RenderVar:  true,
				},
			},
		},
		{
			name: "templates: services module_input names",
			task: Task{
				moduleInputs: config.ModuleInputConfigs{
					&config.ServicesModuleInputConfig{
						ServicesMonitorConfig: config.ServicesMonitorConfig{
							Names:      []string{"api"},
							Datacenter: config.String("dc1"),
							Namespace:  config.String("ns1"),
							Filter:     config.String("filter")},
					},
				},
			},
			expectedTemplates: []tftmpl.Template{
				&tftmpl.ServicesTemplate{
					Names:      []string{"api"},
					Datacenter: "dc1",
					Namespace:  "ns1",
					Filter:     "filter",
					RenderVar:  true,
				},
			},
		},
		{
			name: "templates: multiple module_inputs",
			task: Task{
				moduleInputs: config.ModuleInputConfigs{
					&config.ServicesModuleInputConfig{
						ServicesMonitorConfig: config.ServicesMonitorConfig{
							Regexp:     config.String(".*"),
							Datacenter: config.String("dc1"),
							Namespace:  config.String("ns1"),
							Filter:     config.String("filter"),
						},
					},
					&config.ConsulKVModuleInputConfig{
						ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
							Path:       config.String("path"),
							Recurse:    config.Bool(true),
							Datacenter: config.String("dc1"),
							Namespace:  config.String("ns1"),
						},
					},
				},
			},
			expectedTemplates: []tftmpl.Template{
				&tftmpl.ServicesRegexTemplate{
					Regexp:     ".*",
					Datacenter: "dc1",
					Namespace:  "ns1",
					Filter:     "filter",
					RenderVar:  true,
				},
				&tftmpl.ConsulKVTemplate{
					Path:       "path",
					Recurse:    true,
					Datacenter: "dc1",
					Namespace:  "ns1",
					RenderVar:  true,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.task.logger = logging.NewNullLogger()

			input := &tftmpl.RootModuleInputData{}
			err := tc.task.configureRootModuleInput(input)
			assert.NoError(t, err)

			if len(tc.expectedTemplates) > 0 {
				assert.Equal(t, tc.expectedTemplates, input.Templates)
			}
		})
	}
}
