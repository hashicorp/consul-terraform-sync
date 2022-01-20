package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	taskReq = `{
    "description": "Writes the service name, id, and IP address to a file",
    "enabled": true,
    "name": "new-example-task",
    "providers": [
        "local"
    ],
    "services": [
        "api"
    ],
    "module": "./example-module",
    "variable_files": [],
    "buffer_period": {
        "enabled": true,
        "max": "0s",
        "min": "0s"
    }
}`
)

func TestTaskRequest_String(t *testing.T) {
	var req TaskRequest

	err := json.Unmarshal([]byte(taskReq), &req)
	require.NoError(t, err)

	actual := fmt.Sprintf("%s", req)
	expected := `{"buffer_period":{"enabled":true,"max":"0s","min":"0s"},` +
		`"description":"Writes the service name, id, and IP address to a file",` +
		`"enabled":true,"module":"./example-module","name":"new-example-task",` +
		`"providers":["local"],"services":["api"]}`
	require.Equal(t, expected, actual)
}

// Test only bare minimum, task conversion scenarios covered in
// TestRequest_oapigenTaskFromConfigTask and terraform variable files
// covered in TestRequest_readToVariablesMap
func TestRequest_TaskRequestFromTaskConfig(t *testing.T) {
	cases := []struct {
		name            string
		taskConfig      config.TaskConfig
		expectedRequest TaskRequest
	}{
		{
			name:            "default_values_only",
			taskConfig:      config.TaskConfig{},
			expectedRequest: TaskRequest{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := TaskRequestFromTaskConfig(tc.taskConfig)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedRequest, actual)
		})
	}
}

func TestRequest_readToVariablesMap(t *testing.T) {
	// Simulate input multiple "files" and check various hcl supported types
	inputTFVars := map[string][]byte{
		"simple.tfvars": []byte("singleKey = \"value\""),
		"complex.tfvars": []byte(`
b = true
key = "some_key"
num = 10
obj = {
  argStr = "value"
  argNum = 10
  argList = ["l", "i", "s", "t"]
  argMap = {}
}
l = [1, 2, 3]
tup = ["abc", 123, true]`),
	}
	expectedMap := map[string]string{
		"singleKey": "\"value\"",
		"key":       "\"some_key\"",
		"b":         "true",
		"num":       "10",
		"obj":       "{\"argList\":[\"l\",\"i\",\"s\",\"t\"],\"argMap\":{},\"argNum\":10,\"argStr\":\"value\"}",
		"l":         "[1,2,3]",
		"tup":       "[\"abc\",123,true]",
	}

	// Length should equal number of fields in inputTFVars
	expectedLength := 7

	m := make(map[string]string)

	for k, v := range inputTFVars {
		err := readToVariablesMap(k, bytes.NewReader(v), m)
		assert.NoError(t, err)
	}

	assert.Equal(t, expectedLength, len(m))

	for k := range expectedMap {
		assert.Equal(t, m[k], expectedMap[k])
	}
}

