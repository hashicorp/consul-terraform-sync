// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"encoding/json"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/config"
)

// TaskRequest is a wrapper around the generated TaskRequest
// this allows for the task request to be extended
type TaskRequest oapigen.TaskRequest

// TaskRequestFromTaskConfig converts a taskRequest object to a Config TaskConfig object.
func TaskRequestFromTaskConfig(tc config.TaskConfig) TaskRequest {
	t := oapigenTaskFromConfigTask(tc)
	return TaskRequest{Task: t}
}

// ToTaskConfig converts a TaskRequest object to a Config TaskConfig object.
func (tr TaskRequest) ToTaskConfig() (config.TaskConfig, error) {
	tc := config.TaskConfig{
		Description: tr.Task.Description,
		Name:        &tr.Task.Name,
		Module:      &tr.Task.Module,
		Version:     tr.Task.Version,
		Enabled:     tr.Task.Enabled,
	}

	if tr.Task.Providers != nil {
		tc.Providers = *tr.Task.Providers
	}

	// Convert module input
	if tr.Task.ModuleInput != nil {
		inputs := make(config.ModuleInputConfigs, 0)
		if tr.Task.ModuleInput.Services != nil {
			input := &config.ServicesModuleInputConfig{
				ServicesMonitorConfig: config.ServicesMonitorConfig{
					Regexp:     tr.Task.ModuleInput.Services.Regexp,
					Datacenter: tr.Task.ModuleInput.Services.Datacenter,
					Namespace:  tr.Task.ModuleInput.Services.Namespace,
					Filter:     tr.Task.ModuleInput.Services.Filter,
				},
			}
			if tr.Task.ModuleInput.Services.Names != nil {
				input.Names = *tr.Task.ModuleInput.Services.Names
			}
			if tr.Task.ModuleInput.Services.CtsUserDefinedMeta != nil {
				input.CTSUserDefinedMeta = tr.Task.ModuleInput.Services.CtsUserDefinedMeta.AdditionalProperties
			}
			inputs = append(inputs, input)
		}
		if tr.Task.ModuleInput.ConsulKv != nil {
			input := &config.ConsulKVModuleInputConfig{
				ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
					Datacenter: tr.Task.ModuleInput.ConsulKv.Datacenter,
					Recurse:    tr.Task.ModuleInput.ConsulKv.Recurse,
					Path:       &tr.Task.ModuleInput.ConsulKv.Path,
					Namespace:  tr.Task.ModuleInput.ConsulKv.Namespace,
				},
			}
			inputs = append(inputs, input)
		}
		tc.ModuleInputs = &inputs
	}

	// Convert condition
	if tr.Task.Condition.Services != nil {
		cond := &config.ServicesConditionConfig{
			ServicesMonitorConfig: config.ServicesMonitorConfig{
				Datacenter: tr.Task.Condition.Services.Datacenter,
				Namespace:  tr.Task.Condition.Services.Namespace,
				Filter:     tr.Task.Condition.Services.Filter,
			},
			UseAsModuleInput: tr.Task.Condition.Services.UseAsModuleInput,
		}
		if tr.Task.Condition.Services.Names != nil && len(*tr.Task.Condition.Services.Names) > 0 {
			cond.Names = *tr.Task.Condition.Services.Names
		} else {
			cond.Regexp = tr.Task.Condition.Services.Regexp
		}
		if tr.Task.Condition.Services.CtsUserDefinedMeta != nil {
			cond.ServicesMonitorConfig.CTSUserDefinedMeta =
				tr.Task.Condition.Services.CtsUserDefinedMeta.AdditionalProperties
		}
		tc.Condition = cond
	} else if tr.Task.Condition.ConsulKv != nil {
		tc.Condition = &config.ConsulKVConditionConfig{
			ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
				Datacenter: tr.Task.Condition.ConsulKv.Datacenter,
				Recurse:    tr.Task.Condition.ConsulKv.Recurse,
				Path:       &tr.Task.Condition.ConsulKv.Path,
				Namespace:  tr.Task.Condition.ConsulKv.Namespace,
			},
			UseAsModuleInput: tr.Task.Condition.ConsulKv.UseAsModuleInput,
		}
	} else if tr.Task.Condition.CatalogServices != nil {
		cond := &config.CatalogServicesConditionConfig{
			CatalogServicesMonitorConfig: config.CatalogServicesMonitorConfig{
				Regexp:           config.String(tr.Task.Condition.CatalogServices.Regexp),
				UseAsModuleInput: tr.Task.Condition.CatalogServices.UseAsModuleInput,
				Datacenter:       tr.Task.Condition.CatalogServices.Datacenter,
				Namespace:        tr.Task.Condition.CatalogServices.Namespace,
			},
		}
		if tr.Task.Condition.CatalogServices.NodeMeta != nil {
			cond.NodeMeta = tr.Task.Condition.CatalogServices.NodeMeta.AdditionalProperties
		}
		tc.Condition = cond
	} else if tr.Task.Condition.Schedule != nil {
		tc.Condition = &config.ScheduleConditionConfig{
			ScheduleMonitorConfig: config.ScheduleMonitorConfig{
				Cron: &tr.Task.Condition.Schedule.Cron,
			},
		}
	}

	if tr.Task.BufferPeriod != nil {
		var max time.Duration
		var err error
		if tr.Task.BufferPeriod.Max != nil {
			max, err = time.ParseDuration(*tr.Task.BufferPeriod.Max)
			if err != nil {
				return config.TaskConfig{}, err
			}
		}

		var min time.Duration
		if tr.Task.BufferPeriod.Min != nil {
			min, err = time.ParseDuration(*tr.Task.BufferPeriod.Min)
			if err != nil {
				return config.TaskConfig{}, err
			}
		}

		tc.BufferPeriod = &config.BufferPeriodConfig{
			Enabled: tr.Task.BufferPeriod.Enabled,
			Max:     &max,
			Min:     &min,
		}
	}

	if tr.Task.Variables != nil {
		tc.Variables = make(map[string]string)
		for k, v := range tr.Task.Variables.AdditionalProperties {
			tc.Variables[k] = v
		}
	}

	// Enterprise
	tc.DeprecatedTFVersion = tr.Task.TerraformVersion
	if tr.Task.TerraformCloudWorkspace != nil {
		tc.TFCWorkspace = &config.TerraformCloudWorkspaceConfig{
			ExecutionMode:    tr.Task.TerraformCloudWorkspace.ExecutionMode,
			AgentPoolID:      tr.Task.TerraformCloudWorkspace.AgentPoolId,
			AgentPoolName:    tr.Task.TerraformCloudWorkspace.AgentPoolName,
			TerraformVersion: tr.Task.TerraformCloudWorkspace.TerraformVersion,
		}
	}

	return tc, nil
}

