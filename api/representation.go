package api

import (
	"encoding/json"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/config"
)

// taskRequest is a wrapper around the generated TaskRequest
// this allows for the task request to be extended
type taskRequest oapigen.TaskRequest

// ToTaskConfig converts a taskRequest object to a Config TaskConfig object.
func (tr taskRequest) ToTaskConfig() (config.TaskConfig, error) {
	tc := config.TaskConfig{
		Description: tr.Description,
		Name:        &tr.Name,
		Module:      &tr.Module,
		Version:     tr.Version,
		Enabled:     tr.Enabled,
	}

	if tr.Providers != nil {
		tc.Providers = *tr.Providers
	}

	if tr.Services != nil {
		tc.Services = *tr.Services
	}

	// Convert source input
	if tr.ModuleInput != nil {
		if tr.ModuleInput.Services != nil {
			input := &config.ServicesModuleInputConfig{
				ServicesMonitorConfig: config.ServicesMonitorConfig{
					Regexp: tr.ModuleInput.Services.Regexp,
				},
			}
			if tr.ModuleInput.Services.Names != nil {
				input.Names = *tr.ModuleInput.Services.Names
			}
			tc.ModuleInput = input
		} else if tr.ModuleInput.ConsulKv != nil {
			tc.ModuleInput = &config.ConsulKVModuleInputConfig{
				ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
					Datacenter: tr.ModuleInput.ConsulKv.Datacenter,
					Recurse:    tr.ModuleInput.ConsulKv.Recurse,
					Path:       &tr.ModuleInput.ConsulKv.Path,
					Namespace:  tr.ModuleInput.ConsulKv.Namespace,
				},
			}
		}
	}

	// Convert condition
	if tr.Condition != nil {
		if tr.Condition.Services != nil {
			cond := &config.ServicesConditionConfig{
				UseAsModuleInput: tr.Condition.Services.UseAsModuleInput,
			}
			if tr.Condition.Services.Names != nil && len(*tr.Condition.Services.Names) > 0 {
				cond.Names = *tr.Condition.Services.Names
			} else {
				cond.Regexp = tr.Condition.Services.Regexp
			}
			tc.Condition = cond
		} else if tr.Condition.ConsulKv != nil {
			tc.Condition = &config.ConsulKVConditionConfig{
				ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
					Datacenter: tr.Condition.ConsulKv.Datacenter,
					Recurse:    tr.Condition.ConsulKv.Recurse,
					Path:       &tr.Condition.ConsulKv.Path,
					Namespace:  tr.Condition.ConsulKv.Namespace,
				},
				UseAsModuleInput: tr.Condition.ConsulKv.UseAsModuleInput,
			}
		} else if tr.Condition.CatalogServices != nil {
			tc.Condition = &config.CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig: config.CatalogServicesMonitorConfig{
					Regexp:           config.String(tr.Condition.CatalogServices.Regexp),
					UseAsModuleInput: tr.Condition.CatalogServices.UseAsModuleInput,
					Datacenter:       tr.Condition.CatalogServices.Datacenter,
					Namespace:        tr.Condition.CatalogServices.Namespace,
					NodeMeta:         tr.Condition.CatalogServices.NodeMeta.AdditionalProperties,
				},
			}
		} else if tr.Condition.Schedule != nil {
			tc.Condition = &config.ScheduleConditionConfig{
				Cron: &tr.Condition.Schedule.Cron,
			}
		}
	}

	if tr.BufferPeriod != nil {
		var max time.Duration
		var err error
		if tr.BufferPeriod.Max != nil {
			max, err = time.ParseDuration(*tr.BufferPeriod.Max)
			if err != nil {
				return config.TaskConfig{}, err
			}
		}

		var min time.Duration
		if tr.BufferPeriod.Min != nil {
			min, err = time.ParseDuration(*tr.BufferPeriod.Min)
			if err != nil {
				return config.TaskConfig{}, err
			}
		}

		tc.BufferPeriod = &config.BufferPeriodConfig{
			Enabled: tr.BufferPeriod.Enabled,
			Max:     &max,
			Min:     &min,
		}
	}

	if tr.Variables != nil {
		tc.Variables = make(map[string]string)
		for k, v := range tr.Variables.AdditionalProperties {
			tc.Variables[k] = v
		}
	}
	return tc, nil
}

// String writes out the task request in an easily readable way
// useful for logging
func (tr taskRequest) String() string {
	data, _ := json.Marshal(tr)
	return string(data)
}

type taskResponse oapigen.TaskResponse

func taskResponseFromTaskConfig(tc config.TaskConfig, requestID oapigen.RequestID) taskResponse {
	task := oapigen.Task{
		Description: tc.Description,
		Name:        *tc.Name,
		Module:      *tc.Module,
		Version:     tc.Version,
		Enabled:     tc.Enabled,
	}

	if tc.Variables != nil {
		task.Variables = &oapigen.VariableMap{
			AdditionalProperties: tc.Variables,
		}
	}

	if tc.Providers != nil {
		task.Providers = &tc.Providers
	}

	if tc.Services != nil {
		task.Services = &tc.Services
	}

	if tc.ModuleInput != nil {
		task.ModuleInput = new(oapigen.ModuleInput)
		switch input := tc.ModuleInput.(type) {
		case *config.ServicesModuleInputConfig:
			if len(input.Names) > 0 {
				task.ModuleInput.Services = &oapigen.ServicesModuleInput{
					Names: &input.Names,
				}
			} else {
				task.ModuleInput.Services = &oapigen.ServicesModuleInput{
					Regexp: input.Regexp,
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

	if tc.Condition != nil {
		task.Condition = new(oapigen.Condition)
		switch cond := tc.Condition.(type) {
		case *config.ServicesConditionConfig:
			services := &oapigen.ServicesCondition{
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

	tr := taskResponse{
		RequestId: requestID,
		Task:      &task,
	}
	return tr
}

func (tresp taskResponse) String() string {
	data, _ := json.Marshal(tresp)
	return string(data)
}