func TestRequest_oapigenTaskFromConfigTask(t *testing.T) {
	cases := []struct {
		name            string
		taskConfig      config.TaskConfig
		expectedRequest oapigen.Task
	}{
		{
			name:            "default_values_only",
			taskConfig:      config.TaskConfig{},
			expectedRequest: oapigen.Task{},
		},
		{
			name: "basic_fields_filled",
			taskConfig: config.TaskConfig{
				Description:  config.String("test-description"),
				Name:         config.String("test-name"),
				Providers:    []string{"test-provider-1", "test-provider-2"},
				Services:     []string{"api", "web"},
				Module:       config.String("path"),
				TFVersion:    config.String(""),
				Version:      config.String("test-version"),
				BufferPeriod: config.DefaultBufferPeriodConfig(),
				Enabled:      config.Bool(true),
				Condition:    config.EmptyConditionConfig(),
				ModuleInputs: config.DefaultModuleInputConfigs(),
			},
			expectedRequest: oapigen.Task{
				Name:        "test-name",
				Module:      "path",
				Version:     config.String("test-version"),
				Description: config.String("test-description"),
				BufferPeriod: &oapigen.BufferPeriod{
					Enabled: config.Bool(true),
					Max:     config.String("20s"),
					Min:     config.String("5s"),
				},
				Enabled:     config.Bool(true),
				Condition:   &oapigen.Condition{},
				ModuleInput: &oapigen.ModuleInput{},
				Services:    &[]string{"api", "web"},
				Providers:   &[]string{"test-provider-1", "test-provider-2"},
			},
		},
		{
			name: "with_services_condition_regexp",
			taskConfig: config.TaskConfig{
				Condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Regexp:             config.String("^web.*"),
						Datacenter:         config.String("dc"),
						Namespace:          config.String("ns"),
						Filter:             config.String("filter"),
						CTSUserDefinedMeta: map[string]string{"key": "value"},
					},
					UseAsModuleInput: config.Bool(false),
				},
			},
			expectedRequest: oapigen.Task{
				Condition: &oapigen.Condition{
					Services: &oapigen.ServicesCondition{
						Regexp:           config.String("^web.*"),
						UseAsModuleInput: config.Bool(false),
					},
				},
			},
		},
		{
			name: "with_services_condition_names",
			taskConfig: config.TaskConfig{
				Condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Names:              []string{"api", "web"},
						Datacenter:         config.String(""),
						Namespace:          config.String(""),
						Filter:             config.String(""),
						CTSUserDefinedMeta: map[string]string{},
					},
					UseAsModuleInput: config.Bool(false),
				},
			},
			expectedRequest: oapigen.Task{
				Condition: &oapigen.Condition{
					Services: &oapigen.ServicesCondition{
						Names:            &[]string{"api", "web"},
						UseAsModuleInput: config.Bool(false),
					},
				},
			},
		},
		{
			name: "with_catalog_services_condition",
			taskConfig: config.TaskConfig{
				Condition: &config.CatalogServicesConditionConfig{
					CatalogServicesMonitorConfig: config.CatalogServicesMonitorConfig{
						Regexp:           config.String(".*"),
						UseAsModuleInput: config.Bool(true),
						Datacenter:       config.String("dc2"),
						Namespace:        config.String("ns2"),
						NodeMeta: map[string]string{
							"key1": "value1",
							"key2": "value2",
						},
					},
				},
			},
			expectedRequest: oapigen.Task{
				Condition: &oapigen.Condition{
					CatalogServices: &oapigen.CatalogServicesCondition{
						Regexp:           ".*",
						UseAsModuleInput: config.Bool(true),
						Datacenter:       config.String("dc2"),
						Namespace:        config.String("ns2"),
						NodeMeta: &oapigen.CatalogServicesCondition_NodeMeta{
							AdditionalProperties: map[string]string{
								"key1": "value1",
								"key2": "value2",
							},
						},
					},
				},
			},
		},
		{
			name: "with_consul_kv_condition",
			taskConfig: config.TaskConfig{
				Condition: &config.ConsulKVConditionConfig{
					ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
						Path:       config.String("key-path"),
						Recurse:    config.Bool(true),
						Datacenter: config.String("dc2"),
						Namespace:  config.String("ns2"),
					},
					UseAsModuleInput: config.Bool(true),
				},
			},
			expectedRequest: oapigen.Task{
				Condition: &oapigen.Condition{
					ConsulKv: &oapigen.ConsulKVCondition{
						Path:             "key-path",
						Recurse:          config.Bool(true),
						Datacenter:       config.String("dc2"),
						Namespace:        config.String("ns2"),
						UseAsModuleInput: config.Bool(true),
					},
				},
			},
		},
		{
			name: "with_schedule_condition",
			taskConfig: config.TaskConfig{
				Condition: &config.ScheduleConditionConfig{Cron: config.String("*/10 * * * * * *")},
			},
			expectedRequest: oapigen.Task{
				Condition: &oapigen.Condition{
					Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
				},
			},
		},
		{
			name: "with_module_inputs",
			taskConfig: config.TaskConfig{
				ModuleInputs: &config.ModuleInputConfigs{
					&config.ServicesModuleInputConfig{
						ServicesMonitorConfig: config.ServicesMonitorConfig{
							Regexp:             config.String("^api$"),
							Datacenter:         config.String("dc"),
							Namespace:          config.String("ns"),
							Filter:             config.String("filter"),
							CTSUserDefinedMeta: map[string]string{"key": "value"},
						},
					},
					&config.ConsulKVModuleInputConfig{
						ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
							Path:       config.String("fake-path"),
							Recurse:    config.Bool(false),
							Datacenter: config.String("dc"),
							Namespace:  config.String("ns"),
						},
					},
				},
			},
			expectedRequest: oapigen.Task{
				ModuleInput: &oapigen.ModuleInput{
					Services: &oapigen.ServicesModuleInput{
						Regexp: config.String("^api$"),
					},
					ConsulKv: &oapigen.ConsulKVModuleInput{
						Path:       "fake-path",
						Recurse:    config.Bool(false),
						Datacenter: config.String("dc"),
						Namespace:  config.String("ns"),
					},
				},
			},
		},
		{
			name: "with_module_input_services_names",
			// separate test-case for services names because it can't be
			// combined with 'with_module_inputs' test case.
			// oapigen.ModuleInput.Services can be set with only 1 Services
			// module input
			taskConfig: config.TaskConfig{
				ModuleInputs: &config.ModuleInputConfigs{
					&config.ServicesModuleInputConfig{
						ServicesMonitorConfig: config.ServicesMonitorConfig{
							Names:              []string{"api"},
							Datacenter:         config.String("dc"),
							Namespace:          config.String("ns"),
							Filter:             config.String("filter"),
							CTSUserDefinedMeta: map[string]string{"key": "value"},
						},
					},
				},
			},
			expectedRequest: oapigen.Task{
				ModuleInput: &oapigen.ModuleInput{
					Services: &oapigen.ServicesModuleInput{
						Names: &[]string{"api"},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := oapigenTaskFromConfigTask(tc.taskConfig)
			assert.Equal(t, tc.expectedRequest, actual)
		})
	}
}