// String writes out the task request in an easily readable way
// useful for logging
func (tr TaskRequest) String() string {
	data, _ := json.Marshal(tr)
	return string(data)
}

type TasksResponse oapigen.TasksResponse

func tasksResponseFromTaskConfigs(tcs config.TaskConfigs, requestID oapigen.RequestID) TasksResponse {
	tasks := make([]oapigen.Task, len(tcs))
	for i, tc := range tcs {
		tasks[i] = oapigenTaskFromConfigTask(*tc)
	}

	return TasksResponse{
		Tasks:     &tasks,
		RequestId: requestID,
	}
}

type TaskResponse oapigen.TaskResponse

func taskResponseFromTaskConfig(tc config.TaskConfig, requestID oapigen.RequestID) TaskResponse {
	task := oapigenTaskFromConfigTask(tc)

	tr := TaskResponse{
		RequestId: requestID,
		Task:      &task,
	}
	return tr
}

func (tresp TaskResponse) String() string {
	data, _ := json.Marshal(tresp)
	return string(data)
}

func oapigenTaskFromConfigTask(tc config.TaskConfig) oapigen.Task {
	task := oapigen.Task{
		Description: tc.Description,
		Version:     tc.Version,
		Enabled:     tc.Enabled,
	}

	if tc.Name != nil {
		task.Name = *tc.Name
	}

	if tc.Module != nil {
		task.Module = *tc.Module
	}

	if tc.Variables != nil {
		task.Variables = &oapigen.VariableMap{
			AdditionalProperties: tc.Variables,
		}
	}

	if tc.Providers != nil {
		task.Providers = &tc.Providers
	}

	if tc.ModuleInputs != nil {
		task.ModuleInput = new(oapigen.ModuleInput)
		for _, moduleInput := range *tc.ModuleInputs {
			switch input := moduleInput.(type) {
			case *config.ServicesModuleInputConfig:
				if len(input.Names) > 0 {
					task.ModuleInput.Services = &oapigen.ServicesModuleInput{
						Names:      &input.Names,
						Datacenter: input.Datacenter,
						Namespace:  input.Namespace,
						Filter:     input.Filter,
						CtsUserDefinedMeta: &oapigen.ServicesModuleInput_CtsUserDefinedMeta{
							AdditionalProperties: input.CTSUserDefinedMeta,
						},
					}
				} else {
					task.ModuleInput.Services = &oapigen.ServicesModuleInput{
						Regexp:     input.Regexp,
						Datacenter: input.Datacenter,
						Namespace:  input.Namespace,
						Filter:     input.Filter,
						CtsUserDefinedMeta: &oapigen.ServicesModuleInput_CtsUserDefinedMeta{
							AdditionalProperties: input.CTSUserDefinedMeta,
						},
					}
				}
			case *config.ConsulKVModuleInputConfig:
				task.ModuleInput.ConsulKv = &oapigen.ConsulKVModuleInput{
					Datacenter: input.Datacenter,
					Recurse:    input.Recurse,
					Path:       *input.Path,
					Namespace:  input.Namespace,
				}
			}
		}
	}

	switch cond := tc.Condition.(type) {
	case *config.ServicesConditionConfig:
		services := &oapigen.ServicesCondition{
			Datacenter: cond.Datacenter,
			Namespace:  cond.Namespace,
			Filter:     cond.Filter,
			CtsUserDefinedMeta: &oapigen.ServicesCondition_CtsUserDefinedMeta{
				AdditionalProperties: cond.CTSUserDefinedMeta,
			},
			UseAsModuleInput: cond.UseAsModuleInput,
		}
		if len(cond.Names) > 0 {
			services.Names = &cond.Names
		} else {
			services.Regexp = cond.Regexp
		}
		task.Condition.Services = services
	case *config.CatalogServicesConditionConfig:
		task.Condition.CatalogServices = &oapigen.CatalogServicesCondition{
			Regexp:           *cond.Regexp,
			UseAsModuleInput: cond.UseAsModuleInput,
			Datacenter:       cond.Datacenter,
			Namespace:        cond.Namespace,
			NodeMeta: &oapigen.CatalogServicesCondition_NodeMeta{
				AdditionalProperties: cond.NodeMeta,
			},
		}
	case *config.ConsulKVConditionConfig:
		task.Condition.ConsulKv = &oapigen.ConsulKVCondition{
			Datacenter:       cond.Datacenter,
			Recurse:          cond.Recurse,
			Path:             *cond.Path,
			Namespace:        cond.Namespace,
			UseAsModuleInput: cond.UseAsModuleInput,
		}
	case *config.ScheduleConditionConfig:
		task.Condition.Schedule = &oapigen.ScheduleCondition{
			Cron: *cond.Cron,
		}
	}

	if tc.BufferPeriod != nil {
		max := config.TimeDurationVal(tc.BufferPeriod.Max).String()
		min := config.TimeDurationVal(tc.BufferPeriod.Min).String()
		task.BufferPeriod = &oapigen.BufferPeriod{
			Enabled: tc.BufferPeriod.Enabled,
			Max:     &max,
			Min:     &min,
		}
	}

	// Tasks created via API cannot configure the `services` field, but tasks
	// created via CTS config file can currently configure `services` (deprecated).
	// Handle `services` by converting to condition or module_input. There is
	// config validation so that `services` cannot be configured when
	// `condition "services"` or `module_input "services"` is configured.
	// Use-case: returning tasks with `services via Get Task API
	if tc.DeprecatedServices != nil && len(tc.DeprecatedServices) > 0 {
		_, noCondition := tc.Condition.(*config.NoConditionConfig)
		if tc.Condition == nil || noCondition {
			task.Condition.Services = &oapigen.ServicesCondition{
				Names:            &tc.DeprecatedServices,
				UseAsModuleInput: config.Bool(true),
			}
		} else {
			if tc.ModuleInputs == nil {
				task.ModuleInput = new(oapigen.ModuleInput)
			}
			task.ModuleInput.Services = &oapigen.ServicesModuleInput{
				Names: &tc.DeprecatedServices,
			}
		}
	}

	// Enterprise
	if tc.DeprecatedTFVersion != nil && *tc.DeprecatedTFVersion != "" {
		task.TerraformVersion = tc.DeprecatedTFVersion
	}

	if tc.TFCWorkspace != nil && !tc.TFCWorkspace.IsEmpty() {
		task.TerraformCloudWorkspace = &oapigen.TerraformCloudWorkspace{
			ExecutionMode:    tc.TFCWorkspace.ExecutionMode,
			AgentPoolId:      tc.TFCWorkspace.AgentPoolID,
			AgentPoolName:    tc.TFCWorkspace.AgentPoolName,
			TerraformVersion: tc.TFCWorkspace.TerraformVersion,
		}
	}

	return task
}
