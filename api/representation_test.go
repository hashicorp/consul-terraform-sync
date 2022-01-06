package api

import (
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
	var req taskRequest

	err := json.Unmarshal([]byte(taskReq), &req)
	require.NoError(t, err)

	actual := fmt.Sprintf("%s", req)
	expected := `{"buffer_period":{"enabled":true,"max":"0s","min":"0s"},` +
		`"description":"Writes the service name, id, and IP address to a file",` +
		`"enabled":true,"module":"./example-module","name":"new-example-task",` +
		`"providers":["local"],"services":["api"]}`
	require.Equal(t, expected, actual)
}

func TestTaskRequest_ToRequestTaskConfig(t *testing.T) {
	cases := []struct {
		name               string
		request            *taskRequest
		taskConfigExpected config.TaskConfig
	}{
		{
			name: "minimum_required_only",
			request: &taskRequest{
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
			request: &taskRequest{
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
			request: &taskRequest{
				Name:    "task",
				Module:  "path",
				Enabled: config.Bool(true),
				Condition: &oapigen.Condition{
					Services: &oapigen.ServicesCondition{
						Regexp:            config.String("^web.*"),
						SourceIncludesVar: config.Bool(false),
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
					SourceIncludesVar: config.Bool(false),
				},
			},
		},
		{
			name: "with_services_condition_names",
			request: &taskRequest{
				Name:    "task",
				Module:  "path",
				Enabled: config.Bool(true),
				Condition: &oapigen.Condition{
					Services: &oapigen.ServicesCondition{
						Names:             &[]string{"api", "web"},
						SourceIncludesVar: config.Bool(false),
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
					SourceIncludesVar: config.Bool(false),
				},
			},
		},
		{
			name: "with_catalog_services_condition",
			request: &taskRequest{
				Name:   "task",
				Module: "path",
				Condition: &oapigen.Condition{
					CatalogServices: &oapigen.CatalogServicesCondition{
						Regexp:            ".*",
						SourceIncludesVar: config.Bool(true),
						Datacenter:        config.String("dc2"),
						Namespace:         config.String("ns2"),
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
						Regexp:            config.String(".*"),
						SourceIncludesVar: config.Bool(true),
						Datacenter:        config.String("dc2"),
						Namespace:         config.String("ns2"),
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
			request: &taskRequest{
				Name:     "task",
				Module:   "path",
				Services: &[]string{"api", "web"},
				Condition: &oapigen.Condition{
					ConsulKv: &oapigen.ConsulKVCondition{
						Path:              "key-path",
						Recurse:           config.Bool(true),
						Datacenter:        config.String("dc2"),
						Namespace:         config.String("ns2"),
						SourceIncludesVar: config.Bool(true),
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
					SourceIncludesVar: config.Bool(true),
				},
			},
		},
		{
			name: "with_schedule_condition",
			request: &taskRequest{
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
			name: "with_services_source_input",
			request: &taskRequest{
				Name:   "task",
				Module: "path",
				Condition: &oapigen.Condition{
					Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
				},
				SourceInput: &oapigen.SourceInput{
					Services: &oapigen.ServicesSourceInput{
						Regexp: config.String("^api$"),
					}},
			},
			taskConfigExpected: config.TaskConfig{
				Name:      config.String("task"),
				Module:    config.String("path"),
				Condition: &config.ScheduleConditionConfig{config.String("*/10 * * * * * *")},
				SourceInput: &config.ServicesSourceInputConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Regexp: config.String("^api$"),
					},
				},
			},
		},
		{
			name: "with_consul_kv_source_input",
			request: &taskRequest{
				Name:     "task",
				Module:   "path",
				Services: &[]string{"api", "web"},
				Condition: &oapigen.Condition{
					Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
				},
				SourceInput: &oapigen.SourceInput{
					ConsulKv: &oapigen.ConsulKVSourceInput{
						Path:       "fake-path",
						Recurse:    config.Bool(true),
						Datacenter: config.String("dc"),
						Namespace:  config.String("ns"),
					}},
			},
			taskConfigExpected: config.TaskConfig{
				Name:      config.String("task"),
				Services:  []string{"api", "web"},
				Module:    config.String("path"),
				Condition: &config.ScheduleConditionConfig{Cron: config.String("*/10 * * * * * *")},
				SourceInput: &config.ConsulKVSourceInputConfig{
					config.ConsulKVMonitorConfig{
						Path:       config.String("fake-path"),
						Recurse:    config.Bool(true),
						Datacenter: config.String("dc"),
						Namespace:  config.String("ns"),
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
		request  *taskRequest
		contains string
	}{
		{
			name: "invalid conversion",
			request: &taskRequest{
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
	resp := taskResponse{
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
					Regexp:            ".*",
					SourceIncludesVar: config.Bool(true),
					Datacenter:        config.String("dc2"),
					Namespace:         config.String("ns2"),
					NodeMeta: &oapigen.CatalogServicesCondition_NodeMeta{
						AdditionalProperties: map[string]string{
							"key1": "value1",
							"key2": "value2",
						},
					},
				},
			},
			SourceInput: &oapigen.SourceInput{
				Services: &oapigen.ServicesSourceInput{
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
		`"source_includes_var":true}},"enabled":true,"module":"path","name":"task",` +
		`"source_input":{"services":{"regexp":""}},"version":""}}`
	require.Equal(t, expected, actual)
}

func TestTaskResponse_taskResponseFromTaskConfig(t *testing.T) {
	cases := []struct {
		name             string
		taskConfig       config.TaskConfig
		expectedResponse taskResponse
	}{
		{
			name: "minimum_required_only",
			taskConfig: config.TaskConfig{
				Name:    config.String("test-name"),
				Module:  config.String("path"),
				Enabled: config.Bool(true),
				Condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{Names: []string{"api", "web"}},
				},
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:    "test-name",
					Module:  "path",
					Enabled: config.Bool(true),
					Condition: &oapigen.Condition{
						Services: &oapigen.ServicesCondition{Names: &[]string{"api", "web"}},
					},
				},
			},
		},
		{
			name: "basic_fields_filled",
			taskConfig: config.TaskConfig{
				Description:  config.String("test-description"),
				Name:         config.String("test-name"),
				Providers:    []string{"test-provider-1", "test-provider-2"},
				Services:     []string{"api", "web"},
				Module:       config.String("path"),
				VarFiles:     []string{""},
				TFVersion:    config.String(""),
				Version:      config.String("test-version"),
				BufferPeriod: config.DefaultBufferPeriodConfig(),
				Enabled:      config.Bool(true),
				Condition:    config.EmptyConditionConfig(),
				SourceInput:  config.EmptySourceInputConfig(),
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
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
					SourceInput: &oapigen.SourceInput{},
					Services:    &[]string{"api", "web"},
					Providers:   &[]string{"test-provider-1", "test-provider-2"},
				},
			},
		},
		{
			name: "with_services_condition_regexp",
			taskConfig: config.TaskConfig{
				Name:    config.String("task"),
				Module:  config.String("path"),
				Enabled: config.Bool(true),
				Condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Regexp:             config.String("^web.*"),
						Datacenter:         config.String("dc"),
						Namespace:          config.String("ns"),
						Filter:             config.String("filter"),
						CTSUserDefinedMeta: map[string]string{"key": "value"},
					},
					SourceIncludesVar: config.Bool(false),
				},
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:    "task",
					Module:  "path",
					Enabled: config.Bool(true),
					Condition: &oapigen.Condition{
						Services: &oapigen.ServicesCondition{
							Regexp:            config.String("^web.*"),
							SourceIncludesVar: config.Bool(false),
						},
					},
				},
			},
		},
		{
			name: "with_services_condition_names",
			taskConfig: config.TaskConfig{
				Name:    config.String("task"),
				Module:  config.String("path"),
				Enabled: config.Bool(true),
				Condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Names:              []string{"api", "web"},
						Datacenter:         config.String(""),
						Namespace:          config.String(""),
						Filter:             config.String(""),
						CTSUserDefinedMeta: map[string]string{},
					},
					SourceIncludesVar: config.Bool(false),
				},
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:    "task",
					Module:  "path",
					Enabled: config.Bool(true),
					Condition: &oapigen.Condition{
						Services: &oapigen.ServicesCondition{
							Names:             &[]string{"api", "web"},
							SourceIncludesVar: config.Bool(false),
						},
					},
				},
			},
		},
		{
			name: "with_catalog_services_condition",
			taskConfig: config.TaskConfig{
				Name:    config.String("task"),
				Module:  config.String("path"),
				Enabled: config.Bool(true),
				Condition: &config.CatalogServicesConditionConfig{
					CatalogServicesMonitorConfig: config.CatalogServicesMonitorConfig{
						Regexp:            config.String(".*"),
						SourceIncludesVar: config.Bool(true),
						Datacenter:        config.String("dc2"),
						Namespace:         config.String("ns2"),
						NodeMeta: map[string]string{
							"key1": "value1",
							"key2": "value2",
						},
					},
				},
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:    "task",
					Module:  "path",
					Enabled: config.Bool(true),
					Condition: &oapigen.Condition{
						CatalogServices: &oapigen.CatalogServicesCondition{
							Regexp:            ".*",
							SourceIncludesVar: config.Bool(true),
							Datacenter:        config.String("dc2"),
							Namespace:         config.String("ns2"),
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
		},
		{
			name: "with_consul_kv_condition",
			taskConfig: config.TaskConfig{
				Name:     config.String("task"),
				Services: []string{"api", "web"},
				Module:   config.String("path"),
				Enabled:  config.Bool(true),
				Condition: &config.ConsulKVConditionConfig{
					ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
						Path:       config.String("key-path"),
						Recurse:    config.Bool(true),
						Datacenter: config.String("dc2"),
						Namespace:  config.String("ns2"),
					},
					SourceIncludesVar: config.Bool(true),
				},
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:     "task",
					Services: &[]string{"api", "web"},
					Module:   "path",
					Enabled:  config.Bool(true),
					Condition: &oapigen.Condition{
						ConsulKv: &oapigen.ConsulKVCondition{
							Path:              "key-path",
							Recurse:           config.Bool(true),
							Datacenter:        config.String("dc2"),
							Namespace:         config.String("ns2"),
							SourceIncludesVar: config.Bool(true),
						},
					},
				},
			},
		},
		{
			name: "with_schedule_condition",
			taskConfig: config.TaskConfig{
				Name:      config.String("task"),
				Services:  []string{"api", "web"},
				Module:    config.String("path"),
				Enabled:   config.Bool(true),
				Condition: &config.ScheduleConditionConfig{Cron: config.String("*/10 * * * * * *")},
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:     "task",
					Services: &[]string{"api", "web"},
					Module:   "path",
					Enabled:  config.Bool(true),
					Condition: &oapigen.Condition{
						Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
					},
				},
			},
		},
		{
			name: "with_services_source_input",
			taskConfig: config.TaskConfig{
				Name:      config.String("task"),
				Module:    config.String("path"),
				Enabled:   config.Bool(true),
				Condition: &config.ScheduleConditionConfig{Cron: config.String("*/10 * * * * * *")},
				SourceInput: &config.ServicesSourceInputConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Regexp:             config.String("^api$"),
						Datacenter:         config.String("dc"),
						Namespace:          config.String("ns"),
						Filter:             config.String("filter"),
						CTSUserDefinedMeta: map[string]string{"key": "value"},
					},
				},
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:    "task",
					Module:  "path",
					Enabled: config.Bool(true),
					Condition: &oapigen.Condition{
						Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
					},
					SourceInput: &oapigen.SourceInput{
						Services: &oapigen.ServicesSourceInput{
							Regexp: config.String("^api$"),
						},
					},
				},
			},
		},
		{
			name: "with_consul_kv_source_input",
			taskConfig: config.TaskConfig{
				Name:      config.String("task"),
				Services:  []string{"api", "web"},
				Module:    config.String("path"),
				Enabled:   config.Bool(true),
				Condition: &config.ScheduleConditionConfig{Cron: config.String("*/10 * * * * * *")},
				SourceInput: &config.ConsulKVSourceInputConfig{
					ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
						Path:       config.String("fake-path"),
						Recurse:    config.Bool(false),
						Datacenter: config.String("dc"),
						Namespace:  config.String("ns"),
					},
				},
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:    "task",
					Module:  "path",
					Enabled: config.Bool(true),
					Condition: &oapigen.Condition{
						Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
					},
					SourceInput: &oapigen.SourceInput{
						ConsulKv: &oapigen.ConsulKVSourceInput{
							Path:       "fake-path",
							Recurse:    config.Bool(false),
							Datacenter: config.String("dc"),
							Namespace:  config.String("ns"),
						},
					},
					Services: &[]string{"api", "web"},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := taskResponseFromTaskConfig(tc.taskConfig, "e9926514-79b8-a8fc-8761-9b6aaccf1e15")
			assert.Equal(t, actual, tc.expectedResponse)
		})
	}
}
