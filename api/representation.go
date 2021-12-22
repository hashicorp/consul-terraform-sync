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

// ToTaskRequestConfig converts a taskRequest object to a Config TaskConfig object. It takes as arguments a buffer period,
// and a working directory which are required to finalize the task config.
func (tr taskRequest) ToTaskRequestConfig() (config.TaskConfig, error) {
	tc := config.TaskConfig{
		Description: tr.Description,
		Name:        &tr.Name,
		Module:      &tr.Module,
		Version:     tr.Version,
		Enabled:     tr.Enabled,
		WorkingDir:  tr.WorkingDir,
	}

	if tr.Providers != nil {
		tc.Providers = *tr.Providers
	}

	if tr.Services != nil {
		tc.Services = *tr.Services
	}

	// Convert source input
	if tr.SourceInput != nil {
		if tr.SourceInput.Services != nil {
			si := &config.ServicesSourceInputConfig{
				ServicesMonitorConfig: config.ServicesMonitorConfig{
					Regexp: tr.SourceInput.Services.Regexp,
				},
			}
			if tr.SourceInput.Services.Names != nil {
				si.Names = *tr.SourceInput.Services.Names
			}
			tc.SourceInput = si
		} else if tr.SourceInput.ConsulKv != nil {
			tc.SourceInput = &config.ConsulKVSourceInputConfig{
				ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
					Datacenter: tr.SourceInput.ConsulKv.Datacenter,
					Recurse:    tr.SourceInput.ConsulKv.Recurse,
					Path:       &tr.SourceInput.ConsulKv.Path,
					Namespace:  tr.SourceInput.ConsulKv.Namespace,
				},
			}
		}
	}

	// Convert condition
	if tr.Condition != nil {
		if tr.Condition.Services != nil {
			cond := &config.ServicesConditionConfig{
				ServicesMonitorConfig: config.ServicesMonitorConfig{
					Regexp: tr.Condition.Services.Regexp,
				},
			}
			if tr.Condition.Services.Names != nil {
				cond.Names = *tr.Condition.Services.Names
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
				SourceIncludesVar: tr.Condition.ConsulKv.SourceIncludesVar,
			}
		} else if tr.Condition.CatalogServices != nil {
			tc.Condition = &config.CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig: config.CatalogServicesMonitorConfig{
					Regexp:            config.String(tr.Condition.CatalogServices.Regexp),
					SourceIncludesVar: tr.Condition.CatalogServices.SourceIncludesVar,
					Datacenter:        tr.Condition.CatalogServices.Datacenter,
					Namespace:         tr.Condition.CatalogServices.Namespace,
					NodeMeta:          tr.Condition.CatalogServices.NodeMeta.AdditionalProperties,
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
		Source:      *tc.Source,
		Version:     tc.Version,
		Enabled:     tc.Enabled,
		WorkingDir:  tc.WorkingDir,
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

	if tc.SourceInput != nil {
		task.SourceInput = new(oapigen.SourceInput)
		switch si := tc.SourceInput.(type) {
		case *config.ServicesSourceInputConfig:
			task.SourceInput.Services = &oapigen.ServicesSourceInput{
				Regexp: si.Regexp,
			}
		case *config.ConsulKVSourceInputConfig:
			task.SourceInput.ConsulKv = &oapigen.ConsulKVSourceInput{
				Datacenter: si.Datacenter,
				Recurse:    si.Recurse,
				Path:       *si.Path,
				Namespace:  si.Namespace,
			}
		}
	}

	if tc.Condition != nil {
		task.Condition = new(oapigen.Condition)
		switch cond := tc.Condition.(type) {
		case *config.ServicesConditionConfig:
			task.Condition.Services = &oapigen.ServicesCondition{
				Regexp: cond.Regexp,
			}
		case *config.CatalogServicesConditionConfig:
			task.Condition.CatalogServices = &oapigen.CatalogServicesCondition{
				Regexp:            cond.Regexp,
				SourceIncludesVar: cond.SourceIncludesVar,
				Datacenter:        cond.Datacenter,
				Namespace:         cond.Namespace,
				NodeMeta: &oapigen.CatalogServicesCondition_NodeMeta{
					AdditionalProperties: cond.NodeMeta,
				},
			}
		case *config.ConsulKVConditionConfig:
			task.Condition.ConsulKv = &oapigen.ConsulKVCondition{
				Datacenter:        cond.Datacenter,
				Recurse:           cond.Recurse,
				Path:              *cond.Path,
				Namespace:         cond.Namespace,
				SourceIncludesVar: cond.SourceIncludesVar,
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
