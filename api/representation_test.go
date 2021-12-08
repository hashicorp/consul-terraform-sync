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
    "source": "./example-module",
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
	expected := `{"buffer_period":{"enabled":true,"max":"0s","min":"0s"},"description":"Writes the service name, id, and IP address to a file","enabled":true,"name":"new-example-task","providers":["local"],"services":["api"],"source":"./example-module"}`
	require.Equal(t, expected, actual)
}

func TestTaskRequest_ToConfigTaskConfig(t *testing.T) {
	cases := []struct {
		name               string
		request            *taskRequest
		taskConfigExpected *config.TaskConfig
	}{
		{
			name: "minimum_required_only",
			request: &taskRequest{
				Name:       "test-name",
				Source:     "test-source",
				Services:   &[]string{"api", "web"},
				WorkingDir: config.String("sync-tasks"),
			},
			taskConfigExpected: &config.TaskConfig{
				Description:  config.String(""),
				Name:         config.String("test-name"),
				Providers:    []string{},
				Services:     []string{"api", "web"},
				Source:       config.String("test-source"),
				VarFiles:     []string{},
				Version:      config.String(""),
				TFVersion:    config.String(""),
				BufferPeriod: config.DefaultBufferPeriodConfig(),
				Enabled:      config.Bool(true),
				Condition:    config.EmptyConditionConfig(),
				WorkingDir:   config.String("sync-tasks"),
				SourceInput:  config.EmptySourceInputConfig(),
			},
		},
		{
			name: "basic_fields_filled",
			request: &taskRequest{
				Description: config.String("test-description"),
				Name:        "test-name",
				Services:    &[]string{"api", "web"},
				Providers:   &[]string{"test-provider-1", "test-provider-2"},
				Source:      "test-source",
				Version:     config.String("test-version"),
				Enabled:     config.Bool(true),
				WorkingDir:  config.String("sync-tasks"),
			},
			taskConfigExpected: &config.TaskConfig{
				Description:  config.String("test-description"),
				Name:         config.String("test-name"),
				Providers:    []string{"test-provider-1", "test-provider-2"},
				Services:     []string{"api", "web"},
				Source:       config.String("test-source"),
				VarFiles:     []string{},
				Version:      config.String("test-version"),
				TFVersion:    config.String(""),
				BufferPeriod: config.DefaultBufferPeriodConfig(),
				Enabled:      config.Bool(true),
				Condition:    config.EmptyConditionConfig(),
				WorkingDir:   config.String("sync-tasks"),
				SourceInput:  config.EmptySourceInputConfig(),
			},
		},
		{
			name: "with_services_condition",
			request: &taskRequest{
				Name:   "task",
				Source: "test-source",
				BufferPeriod: &oapigen.BufferPeriod{
					Enabled: config.Bool(false),
					Max:     config.String("0s"),
					Min:     config.String("0s"),
				},
				Enabled: config.Bool(true),
				Condition: &oapigen.Condition{
					Services: &oapigen.ServicesCondition{
						Regexp: config.String("^web.*"),
					},
				},
				WorkingDir: config.String("sync-tasks/task"),
			},
			taskConfigExpected: &config.TaskConfig{
				Description: config.String(""),
				Name:        config.String("task"),
				Providers:   []string{},
				Services:    []string{},
				Source:      config.String("test-source"),
				VarFiles:    []string{},
				Version:     config.String(""),
				TFVersion:   config.String(""),
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(false),
					Min:     config.TimeDuration(0 * time.Second),
					Max:     config.TimeDuration(0 * time.Second),
				},
				Enabled: config.Bool(true),
				Condition: &config.ServicesConditionConfig{
					config.ServicesMonitorConfig{
						Regexp:             config.String("^web.*"),
						Datacenter:         config.String(""),
						Namespace:          config.String(""),
						Filter:             config.String(""),
						CTSUserDefinedMeta: map[string]string{},
					},
				},
				WorkingDir:  config.String("sync-tasks/task"),
				SourceInput: config.EmptySourceInputConfig(),
			},
		},
		{
			name: "with_catalog_services_condition",
			request: &taskRequest{
				Name:   "task",
				Source: "test-source",
				BufferPeriod: &oapigen.BufferPeriod{
					Enabled: config.Bool(false),
					Max:     config.String("0s"),
					Min:     config.String("0s"),
				},
				Enabled: config.Bool(true),
				Condition: &oapigen.Condition{
					CatalogServices: &oapigen.CatalogServicesCondition{
						Regexp:            config.String(".*"),
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
				WorkingDir: config.String("sync-tasks/task"),
			},
			taskConfigExpected: &config.TaskConfig{
				Description: config.String(""),
				Name:        config.String("task"),
				Providers:   []string{},
				Services:    []string{},
				Source:      config.String("test-source"),
				VarFiles:    []string{},
				Version:     config.String(""),
				TFVersion:   config.String(""),
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(false),
					Min:     config.TimeDuration(0 * time.Second),
					Max:     config.TimeDuration(0 * time.Second),
				},
				Enabled: config.Bool(true),
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
				WorkingDir:  config.String("sync-tasks/task"),
				SourceInput: config.EmptySourceInputConfig(),
			},
		},
		{
			name: "with_consul_kv_condition",
			request: &taskRequest{
				Name:     "task",
				Source:   "test-source",
				Services: &[]string{"api", "web"},
				BufferPeriod: &oapigen.BufferPeriod{
					Enabled: config.Bool(false),
					Max:     config.String("0s"),
					Min:     config.String("0s"),
				},
				Enabled: config.Bool(true),
				Condition: &oapigen.Condition{
					ConsulKv: &oapigen.ConsulKVCondition{
						Path:              "key-path",
						Recurse:           config.Bool(true),
						Datacenter:        config.String("dc2"),
						Namespace:         config.String("ns2"),
						SourceIncludesVar: config.Bool(true),
					},
				},
				WorkingDir: config.String("sync-tasks/task"),
			},
			taskConfigExpected: &config.TaskConfig{
				Description: config.String(""),
				Name:        config.String("task"),
				Providers:   []string{},
				Services:    []string{"api", "web"},
				Source:      config.String("test-source"),
				VarFiles:    []string{},
				Version:     config.String(""),
				TFVersion:   config.String(""),
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(false),
					Min:     config.TimeDuration(0 * time.Second),
					Max:     config.TimeDuration(0 * time.Second),
				},
				Enabled: config.Bool(true),
				Condition: &config.ConsulKVConditionConfig{
					ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
						Path:       config.String("key-path"),
						Recurse:    config.Bool(true),
						Datacenter: config.String("dc2"),
						Namespace:  config.String("ns2"),
					},
					SourceIncludesVar: config.Bool(true),
				},
				WorkingDir:  config.String("sync-tasks/task"),
				SourceInput: config.EmptySourceInputConfig(),
			},
		},
		{
			name: "with_schedule_condition",
			request: &taskRequest{
				Name:     "task",
				Source:   "test-source",
				Services: &[]string{"api", "web"},
				BufferPeriod: &oapigen.BufferPeriod{
					Enabled: config.Bool(false),
					Max:     config.String("0s"),
					Min:     config.String("0s"),
				},
				Enabled: config.Bool(true),
				Condition: &oapigen.Condition{
					Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
				},
				WorkingDir: config.String("sync-tasks/task"),
			},
			taskConfigExpected: &config.TaskConfig{
				Description: config.String(""),
				Name:        config.String("task"),
				Providers:   []string{},
				Services:    []string{"api", "web"},
				Source:      config.String("test-source"),
				VarFiles:    []string{},
				Version:     config.String(""),
				TFVersion:   config.String(""),
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(false),
					Min:     config.TimeDuration(0 * time.Second),
					Max:     config.TimeDuration(0 * time.Second),
				},
				Enabled:     config.Bool(true),
				Condition:   &config.ScheduleConditionConfig{config.String("*/10 * * * * * *")},
				WorkingDir:  config.String("sync-tasks/task"),
				SourceInput: config.EmptySourceInputConfig(),
			},
		},
		{
			name: "with_services_source_input",
			request: &taskRequest{
				Name:   "task",
				Source: "test-source",
				BufferPeriod: &oapigen.BufferPeriod{
					Enabled: config.Bool(false),
					Max:     config.String("0s"),
					Min:     config.String("0s"),
				},
				Enabled: config.Bool(true),
				Condition: &oapigen.Condition{
					Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
				},
				WorkingDir: config.String("sync-tasks/task"),
				SourceInput: &oapigen.SourceInput{
					Services: &oapigen.ServicesSourceInput{
						Regexp: config.String("^api$"),
					}},
			},
			taskConfigExpected: &config.TaskConfig{
				Description: config.String(""),
				Name:        config.String("task"),
				Services:    []string{},
				Providers:   []string{},
				Source:      config.String("test-source"),
				VarFiles:    []string{},
				Version:     config.String(""),
				TFVersion:   config.String(""),
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(false),
					Min:     config.TimeDuration(0 * time.Second),
					Max:     config.TimeDuration(0 * time.Second),
				},
				Enabled:    config.Bool(true),
				Condition:  &config.ScheduleConditionConfig{config.String("*/10 * * * * * *")},
				WorkingDir: config.String("sync-tasks/task"),
				SourceInput: &config.ServicesSourceInputConfig{
					config.ServicesMonitorConfig{
						Regexp:             config.String("^api$"),
						Datacenter:         config.String(""),
						Namespace:          config.String(""),
						Filter:             config.String(""),
						CTSUserDefinedMeta: map[string]string{},
					}},
			},
		},
		{
			name: "with_consul_kv_source_input",
			request: &taskRequest{
				Name:     "task",
				Source:   "test-source",
				Services: &[]string{"api", "web"},
				BufferPeriod: &oapigen.BufferPeriod{
					Enabled: config.Bool(false),
					Max:     config.String("0s"),
					Min:     config.String("0s"),
				},
				Enabled: config.Bool(true),
				Condition: &oapigen.Condition{
					Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
				},
				WorkingDir: config.String("sync-tasks/task"),
				SourceInput: &oapigen.SourceInput{
					ConsulKv: &oapigen.ConsulKVSourceInput{
						Path:       "fake-path",
						Recurse:    config.Bool(false),
						Datacenter: config.String(""),
						Namespace:  config.String(""),
					}},
			},
			taskConfigExpected: &config.TaskConfig{
				Description: config.String(""),
				Name:        config.String("task"),
				Services:    []string{"api", "web"},
				Providers:   []string{},
				Source:      config.String("test-source"),
				VarFiles:    []string{},
				Version:     config.String(""),
				TFVersion:   config.String(""),
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(false),
					Min:     config.TimeDuration(0 * time.Second),
					Max:     config.TimeDuration(0 * time.Second),
				},
				Enabled:    config.Bool(true),
				Condition:  &config.ScheduleConditionConfig{config.String("*/10 * * * * * *")},
				WorkingDir: config.String("sync-tasks/task"),
				SourceInput: &config.ConsulKVSourceInputConfig{
					config.ConsulKVMonitorConfig{
						Path:       config.String("fake-path"),
						Recurse:    config.Bool(false),
						Datacenter: config.String(""),
						Namespace:  config.String(""),
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := tc.request.ToConfigTaskConfig(config.DefaultBufferPeriodConfig(), "task-sync")
			require.NoError(t, err)
			assert.Equal(t, actual, tc.taskConfigExpected)
		})
	}
}

