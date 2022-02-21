package command

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/mitchellh/cli"
	"github.com/stretchr/testify/assert"
)

func TestHandleDeprecations_Errors(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	ui := &cli.BasicUi{
		Writer: &b,
	}

	cases := []struct {
		name           string
		inputTask      config.TaskConfig
		outputContains []string
	}{
		{
			name: "invalid_services_field",
			inputTask: config.TaskConfig{
				DeprecatedServices: []string{"web"},
			},
			outputContains: []string{
				"Error: unable to create request",
				"the 'services' field in the task block is no longer supported",
				`Please replace the 'services' field with the following 'condition "services"' block`,
				`  condition "services" {`,
				`    names=["web"]`,
			},
		},
		{
			name: "invalid_services_field_with_services_con",
			inputTask: config.TaskConfig{
				DeprecatedServices: []string{"web"},
				Condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Regexp: config.String("*"),
					},
				},
			},
			outputContains: []string{
				"Error: unable to create request",
				"the 'services' field in the task block is no longer supported",
				`list of 'services' and 'condition "services"' block cannot both be configured.`,
				`Consider using the 'names' field under 'condition "services`,
			},
		},
		{
			name: "invalid_services_field_with_services_mi",
			inputTask: config.TaskConfig{
				DeprecatedServices: []string{"web"},
				ModuleInputs: &config.ModuleInputConfigs{
					&config.ServicesModuleInputConfig{
						ServicesMonitorConfig: config.ServicesMonitorConfig{
							Regexp: config.String("*"),
						},
					},
				},
			},
			outputContains: []string{
				"Error: unable to create request",
				"the 'services' field in the task block is no longer supported",
				`list of 'services' and 'module_input "services"' block cannot both be configured.`,
				`Consider using the 'names' field under 'module_input "services`,
			},
		},
		{
			name: "invalid_services_field_with_con_and_services_mi",
			inputTask: config.TaskConfig{
				DeprecatedServices: []string{"web"},
				ModuleInputs: &config.ModuleInputConfigs{
					&config.ServicesModuleInputConfig{
						ServicesMonitorConfig: config.ServicesMonitorConfig{
							Regexp: config.String("*"),
						},
					},
				},
				Condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Regexp: config.String("*"),
					},
				},
			},
			outputContains: []string{
				"Error: unable to create request",
				"the 'services' field in the task block is no longer supported",
				`list of 'services' and 'condition "services"' block cannot both be configured.`,
				`Consider using the 'names' field under 'condition "services`,
				"the 'services' field in the task block is no longer supported",
				`list of 'services' and 'module_input "services"' block cannot both be configured.`,
				`Consider using the 'names' field under 'module_input "services`,
			},
		},
		{
			name: "invalid_services_field_with_schedule_con",
			inputTask: config.TaskConfig{
				DeprecatedServices: []string{"web"},
				Condition:          &config.ScheduleConditionConfig{Cron: config.String("*/10 * * * * * *")},
			},
			outputContains: []string{
				"Error: unable to create request",
				"the 'services' field in the task block is no longer supported",
				`Please replace the 'services' field with the following 'module_input' block`,
				`  module_input "services" {`,
				`    names=["web"]`,
			},
		},
		{
			name: "invalid_services_field_with_kv_con",
			inputTask: config.TaskConfig{
				DeprecatedServices: []string{"web"},
				Condition: &config.ConsulKVConditionConfig{
					ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
						Path: config.String("key-path"),
					},
				},
			},
			outputContains: []string{
				"Error: unable to create request",
				"the 'services' field in the task block is no longer supported",
				`Please replace the 'services' field with the following 'module_input' block`,
				`  module_input "services" {`,
				`    names=["web"]`,
			},
		},
		{
			name: "invalid_services_field_with_catalog_con",
			inputTask: config.TaskConfig{
				DeprecatedServices: []string{"web"},
				Condition: &config.CatalogServicesConditionConfig{
					CatalogServicesMonitorConfig: config.CatalogServicesMonitorConfig{
						Regexp: config.String("regex"),
					},
				},
			},
			outputContains: []string{
				"Error: unable to create request",
				"the 'services' field in the task block is no longer supported",
				`Please replace the 'services' field with the following 'module_input' block`,
				`  module_input "services" {`,
				`    names=["web"]`,
			},
		},
		{
			name: "invalid_services_source_input",
			inputTask: config.TaskConfig{
				DeprecatedSourceInputs: &config.ModuleInputConfigs{
					&config.ServicesModuleInputConfig{
						ServicesMonitorConfig: config.ServicesMonitorConfig{
							Regexp: config.String(".*"),
						},
					},
				},
			},
			outputContains: []string{
				"Error: unable to create request",
				"the 'source_input' block in the task is no longer supported",
				"Please replace 'source_input' with 'module_input' in your task configuration",
				"|  -   source_input \"services\" {",
				"|  +   module_input \"services\" {",
			},
		},
		{
			name: "invalid_kv_source_input",
			inputTask: config.TaskConfig{
				DeprecatedSourceInputs: &config.ModuleInputConfigs{
					&config.ConsulKVModuleInputConfig{
						ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
							Path: config.String("key-path"),
						},
					},
				},
			},
			outputContains: []string{
				"Error: unable to create request",
				"the 'source_input' block in the task is no longer supported",
				"Please replace 'source_input' with 'module_input' in your task configuration",
				"|  -   source_input \"consul-kv\" {",
				"|  +   module_input \"consul-kv\" {",
			},
		},
		{
			name: "invalid_services_con_source_includes_var",
			inputTask: config.TaskConfig{
				Condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Regexp: config.String("*"),
					},
					DeprecatedSourceIncludesVar: config.Bool(true),
				},
			},
			outputContains: []string{
				"Error: unable to create request",
				`the 'source_includes_var' field in the task's 'condition "services"' block is no longer supported`,
				"Please replace 'source_includes_var' with 'use_as_module_input' in your condition configuration",
				"|  -     source_includes_var = true",
				"|  +     use_as_module_input = true",
			},
		},
		{
			name: "invalid_kv_con_source_includes_var",
			inputTask: config.TaskConfig{
				Condition: &config.ConsulKVConditionConfig{
					ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
						Path: config.String("key-path"),
					},
					DeprecatedSourceIncludesVar: config.Bool(false),
				},
			},
			outputContains: []string{
				"Error: unable to create request",
				`the 'source_includes_var' field in the task's 'condition "consul-kv"' block is no longer supported`,
				"Please replace 'source_includes_var' with 'use_as_module_input' in your condition configuration",
				"|  -     source_includes_var = false",
				"|  +     use_as_module_input = false",
			},
		},
		{
			name: "invalid_catalog_con_source_includes_var",
			inputTask: config.TaskConfig{
				Condition: &config.CatalogServicesConditionConfig{
					CatalogServicesMonitorConfig: config.CatalogServicesMonitorConfig{
						Regexp:                      config.String("regex"),
						DeprecatedSourceIncludesVar: config.Bool(true),
					},
				},
			},
			outputContains: []string{
				"Error: unable to create request",
				`the 'source_includes_var' field in the task's 'condition "catalog-services"' block is no longer supported`,
				"Please replace 'source_includes_var' with 'use_as_module_input' in your condition configuration",
				"|  -     source_includes_var = true",
				"|  +     use_as_module_input = true",
			},
		},
		{
			name: "invalid_source",
			inputTask: config.TaskConfig{
				DeprecatedSource: config.String("some/path"),
			},
			outputContains: []string{
				"Error: unable to create request",
				"the 'source' field in the task block is no longer supported",
				"Please replace 'source' with 'module' in your task configuration",
				`|  -   source =  "some/path"`,
				`|  +   module =  "some/path"`,
			},
		},
		{
			name: "multiple_invalid_fields",
			inputTask: config.TaskConfig{
				DeprecatedServices: []string{"web"},
				DeprecatedSource:   config.String("some/path"),
				Condition: &config.ConsulKVConditionConfig{
					ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
						Path: config.String("key-path"),
					},
					DeprecatedSourceIncludesVar: config.Bool(true),
				},
				DeprecatedSourceInputs: &config.ModuleInputConfigs{
					&config.ServicesModuleInputConfig{
						ServicesMonitorConfig: config.ServicesMonitorConfig{
							Regexp: config.String(".*"),
						},
					},
				},
			},
			outputContains: []string{
				"Error: unable to create request",
				"the 'source' field in the task block is no longer supported",
				"the 'services' field in the task block is no longer supported",
				`the 'source_includes_var' field in the task's 'condition "consul-kv"' block is no longer supported`,
				"the 'source_input' block in the task is no longer supported",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := handleDeprecations(ui, &tc.inputTask)
			assert.Error(t, err)

			// remove newlines from output
			re := regexp.MustCompile(`\r?\n`)
			output := re.ReplaceAllString(b.String(), " ")

			for _, expect := range tc.outputContains {
				assert.Contains(t, output, expect)
			}
			b.Reset()
		})
	}
}

func TestHandleDeprecations_NoError(t *testing.T) {
	t.Parallel()
	ui := &cli.BasicUi{}

	inputTask := config.TaskConfig{}

	err := handleDeprecations(ui, &inputTask)
	assert.NoError(t, err)
}
