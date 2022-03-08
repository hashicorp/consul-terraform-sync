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
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
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

func TestTask_BufferPeriod(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		bp           *BufferPeriod
		isConfigured bool
	}{
		{
			name: "configured",
			bp: &BufferPeriod{
				Min: 0,
				Max: 5,
			},
			isConfigured: true,
		},
		{
			name:         "not configured",
			isConfigured: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var task Task
			task.bufferPeriod = tc.bp
			bp, isConfigured := task.BufferPeriod()
			require.Equal(t, tc.isConfigured, isConfigured)

			if tc.bp == nil {
				require.Equal(t, BufferPeriod{}, bp)
			} else {
				require.Equal(t, *tc.bp, bp)
			}
		})
	}
}

func TestTask_Condition(t *testing.T) {
	t.Parallel()

	var task Task
	task.condition = &config.ConsulKVConditionConfig{}
	con := task.Condition()
	assert.Equal(t, task.condition, con)
}

func TestTask_ModuleInputs(t *testing.T) {
	t.Parallel()

	var task Task
	task.moduleInputs = config.ModuleInputConfigs{}
	con := task.ModuleInputs()
	assert.Equal(t, task.moduleInputs, con)
}

func TestTask_IsScheduled(t *testing.T) {
	cases := []struct {
		name        string
		condition   config.ConditionConfig
		isScheduled bool
	}{
		{
			name:        "schedule condition",
			condition:   &config.ScheduleConditionConfig{},
			isScheduled: true,
		},
		{
			name:        "non schedule condition",
			condition:   &config.ConsulKVConditionConfig{},
			isScheduled: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var task Task
			task.condition = tc.condition
			isScheduled := task.IsScheduled()
			assert.Equal(t, tc.isScheduled, isScheduled)
		})
	}
}

func TestTask_Description(t *testing.T) {
	t.Parallel()

	var task Task
	task.description = "some description"
	description := task.Description()
	assert.Equal(t, task.description, description)
}

func TestTask_Name(t *testing.T) {
	t.Parallel()

	var task Task
	task.name = "task-name"
	name := task.Name()
	assert.Equal(t, task.name, name)
}

func TestTask_IsEnabled(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		isEnabled bool
	}{
		{
			name:      "enabled",
			isEnabled: true,
		},
		{
			name:      "disabled",
			isEnabled: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var task Task
			task.enabled = tc.isEnabled
			isEnabled := task.IsEnabled()
			assert.Equal(t, tc.isEnabled, isEnabled)
		})
	}
}

func TestTask_Enable(t *testing.T) {
	var task Task
	task.enabled = false
	task.Enable()
	assert.True(t, task.enabled)
}

func TestTask_Disable(t *testing.T) {
	var task Task
	task.enabled = true
	task.Disable()
	assert.False(t, task.enabled)
}

func TestTask_Env(t *testing.T) {
	var task Task
	task.env = map[string]string{
		"k1": "v1",
		"k2": "v2",
	}
	env := task.Env()
	assert.Equal(t, task.Env(), env)

	// Make sure we made a deep copy
	env["k3"] = "v3"
	assert.NotEqual(t, task.Env(), env)
}

func TestTask_Providers(t *testing.T) {
	var task Task
	task.providers = NewTerraformProviderBlocks(
		hcltmpl.NewNamedBlocksTest([]map[string]interface{}{
			{"local": map[string]interface{}{
				"configs": "stuff",
			}},
			{"null": map[string]interface{}{}},
		}))
	providers := task.Providers()
	assert.Equal(t, task.providers, providers)
}

func TestTask_Services(t *testing.T) {
	var task Task
	task.services = []Service{
		{UserDefinedMeta: make(map[string]string)},
		{Datacenter: "1", UserDefinedMeta: map[string]string{"test": "test"}},
	}
	services := task.Services()
	assert.Equal(t, task.services, services)
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
					{Name: "web"},
					{Name: "api"},
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

func TestTask_Module(t *testing.T) {
	var task Task
	task.module = "some module"
	module := task.Module()
	assert.Equal(t, task.module, module)
}

func TestTask_Variables(t *testing.T) {
	var task Task
	task.variables = map[string]cty.Value{
		"attr": cty.StringVal("value"),
		"block": cty.ObjectVal(map[string]cty.Value{
			"inner": cty.MustParseNumberVal("1"),
		}),
	}
	variables := task.Variables()
	assert.Equal(t, task.variables, variables)
}

func TestTask_Version(t *testing.T) {
	var task Task
	task.version = "some version"
	version := task.Version()
	assert.Equal(t, task.version, version)
}

func TestTask_WorkingDir(t *testing.T) {
	var task Task
	task.workingDir = "working-dir"
	workingDir := task.WorkingDir()
	assert.Equal(t, task.workingDir, workingDir)
}

func TestTask_TFVersion(t *testing.T) {
	var task Task
	task.tfVersion = "1.0.0"
	tfVersion := task.TFVersion()
	assert.Equal(t, task.tfVersion, tfVersion)
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
			name: "templates: catalog services condition",
			task: Task{
				condition: &config.CatalogServicesConditionConfig{
					CatalogServicesMonitorConfig: config.CatalogServicesMonitorConfig{
						Regexp:           config.String("^web.*"),
						Datacenter:       config.String("dc1"),
						Namespace:        config.String("ns1"),
						NodeMeta:         map[string]string{"test": "test"},
						UseAsModuleInput: config.Bool(true),
					},
				},
			},
			expectedTemplates: []tftmpl.Template{
				&tftmpl.CatalogServicesTemplate{
					Regexp:     "^web.*",
					Datacenter: "dc1",
					Namespace:  "ns1",
					NodeMeta:   map[string]string{"test": "test"},
					RenderVar:  true,
				},
			},
		},
		{
			name: "templates: consul kv condition",
			task: Task{
				condition: &config.ConsulKVConditionConfig{
					ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
						Path:       config.String("/path/to/key"),
						Datacenter: config.String("dc1"),
						Namespace:  config.String("ns1"),
						Recurse:    config.Bool(true),
					},
					UseAsModuleInput: config.Bool(true),
				},
			},
			expectedTemplates: []tftmpl.Template{
				&tftmpl.ConsulKVTemplate{
					Path:       "/path/to/key",
					Datacenter: "dc1",
					Namespace:  "ns1",
					Recurse:    true,
					RenderVar:  true,
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