func TestTaskRequest_ToConfigTaskConfig_Error(t *testing.T) {
	cases := []struct {
		name     string
		request  *taskRequest
		contains string
	}{
		{
			name: "invalid config - missing required field",
			request: &taskRequest{
				Name:       "test-name",
				Services:   &[]string{"api", "web"},
				WorkingDir: config.String("sync-tasks"),
			},
			contains: "source for the task is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.request.ToConfigTaskConfig(config.DefaultBufferPeriodConfig(), "task-sync")
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
			Source:  "test-source",
			Version: config.String(""),
			BufferPeriod: &oapigen.BufferPeriod{
				Enabled: config.Bool(false),
				Max:     config.String("0s"),
				Min:     config.String("0s"),
			},
			Enabled: config.Bool(true),
			Condition: &oapigen.Condition{
				CatalogServices: &oapigen.CatalogServicesCondition{
					Regexp:            config.String(".*"),
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
			WorkingDir: config.String("sync-tasks/task"),
		},
	}

	actual := fmt.Sprintf("%s", resp)
	expected := `{"request_id":"e9926514-79b8-a8fc-8761-9b6aaccf1e15",` +
		`"task":{"buffer_period":{"enabled":false,"max":"0s","min":"0s"},` +
		`"condition":{"catalog_services":{"datacenter":"dc2","namespace":"ns2",` +
		`"node_meta":{"key1":"value1","key2":"value2"},"regexp":".*",` +
		`"source_includes_var":true}},"enabled":true,"name":"task","source":"test-source",` +
		`"source_input":{"services":{"regexp":""}},"version":"","working_dir":"sync-tasks/task"}}`
	require.Equal(t, expected, actual)
}

func TestTaskResponse_taskResponseFromConfigTaskConfig(t *testing.T) {
	cases := []struct {
		name             string
		taskConfig       *config.TaskConfig
		expectedResponse taskResponse
	}{
		{
			name: "minimum_required_only",
			taskConfig: &config.TaskConfig{
				Description:  config.String(""),
				Name:         config.String("test-name"),
				Providers:    []string{},
				Services:     []string{"api", "web"},
				Source:       config.String("test-source"),
				VarFiles:     []string{},
				Version:      config.String(""),
				BufferPeriod: config.DefaultBufferPeriodConfig(),
				Enabled:      config.Bool(true),
				Condition:    config.EmptyConditionConfig(),
				WorkingDir:   config.String("sync-tasks"),
				SourceInput:  config.EmptySourceInputConfig(),
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:        "test-name",
					Source:      "test-source",
					Version:     config.String(""),
					Description: config.String(""),
					BufferPeriod: &oapigen.BufferPeriod{
						Enabled: config.Bool(true),
						Max:     config.String("20s"),
						Min:     config.String("5s"),
					},
					Enabled:     config.Bool(true),
					Condition:   &oapigen.Condition{},
					SourceInput: &oapigen.SourceInput{},
					WorkingDir:  config.String("sync-tasks"),
					Services:    &[]string{"api", "web"},
					Providers:   &[]string{},
				},
			},
		},
		{
			name: "basic_fields_filled",
			taskConfig: &config.TaskConfig{
				Description:  config.String("test-description"),
				Name:         config.String("test-name"),
				Providers:    []string{"test-provider-1", "test-provider-2"},
				Services:     []string{"api", "web"},
				Source:       config.String("test-source"),
				VarFiles:     []string{},
				TFVersion:    config.String(""),
				Version:      config.String("test-version"),
				BufferPeriod: config.DefaultBufferPeriodConfig(),
				Enabled:      config.Bool(true),
				Condition:    config.EmptyConditionConfig(),
				WorkingDir:   config.String("sync-tasks"),
				SourceInput:  config.EmptySourceInputConfig(),
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:        "test-name",
					Source:      "test-source",
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
					WorkingDir:  config.String("sync-tasks"),
					Services:    &[]string{"api", "web"},
					Providers:   &[]string{"test-provider-1", "test-provider-2"},
				},
			},
		},
		{
			name: "with_services_condition",
			taskConfig: &config.TaskConfig{
				Description:  config.String(""),
				Name:         config.String("task"),
				Providers:    []string{},
				Services:     []string{},
				Source:       config.String("test-source"),
				VarFiles:     []string{},
				Version:      config.String(""),
				TFVersion:    config.String(""),
				BufferPeriod: config.DefaultBufferPeriodConfig(),
				Enabled:      config.Bool(true),
				Condition: &config.ServicesConditionConfig{
					config.ServicesMonitorConfig{
						Regexp:             config.String("^web.*"),
						Datacenter:         config.String(""),
						Namespace:          config.String(""),
						Filter:             config.String(""),
						CTSUserDefinedMeta: map[string]string{},
					},
				},
				WorkingDir:  config.String("sync-tasks"),
				SourceInput: config.EmptySourceInputConfig(),
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:        "task",
					Source:      "test-source",
					Version:     config.String(""),
					Description: config.String(""),
					BufferPeriod: &oapigen.BufferPeriod{
						Enabled: config.Bool(true),
						Max:     config.String("20s"),
						Min:     config.String("5s"),
					},
					Enabled: config.Bool(true),
					Condition: &oapigen.Condition{
						Services: &oapigen.ServicesCondition{
							Regexp: config.String("^web.*"),
						},
					},
					SourceInput: &oapigen.SourceInput{},
					WorkingDir:  config.String("sync-tasks"),
					Services:    &[]string{},
					Providers:   &[]string{},
				},
			},
		},
		{
			name: "with_catalog_services_condition",
			taskConfig: &config.TaskConfig{
				Description:  config.String(""),
				Name:         config.String("task"),
				Providers:    []string{},
				Services:     []string{},
				Source:       config.String("test-source"),
				VarFiles:     []string{},
				Version:      config.String(""),
				TFVersion:    config.String(""),
				BufferPeriod: config.DefaultBufferPeriodConfig(),
				Enabled:      config.Bool(true),
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
				WorkingDir:  config.String("sync-tasks"),
				SourceInput: config.EmptySourceInputConfig(),
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:        "task",
					Source:      "test-source",
					Version:     config.String(""),
					Description: config.String(""),
					BufferPeriod: &oapigen.BufferPeriod{
						Enabled: config.Bool(true),
						Max:     config.String("20s"),
						Min:     config.String("5s"),
					},
					Enabled: config.Bool(true),
					Condition: &oapigen.Condition{
						CatalogServices: &oapigen.CatalogServicesCondition{
							Regexp:            config.String(".*"),
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
					SourceInput: &oapigen.SourceInput{},
					WorkingDir:  config.String("sync-tasks"),
					Services:    &[]string{},
					Providers:   &[]string{},
				},
			},
		},
		{
			name: "with_consul_kv_condition",
			taskConfig: &config.TaskConfig{
				Description:  config.String(""),
				Name:         config.String("task"),
				Providers:    []string{},
				Services:     []string{"api", "web"},
				Source:       config.String("test-source"),
				VarFiles:     []string{},
				Version:      config.String(""),
				TFVersion:    config.String(""),
				BufferPeriod: config.DefaultBufferPeriodConfig(),
				Enabled:      config.Bool(true),
				Condition: &config.ConsulKVConditionConfig{
					ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
						Path:       config.String("key-path"),
						Recurse:    config.Bool(true),
						Datacenter: config.String("dc2"),
						Namespace:  config.String("ns2"),
					},
					SourceIncludesVar: config.Bool(true),
				},
				WorkingDir:  config.String("sync-tasks"),
				SourceInput: config.EmptySourceInputConfig(),
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:        "task",
					Source:      "test-source",
					Version:     config.String(""),
					Description: config.String(""),
					BufferPeriod: &oapigen.BufferPeriod{
						Enabled: config.Bool(true),
						Max:     config.String("20s"),
						Min:     config.String("5s"),
					},
					Enabled: config.Bool(true),
					Condition: &oapigen.Condition{
						ConsulKv: &oapigen.ConsulKVCondition{
							Path:              "key-path",
							Recurse:           config.Bool(true),
							Datacenter:        config.String("dc2"),
							Namespace:         config.String("ns2"),
							SourceIncludesVar: config.Bool(true),
						},
					},
					SourceInput: &oapigen.SourceInput{},
					WorkingDir:  config.String("sync-tasks"),
					Services:    &[]string{"api", "web"},
					Providers:   &[]string{},
				},
			},
		},
		{
			name: "with_schedule_condition",
			taskConfig: &config.TaskConfig{
				Description:  config.String(""),
				Name:         config.String("task"),
				Providers:    []string{},
				Services:     []string{"api", "web"},
				Source:       config.String("test-source"),
				VarFiles:     []string{},
				Version:      config.String(""),
				TFVersion:    config.String(""),
				BufferPeriod: config.DefaultBufferPeriodConfig(),
				Enabled:      config.Bool(true),
				Condition:    &config.ScheduleConditionConfig{config.String("*/10 * * * * * *")},
				WorkingDir:   config.String("sync-tasks"),
				SourceInput:  config.EmptySourceInputConfig(),
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:        "task",
					Source:      "test-source",
					Version:     config.String(""),
					Description: config.String(""),
					BufferPeriod: &oapigen.BufferPeriod{
						Enabled: config.Bool(true),
						Max:     config.String("20s"),
						Min:     config.String("5s"),
					},
					Enabled: config.Bool(true),
					Condition: &oapigen.Condition{
						Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
					},
					SourceInput: &oapigen.SourceInput{},
					WorkingDir:  config.String("sync-tasks"),
					Services:    &[]string{"api", "web"},
					Providers:   &[]string{},
				},
			},
		},
		{
			name: "with_services_source_input",
			taskConfig: &config.TaskConfig{
				Description:  config.String(""),
				Name:         config.String("task"),
				Services:     []string{},
				Providers:    []string{},
				Source:       config.String("test-source"),
				VarFiles:     []string{},
				Version:      config.String(""),
				TFVersion:    config.String(""),
				BufferPeriod: config.DefaultBufferPeriodConfig(),
				Enabled:      config.Bool(true),
				Condition:    &config.ScheduleConditionConfig{config.String("*/10 * * * * * *")},
				WorkingDir:   config.String("sync-tasks"),
				SourceInput: &config.ServicesSourceInputConfig{
					config.ServicesMonitorConfig{
						Regexp:             config.String("^api$"),
						Datacenter:         config.String(""),
						Namespace:          config.String(""),
						Filter:             config.String(""),
						CTSUserDefinedMeta: map[string]string{},
					}},
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:        "task",
					Source:      "test-source",
					Version:     config.String(""),
					Description: config.String(""),
					BufferPeriod: &oapigen.BufferPeriod{
						Enabled: config.Bool(true),
						Max:     config.String("20s"),
						Min:     config.String("5s"),
					},
					Enabled: config.Bool(true),
					Condition: &oapigen.Condition{
						Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
					},
					SourceInput: &oapigen.SourceInput{
						Services: &oapigen.ServicesSourceInput{
							Regexp: config.String("^api$"),
						},
					},
					WorkingDir: config.String("sync-tasks"),
					Services:   &[]string{},
					Providers:  &[]string{},
				},
			},
		},
		{
			name: "with_consul_kv_source_input",
			taskConfig: &config.TaskConfig{
				Description:  config.String(""),
				Name:         config.String("task"),
				Services:     []string{"api", "web"},
				Providers:    []string{},
				Source:       config.String("test-source"),
				VarFiles:     []string{},
				Version:      config.String(""),
				TFVersion:    config.String(""),
				BufferPeriod: config.DefaultBufferPeriodConfig(),
				Enabled:      config.Bool(true),
				Condition:    &config.ScheduleConditionConfig{config.String("*/10 * * * * * *")},
				WorkingDir:   config.String("sync-tasks"),
				SourceInput: &config.ConsulKVSourceInputConfig{
					config.ConsulKVMonitorConfig{
						Path:       config.String("fake-path"),
						Recurse:    config.Bool(false),
						Datacenter: config.String(""),
						Namespace:  config.String(""),
					},
				},
			},
			expectedResponse: taskResponse{
				RequestId: "e9926514-79b8-a8fc-8761-9b6aaccf1e15",
				Task: &oapigen.Task{
					Name:        "task",
					Source:      "test-source",
					Version:     config.String(""),
					Description: config.String(""),
					BufferPeriod: &oapigen.BufferPeriod{
						Enabled: config.Bool(true),
						Max:     config.String("20s"),
						Min:     config.String("5s"),
					},
					Enabled: config.Bool(true),
					Condition: &oapigen.Condition{
						Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
					},
					SourceInput: &oapigen.SourceInput{
						ConsulKv: &oapigen.ConsulKVSourceInput{
							Path:       "fake-path",
							Recurse:    config.Bool(false),
							Datacenter: config.String(""),
							Namespace:  config.String(""),
						},
					},
					WorkingDir: config.String("sync-tasks"),
					Services:   &[]string{"api", "web"},
					Providers:  &[]string{},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := taskResponseFromConfigTaskConfig(tc.taskConfig, "e9926514-79b8-a8fc-8761-9b6aaccf1e15")
			assert.Equal(t, actual, tc.expectedResponse)
		})
	}
}