func TestTaskRequest_ToRequestTaskConfig(t *testing.T) {
	cases := []struct {
		name               string
		request            *TaskRequest
		taskConfigExpected config.TaskConfig
	}{
		{
			name: "minimum_required_only",
			request: &TaskRequest{
				Name:     "test-name",
				Module:   "path",
				Services: &[]string{"api", "web"},
			},
			taskConfigExpected: config.TaskConfig{
				Name:     config.String("test-name"),
				Services: []string{"api", "web"},
				Module:   config.String("path"),
			},
		},
		{
			name: "basic_fields_filled",
			request: &TaskRequest{
				Description: config.String("test-description"),
				Name:        "test-name",
				Services:    &[]string{"api", "web"},
				Providers:   &[]string{"test-provider-1", "test-provider-2"},
				Module:      "path",
				Version:     config.String("test-version"),
				BufferPeriod: &oapigen.BufferPeriod{
					Enabled: config.Bool(true),
					Max:     config.String("5m"),
					Min:     config.String("30s"),
				},
				Enabled: config.Bool(true),
			},
			taskConfigExpected: config.TaskConfig{
				Description: config.String("test-description"),
				Name:        config.String("test-name"),
				Providers:   []string{"test-provider-1", "test-provider-2"},
				Services:    []string{"api", "web"},
				Module:      config.String("path"),
				Version:     config.String("test-version"),
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(true),
					Max:     config.TimeDuration(time.Duration(5 * time.Minute)),
					Min:     config.TimeDuration(time.Duration(30 * time.Second)),
				},
				Enabled: config.Bool(true),
			},
		},
		{
			name: "with_services_condition_regexp",
			request: &TaskRequest{
				Name:    "task",
				Module:  "path",
				Enabled: config.Bool(true),
				Condition: &oapigen.Condition{
					Services: &oapigen.ServicesCondition{
						Regexp:           config.String("^web.*"),
						UseAsModuleInput: config.Bool(false),
					},
				},
			},
			taskConfigExpected: config.TaskConfig{
				Name:    config.String("task"),
				Module:  config.String("path"),
				Enabled: config.Bool(true),
				Condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Regexp: config.String("^web.*"),
					},
					UseAsModuleInput: config.Bool(false),
				},
			},
		},
		{
			name: "with_services_condition_names",
			request: &TaskRequest{
				Name:    "task",
				Module:  "path",
				Enabled: config.Bool(true),
				Condition: &oapigen.Condition{
					Services: &oapigen.ServicesCondition{
						Names:            &[]string{"api", "web"},
						UseAsModuleInput: config.Bool(false),
					},
				},
			},
			taskConfigExpected: config.TaskConfig{
				Name:    config.String("task"),
				Module:  config.String("path"),
				Enabled: config.Bool(true),
				Condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Names: []string{"api", "web"},
					},
					UseAsModuleInput: config.Bool(false),
				},
			},
		},
		{
			name: "with_catalog_services_condition",
			request: &TaskRequest{
				Name:   "task",
				Module: "path",
				Condition: &oapigen.Condition{
					CatalogServices: &oapigen.CatalogServicesCondition{
						Regexp:           ".*",
						UseAsModuleInput: config.Bool(true),
						Datacenter:       config.String("dc2"),
						Namespace:        config.String("ns2"),
						NodeMeta: &oapigen.CatalogServicesCondition_NodeMeta{
							AdditionalProperties: map[string]string{
								"key1": "value1",
								"key2": "value2",
							},
						},
					},
				},
			},
			taskConfigExpected: config.TaskConfig{
				Name:   config.String("task"),
				Module: config.String("path"),
				Condition: &config.CatalogServicesConditionConfig{
					config.CatalogServicesMonitorConfig{
						Regexp:           config.String(".*"),
						UseAsModuleInput: config.Bool(true),
						Datacenter:       config.String("dc2"),
						Namespace:        config.String("ns2"),
						NodeMeta: map[string]string{
							"key1": "value1",
							"key2": "value2",
						},
					},
				},
			},
		},
		{
			name: "with_consul_kv_condition",
			request: &TaskRequest{
				Name:     "task",
				Module:   "path",
				Services: &[]string{"api", "web"},
				Condition: &oapigen.Condition{
					ConsulKv: &oapigen.ConsulKVCondition{
						Path:             "key-path",
						Recurse:          config.Bool(true),
						Datacenter:       config.String("dc2"),
						Namespace:        config.String("ns2"),
						UseAsModuleInput: config.Bool(true),
					},
				},
			},
			taskConfigExpected: config.TaskConfig{
				Name:     config.String("task"),
				Services: []string{"api", "web"},
				Module:   config.String("path"),
				Condition: &config.ConsulKVConditionConfig{
					ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
						Path:       config.String("key-path"),
						Recurse:    config.Bool(true),
						Datacenter: config.String("dc2"),
						Namespace:  config.String("ns2"),
					},
					UseAsModuleInput: config.Bool(true),
				},
			},
		},
		{
			name: "with_schedule_condition",
			request: &TaskRequest{
				Name:     "task",
				Module:   "path",
				Services: &[]string{"api", "web"},
				Condition: &oapigen.Condition{
					Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
				},
			},
			taskConfigExpected: config.TaskConfig{
				Name:      config.String("task"),
				Services:  []string{"api", "web"},
				Module:    config.String("path"),
				Condition: &config.ScheduleConditionConfig{Cron: config.String("*/10 * * * * * *")},
			},
		},
		{
			name: "with_module_inputs",
			request: &TaskRequest{
				Name:   "task",
				Module: "path",
				Condition: &oapigen.Condition{
					Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
				},
				ModuleInput: &oapigen.ModuleInput{
					Services: &oapigen.ServicesModuleInput{
						Regexp: config.String("^api$"),
					},
					ConsulKv: &oapigen.ConsulKVModuleInput{
						Path:       "fake-path",
						Recurse:    config.Bool(true),
						Datacenter: config.String("dc"),
						Namespace:  config.String("ns"),
					},
				},
			},
			taskConfigExpected: config.TaskConfig{
				Name:   config.String("task"),
				Module: config.String("path"),
				Condition: &config.ScheduleConditionConfig{
					Cron: config.String("*/10 * * * * * *"),
				},
				ModuleInputs: &config.ModuleInputConfigs{
					&config.ServicesModuleInputConfig{
						ServicesMonitorConfig: config.ServicesMonitorConfig{
							Regexp: config.String("^api$"),
						},
					},
					&config.ConsulKVModuleInputConfig{
						ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
							Path:       config.String("fake-path"),
							Recurse:    config.Bool(true),
							Datacenter: config.String("dc"),
							Namespace:  config.String("ns"),
						},
					},
				},
			},
		},
		{
			name: "with_module_input_services_names",
			// separate test-case for services names because it can't be
			// combined with 'with_module_inputs' test case.
			// oapigen.ModuleInput.Services can be set with only 1 Services
			// module input
			request: &TaskRequest{
				Name:   "task",
				Module: "path",
				Condition: &oapigen.Condition{
					Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
				},
				ModuleInput: &oapigen.ModuleInput{
					Services: &oapigen.ServicesModuleInput{
						Names: &[]string{"api"},
					},
				},
			},
			taskConfigExpected: config.TaskConfig{
				Name:   config.String("task"),
				Module: config.String("path"),
				Condition: &config.ScheduleConditionConfig{
					Cron: config.String("*/10 * * * * * *"),
				},
				ModuleInputs: &config.ModuleInputConfigs{
					&config.ServicesModuleInputConfig{
						ServicesMonitorConfig: config.ServicesMonitorConfig{
							Names: []string{"api"},
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := tc.request.ToTaskConfig()
			require.NoError(t, err)
			assert.Equal(t, tc.taskConfigExpected, actual)
		})
	}
}

func TestTaskRequest_ToRequestTaskConfig_Error(t *testing.T) {
	cases := []struct {
		name     string
		request  *TaskRequest
		contains string
	}{
		{
			name: "invalid conversion",
			request: &TaskRequest{
				Name:     "test-name",
				Services: &[]string{"api", "web"},
				BufferPeriod: &oapigen.BufferPeriod{
					Max: config.String("invalid"),
				},
			},
			contains: "invalid duration",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.request.ToTaskConfig()
			fmt.Println(err)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.contains)
		})
	}
}

