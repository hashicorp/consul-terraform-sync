package api

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// TaskRequest is a wrapper around the generated TaskRequest
// this allows for the task request to be extended
type TaskRequest oapigen.TaskRequest

// TaskRequestFromTaskConfig converts a taskRequest object to a Config TaskConfig object.
func TaskRequestFromTaskConfig(tc config.TaskConfig) (TaskRequest, error) {
	if len(tc.VarFiles) > 0 {
		tc.Variables = make(map[string]string)
		for _, vf := range tc.VarFiles {
			f, err := os.Open(vf)
			if err != nil {
				return TaskRequest{}, err
			}

			err = readToVariablesMap(vf, f, tc.Variables)
			if err != nil {
				return TaskRequest{}, err
			}
		}
	}

	tr := oapigenTaskFromConfigTask(tc)
	return TaskRequest(tr), nil
}

func readToVariablesMap(filename string, reader io.Reader, variables map[string]string) error {
	// Load all variables from passed in variable files before
	// converting to map[string]string
	loadedVars := make(hcltmpl.Variables)
	tfvars, err := tftmpl.LoadModuleVariables(filename, reader)
	if err != nil {
		return err
	}

	for k, v := range tfvars {
		loadedVars[k] = v
	}

	// Value can be anything so marshal it to equivalent json
	// and store json as the string value in the map
	for k, v := range loadedVars {
		b, err := ctyjson.Marshal(v, v.Type())
		if err != nil {
			return err
		}
		variables[k] = string(b)
	}

	return nil
}

// ToTaskConfig converts a TaskRequest object to a Config TaskConfig object.
func (tr TaskRequest) ToTaskConfig() (config.TaskConfig, error) {
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
		inputs := make(config.ModuleInputConfigs, 0)
		if tr.ModuleInput.Services != nil {
			input := &config.ServicesModuleInputConfig{
				ServicesMonitorConfig: config.ServicesMonitorConfig{
					Regexp: tr.ModuleInput.Services.Regexp,
				},
			}
			if tr.ModuleInput.Services.Names != nil {
				input.Names = *tr.ModuleInput.Services.Names
			}
			inputs = append(inputs, input)
		}
		if tr.ModuleInput.ConsulKv != nil {
			input := &config.ConsulKVModuleInputConfig{
				ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
					Datacenter: tr.ModuleInput.ConsulKv.Datacenter,
					Recurse:    tr.ModuleInput.ConsulKv.Recurse,
					Path:       &tr.ModuleInput.ConsulKv.Path,
					Namespace:  tr.ModuleInput.ConsulKv.Namespace,
				},
			}
			inputs = append(inputs, input)
		}
		tc.ModuleInputs = &inputs
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
func (tr TaskRequest) String() string {
	data, _ := json.Marshal(tr)
	return string(data)
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

	if tc.Services != nil {
		task.Services = &tc.Services
	}

	if tc.ModuleInputs != nil {
		task.ModuleInput = new(oapigen.ModuleInput)
		for _, moduleInput := range *tc.ModuleInputs {
			switch input := moduleInput.(type) {
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

	return task
}
