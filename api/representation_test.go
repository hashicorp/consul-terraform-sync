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

func TestTaskRequest_ToRequestTaskConfig(t *testing.T) {
	cases := []struct {
		name               string
		request            *taskRequest
		taskConfigExpected config.TaskConfig
	}{
		{
			name: "minimum_required_only",
			request: &taskRequest{
				Name:       "test-name",
				Source:     "test-source",
				Services:   &[]string{"api", "web"},
				WorkingDir: config.String("sync-tasks"),
			},
			taskConfigExpected: config.TaskConfig{
				Name:       config.String("test-name"),
				Services:   []string{"api", "web"},
				Source:     config.String("test-source"),
				WorkingDir: config.String("sync-tasks"),
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
				BufferPeriod: &oapigen.BufferPeriod{
					Enabled: config.Bool(true),
					Max:     config.String("5m"),
					Min:     config.String("30s"),
				},
				Enabled:    config.Bool(true),
				WorkingDir: config.String("sync-tasks"),
			},
			taskConfigExpected: config.TaskConfig{
				Description: config.String("test-description"),
				Name:        config.String("test-name"),
				Providers:   []string{"test-provider-1", "test-provider-2"},
				Services:    []string{"api", "web"},
				Source:      config.String("test-source"),
				Version:     config.String("test-version"),
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(true),
					Max:     config.TimeDuration(time.Duration(5 * time.Minute)),
					Min:     config.TimeDuration(time.Duration(30 * time.Second)),
				},
				Enabled:    config.Bool(true),
				WorkingDir: config.String("sync-tasks"),
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
			},
			taskConfigExpected: config.TaskConfig{
				Name:   config.String("task"),
				Source: config.String("test-source"),
				BufferPeriod: &config.BufferPeriodConfig{
					Enabled: config.Bool(false),
					Max:     config.TimeDuration(0),
					Min:     config.TimeDuration(0),
				},
				Enabled: config.Bool(true),
				Condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Regexp: config.String("^web.*"),
					},
				},
			},
		},
		{
			name: "with_catalog_services_condition",
			request: &taskRequest{
				Name:   "task",
				Source: "test-source",
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
			},
			taskConfigExpected: config.TaskConfig{
				Name:   config.String("task"),
				Source: config.String("test-source"),
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
				Source:   "test-source",
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
				Source:   config.String("test-source"),
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
				Source:   "test-source",
				Services: &[]string{"api", "web"},
				Condition: &oapigen.Condition{
					Schedule: &oapigen.ScheduleCondition{Cron: "*/10 * * * * * *"},
				},
			},
			taskConfigExpected: config.TaskConfig{
				Name:      config.String("task"),
				Services:  []string{"api", "web"},
				Source:    config.String("test-source"),
				Condition: &config.ScheduleConditionConfig{Cron: config.String("*/10 * * * * * *")},
			},
		},
		{
			name: "with_services_source_input",
			request: &taskRequest{
				Name:   "task",
				Source: "test-source",
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
				Source:    config.String("test-source"),
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
				Source:   "test-source",
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
				Source:    config.String("test-source"),
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
			actual, err := tc.request.ToTaskRequestConfig()
			require.NoError(t, err)
			assert.Equal(t, tc.taskConfigExpected, actual)
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
			name: "invalid conversion",
			request: &taskRequest{
				Name:     "test-name",
				Services: &[]string{"api", "web"},
				BufferPeriod: &oapigen.BufferPeriod{
					Max: config.String("invalid"),
				},
				// TODO test object validation outside of type conversion
			},
			contains: "invalid duration",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.request.ToTaskRequestConfig()
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

	actual := fmt.Sprint(resp)
	expected := `{"request_id":"e9926514-79b8-a8fc-8761-9b6aaccf1e15",` +
		`"task":{"buffer_period":{"enabled":false,"max":"0s","min":"0s"},` +
		`"condition":{"catalog_services":{"datacenter":"dc2","namespace":"ns2",` +
		`"node_meta":{"key1":"value1","key2":"value2"},"regexp":".*",` +
		`"source_includes_var":true}},"enabled":true,"name":"task","source":"test-source",` +
		`"source_input":{"services":{"regexp":""}},"version":"","working_dir":"sync-tasks/task"}}`
	require.Equal(t, expected, actual)
}