func TestTaskResponse_String(t *testing.T) {
	resp := TaskResponse{
		RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
		Task: &oapigen.Task{
			Name:    "task",
			Module:  "path",
			Version: config.String(""),
			BufferPeriod: &oapigen.BufferPeriod{
				Enabled: config.Bool(false),
				Max:     config.String("0s"),
				Min:     config.String("0s"),
			},
			Enabled: config.Bool(true),
			Condition: &oapigen.Condition{
				CatalogServices: &oapigen.CatalogServicesCondition{
					Regexp:           ".*",
					UseAsModuleInput: config.Bool(true),
					Datacenter:       config.String("dc2"),
					Namespace:        config.String("ns2"),
					NodeMeta: &oapigen.CatalogServicesCondition_NodeMeta{
						AdditionalProperties: map[string]string{
							"key1": "value1",
							"key2": "value2",
						},
					},
				},
			},
			ModuleInput: &oapigen.ModuleInput{
				Services: &oapigen.ServicesModuleInput{
					Regexp: config.String(""),
				},
			},
		},
	}

	actual := fmt.Sprint(resp)
	expected := `{"request_id":"e9926514-79b8-a8fc-8761-9b6aaccf1e15",` +
		`"task":{"buffer_period":{"enabled":false,"max":"0s","min":"0s"},` +
		`"condition":{"catalog_services":{"datacenter":"dc2","namespace":"ns2",` +
		`"node_meta":{"key1":"value1","key2":"value2"},"regexp":".*",` +
		`"use_as_module_input":true}},"enabled":true,"module":"path",` +
		`"module_input":{"services":{"regexp":""}},"name":"task",` +
		`"version":""}}`
	require.Equal(t, expected, actual)
}

// Test only bare minimum, task conversion scenarios covered in
// TestRequest_oapigenTaskFromConfigTask
func TestTaskResponse_taskResponseFromTaskConfig(t *testing.T) {
	tc := struct {
		taskConfig       config.TaskConfig
		expectedResponse TaskResponse
	}{
		taskConfig: config.TaskConfig{},
		expectedResponse: TaskResponse{
			RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
			Task:      &oapigen.Task{},
		},
	}

	actual := taskResponseFromTaskConfig(tc.taskConfig, "e9926514-79b8-a8fc-8761-9b6aaccf1e15")
	assert.Equal(t, tc.expectedResponse, actual)
}
