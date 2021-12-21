package api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/config"
)

// taskRequest is a wrapper around the generated TaskRequest
// this allows for the task request to be extended
type taskRequest oapigen.TaskRequest

type taskRequestConfig struct {
	config.TaskConfig
	variables map[string]string
}

// ToTaskRequestConfig converts a taskRequest object to a Config TaskConfig object. It takes as arguments a buffer period,
// and a working directory which are required to finalize the task config.
func (tr taskRequest) ToTaskRequestConfig(bp *config.BufferPeriodConfig, wd string) (taskRequestConfig, error) {
	tc := config.TaskConfig{
		Description: tr.Description,
		Name:        &tr.Name,
		Module:      &tr.Source,
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
					Regexp:            tr.Condition.CatalogServices.Regexp,
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
				return taskRequestConfig{}, err
			}
		}

		var min time.Duration
		if tr.BufferPeriod.Min != nil {
			min, err = time.ParseDuration(*tr.BufferPeriod.Min)
			if err != nil {
				return taskRequestConfig{}, err
			}
		}

		tc.BufferPeriod = &config.BufferPeriodConfig{
			Enabled: tr.BufferPeriod.Enabled,
			Max:     &max,
			Min:     &min,
		}
	}

	tc.Finalize(bp, wd)
	err := tc.Validate()
	if err != nil {
		return taskRequestConfig{}, err
	}

	var variables map[string]string
	if tr.Variables != nil {
		variables = tr.Variables.AdditionalProperties
	}

	trc := taskRequestConfig{
		TaskConfig: tc,
		variables:  variables,
	}
	return trc, nil
}

// String writes out the task request in an easily readable way
// useful for logging
func (tr taskRequest) String() string {
	data, _ := json.Marshal(tr)
	return fmt.Sprintf("%s", data)
}

type taskResponse oapigen.TaskResponse

func taskResponseFromTaskRequestConfig(trc taskRequestConfig, requestID oapigen.RequestID) taskResponse {
	task := oapigen.Task{
		Description: trc.Description,
		Name:        *trc.Name,
		Source:      *trc.Module,
		Version:     trc.Version,
		Enabled:     trc.Enabled,
		WorkingDir:  trc.WorkingDir,
	}

	if trc.variables != nil {
		task.Variables = &oapigen.VariableMap{
			AdditionalProperties: trc.variables,
		}
	}

	if trc.Providers != nil {
		task.Providers = &trc.Providers
	}

	if trc.Services != nil {
		task.Services = &trc.Services
	}

	task.SourceInput = new(oapigen.SourceInput)
	switch si := trc.SourceInput.(type) {
	case *config.ServicesSourceInputConfig:
		if len(si.Names) > 0 {
			task.SourceInput.Services = &oapigen.ServicesSourceInput{
				Names: &si.Names,
			}
		} else {
			task.SourceInput.Services = &oapigen.ServicesSourceInput{
				Regexp: si.Regexp,
			}
		}
	case *config.ConsulKVSourceInputConfig:
		task.SourceInput.ConsulKv = &oapigen.ConsulKVSourceInput{
			Datacenter: si.Datacenter,
			Recurse:    si.Recurse,
			Path:       *si.Path,
			Namespace:  si.Namespace,
		}
	}

	task.Condition = new(oapigen.Condition)
	switch cond := trc.Condition.(type) {
	case *config.ServicesConditionConfig:
		if len(cond.Names) > 0 {
			task.Condition.Services = &oapigen.ServicesCondition{
				Names: &cond.Names,
			}
		} else {
			task.Condition.Services = &oapigen.ServicesCondition{
				Regexp: cond.Regexp,
			}
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

	max := config.TimeDurationVal(trc.BufferPeriod.Max).String()
	min := config.TimeDurationVal(trc.BufferPeriod.Min).String()
	task.BufferPeriod = &oapigen.BufferPeriod{
		Enabled: trc.BufferPeriod.Enabled,
		Max:     &max,
		Min:     &min,
	}

	tr := taskResponse{
		RequestId: requestID,
		Task:      &task,
	}
	return tr
}

func (tresp taskResponse) String() string {
	data, _ := json.Marshal(tresp)
	return fmt.Sprintf("%s", data)
}
