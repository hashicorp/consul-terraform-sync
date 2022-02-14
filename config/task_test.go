package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTaskConfig_Copy(t *testing.T) {
	cases := []struct {
		name string
		a    *TaskConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&TaskConfig{},
		},
		{
			"same_enabled",
			&TaskConfig{
				Description:        String("description"),
				Name:               String("name"),
				Providers:          []string{"provider"},
				DeprecatedServices: []string{"service"},
				Module:             String("path"),
				Version:            String("0.0.0"),
				Enabled:            Bool(true),
				Condition: &CatalogServicesConditionConfig{
					CatalogServicesMonitorConfig{
						Regexp:           String(".*"),
						UseAsModuleInput: Bool(true),
						Datacenter:       String("dc2"),
						Namespace:        String("ns2"),
						NodeMeta: map[string]string{
							"key": "value",
						},
					},
				},
				DeprecatedSourceInputs: &ModuleInputConfigs{
					&ServicesModuleInputConfig{
						ServicesMonitorConfig: ServicesMonitorConfig{
							Regexp: String(".*"),
						},
					},
				},
				ModuleInputs: &ModuleInputConfigs{
					&ConsulKVModuleInputConfig{
						ConsulKVMonitorConfig: ConsulKVMonitorConfig{
							Path: String("path"),
						},
					},
				},
				WorkingDir: String("cts-dir"),
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Copy()
			assert.Equal(t, tc.a, r)
		})
	}
}

func TestTaskConfig_Merge(t *testing.T) {
	cases := []struct {
		name string
		a    *TaskConfig
		b    *TaskConfig
		r    *TaskConfig
	}{
		{
			"nil_a",
			nil,
			&TaskConfig{},
			&TaskConfig{},
		},
		{
			"nil_b",
			&TaskConfig{},
			nil,
			&TaskConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&TaskConfig{},
			&TaskConfig{},
			&TaskConfig{},
		},
		{
			"description_overrides",
			&TaskConfig{Description: String("description")},
			&TaskConfig{Description: String("describe")},
			&TaskConfig{Description: String("describe")},
		},
		{
			"description_empty_one",
			&TaskConfig{Description: String("description")},
			&TaskConfig{},
			&TaskConfig{Description: String("description")},
		},
		{
			"description_empty_two",
			&TaskConfig{},
			&TaskConfig{Description: String("description")},
			&TaskConfig{Description: String("description")},
		},
		{
			"description_same",
			&TaskConfig{Description: String("description")},
			&TaskConfig{Description: String("description")},
			&TaskConfig{Description: String("description")},
		},
		{
			"name_overrides",
			&TaskConfig{Name: String("name")},
			&TaskConfig{Name: String("service")},
			&TaskConfig{Name: String("service")},
		},
		{
			"name_empty_one",
			&TaskConfig{Name: String("name")},
			&TaskConfig{},
			&TaskConfig{Name: String("name")},
		},
		{
			"name_empty_two",
			&TaskConfig{},
			&TaskConfig{Name: String("name")},
			&TaskConfig{Name: String("name")},
		},
		{
			"name_same",
			&TaskConfig{Name: String("name")},
			&TaskConfig{Name: String("name")},
			&TaskConfig{Name: String("name")},
		},
		{
			"services_merges",
			&TaskConfig{DeprecatedServices: []string{"a"}},
			&TaskConfig{DeprecatedServices: []string{"b"}},
			&TaskConfig{DeprecatedServices: []string{"a", "b"}},
		},
		{
			"services_empty_one",
			&TaskConfig{DeprecatedServices: []string{"service"}},
			&TaskConfig{},
			&TaskConfig{DeprecatedServices: []string{"service"}},
		},
		{
			"services_empty_two",
			&TaskConfig{},
			&TaskConfig{DeprecatedServices: []string{"service"}},
			&TaskConfig{DeprecatedServices: []string{"service"}},
		},
		{
			"providers_merges",
			&TaskConfig{Providers: []string{"a"}},
			&TaskConfig{Providers: []string{"b"}},
			&TaskConfig{Providers: []string{"a", "b"}},
		},
		{
			"providers_empty_one",
			&TaskConfig{Providers: []string{"provider"}},
			&TaskConfig{},
			&TaskConfig{Providers: []string{"provider"}},
		},
		{
			"providers_empty_two",
			&TaskConfig{},
			&TaskConfig{Providers: []string{"provider"}},
			&TaskConfig{Providers: []string{"provider"}},
		},
		{
			"source_overrides",
			&TaskConfig{DeprecatedSource: String("path")},
			&TaskConfig{DeprecatedSource: String("")},
			&TaskConfig{DeprecatedSource: String("")},
		},
		{
			"source_empty_one",
			&TaskConfig{DeprecatedSource: String("path")},
			&TaskConfig{},
			&TaskConfig{DeprecatedSource: String("path")},
		},
		{
			"source_empty_two",
			&TaskConfig{},
			&TaskConfig{DeprecatedSource: String("path")},
			&TaskConfig{DeprecatedSource: String("path")},
		},
		{
			"source_same",
			&TaskConfig{DeprecatedSource: String("path")},
			&TaskConfig{DeprecatedSource: String("path")},
			&TaskConfig{DeprecatedSource: String("path")},
		},
		{
			"module_overrides",
			&TaskConfig{Module: String("module")},
			&TaskConfig{Module: String("")},
			&TaskConfig{Module: String("")},
		},
		{
			"module_empty_one",
			&TaskConfig{Module: String("module")},
			&TaskConfig{},
			&TaskConfig{Module: String("module")},
		},
		{
			"module_empty_two",
			&TaskConfig{},
			&TaskConfig{Module: String("module")},
			&TaskConfig{Module: String("module")},
		},
		{
			"module_same",
			&TaskConfig{Module: String("module")},
			&TaskConfig{Module: String("module")},
			&TaskConfig{Module: String("module")},
		},
		{
			"version_overrides",
			&TaskConfig{Version: String("0.0.0")},
			&TaskConfig{Version: String("")},
			&TaskConfig{Version: String("")},
		},
		{
			"version_empty_one",
			&TaskConfig{Version: String("0.0.0")},
			&TaskConfig{},
			&TaskConfig{Version: String("0.0.0")},
		},
		{
			"version_empty_two",
			&TaskConfig{},
			&TaskConfig{Version: String("0.0.0")},
			&TaskConfig{Version: String("0.0.0")},
		},
		{
			"version_same",
			&TaskConfig{Version: String("0.0.0")},
			&TaskConfig{Version: String("0.0.0")},
			&TaskConfig{Version: String("0.0.0")},
		},
		{
			"tf_version_merges",
			&TaskConfig{TFVersion: String("0.14.0")},
			&TaskConfig{TFVersion: String("0.15.5")},
			&TaskConfig{TFVersion: String("0.15.5")},
		},
		{
			"tf_version_empty_one",
			&TaskConfig{TFVersion: String("0.15.0")},
			&TaskConfig{},
			&TaskConfig{TFVersion: String("0.15.0")},
		},
		{
			"tf_version_empty_two",
			&TaskConfig{},
			&TaskConfig{TFVersion: String("0.15.0")},
			&TaskConfig{TFVersion: String("0.15.0")},
		},
		{
			"tf_version_same",
			&TaskConfig{TFVersion: String("0.15.0")},
			&TaskConfig{TFVersion: String("0.15.0")},
			&TaskConfig{TFVersion: String("0.15.0")},
		},
		{
			"enabled_overrides",
			&TaskConfig{Enabled: Bool(false)},
			&TaskConfig{Enabled: Bool(true)},
			&TaskConfig{Enabled: Bool(true)},
		},
		{
			"enabled_empty_one",
			&TaskConfig{Enabled: Bool(false)},
			&TaskConfig{},
			&TaskConfig{Enabled: Bool(false)},
		},
		{
			"enabled_empty_two",
			&TaskConfig{},
			&TaskConfig{Enabled: Bool(false)},
			&TaskConfig{Enabled: Bool(false)},
		},
		{
			"enabled_same",
			&TaskConfig{Enabled: Bool(false)},
			&TaskConfig{Enabled: Bool(false)},
			&TaskConfig{Enabled: Bool(false)},
		},
		{
			"condition_overrides",
			&TaskConfig{Condition: &CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{Condition: &CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String("")}}},
			&TaskConfig{Condition: &CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String("")}}},
		},
		{
			"condition_empty_one",
			&TaskConfig{Condition: &CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{},
			&TaskConfig{Condition: &CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String(".*")}}},
		},
		{
			"condition_empty_two",
			&TaskConfig{},
			&TaskConfig{Condition: &CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{Condition: &CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String(".*")}}},
		},
		{
			"condition_same",
			&TaskConfig{Condition: &CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{Condition: &CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{Condition: &CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String(".*")}}},
		},
		{
			"working_dir_overrides",
			&TaskConfig{WorkingDir: String("cts-dir")},
			&TaskConfig{WorkingDir: String("cts-dir-override")},
			&TaskConfig{WorkingDir: String("cts-dir-override")},
		},
		{
			"working_dir_empty_one",
			&TaskConfig{WorkingDir: String("cts-dir")},
			&TaskConfig{},
			&TaskConfig{WorkingDir: String("cts-dir")},
		},
		{
			"working_dir_empty_two",
			&TaskConfig{},
			&TaskConfig{WorkingDir: String("cts-dir")},
			&TaskConfig{WorkingDir: String("cts-dir")},
		},
		{
			"working_dir_same",
			&TaskConfig{WorkingDir: String("cts-dir")},
			&TaskConfig{WorkingDir: String("cts-dir")},
			&TaskConfig{WorkingDir: String("cts-dir")},
		},
		{
			"source_input_merges",
			&TaskConfig{DeprecatedSourceInputs: &ModuleInputConfigs{&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String("a")}}}},
			&TaskConfig{DeprecatedSourceInputs: &ModuleInputConfigs{&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String("b")}}}},
			&TaskConfig{DeprecatedSourceInputs: &ModuleInputConfigs{
				&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String("a")}},
				&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String("b")}},
			}},
		},
		{
			"source_input_empty_one",
			&TaskConfig{DeprecatedSourceInputs: &ModuleInputConfigs{&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}}},
			&TaskConfig{},
			&TaskConfig{DeprecatedSourceInputs: &ModuleInputConfigs{&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}}},
		},
		{
			"source_input_empty_two",
			&TaskConfig{},
			&TaskConfig{DeprecatedSourceInputs: &ModuleInputConfigs{&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}}},
			&TaskConfig{DeprecatedSourceInputs: &ModuleInputConfigs{&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}}},
		},
		{
			"module_input_merges",
			&TaskConfig{ModuleInputs: &ModuleInputConfigs{&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String("a")}}}},
			&TaskConfig{ModuleInputs: &ModuleInputConfigs{&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String("b")}}}},
			&TaskConfig{ModuleInputs: &ModuleInputConfigs{
				&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String("a")}},
				&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String("b")}},
			}},
		},
		{
			"module_input_empty_one",
			&TaskConfig{ModuleInputs: &ModuleInputConfigs{&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}}},
			&TaskConfig{},
			&TaskConfig{ModuleInputs: &ModuleInputConfigs{&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}}},
		},
		{
			"module_input_empty_two",
			&TaskConfig{},
			&TaskConfig{ModuleInputs: &ModuleInputConfigs{&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}}},
			&TaskConfig{ModuleInputs: &ModuleInputConfigs{&ServicesModuleInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}}},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestTaskConfig_Finalize(t *testing.T) {
	cases := []struct {
		name string
		i    *TaskConfig
		r    *TaskConfig
	}{
		{
			"empty",
			&TaskConfig{},
			&TaskConfig{
				Description:        String(""),
				Name:               String(""),
				Providers:          []string{},
				DeprecatedServices: []string{},
				Module:             String(""),
				VarFiles:           []string{},
				Variables:          map[string]string{},
				Version:            String(""),
				TFVersion:          String(""),
				BufferPeriod:       DefaultBufferPeriodConfig(),
				Enabled:            Bool(true),
				Condition:          EmptyConditionConfig(),
				WorkingDir:         String("sync-tasks"),
				ModuleInputs:       DefaultModuleInputConfigs(),
			},
		},
		{
			"with_name",
			&TaskConfig{
				Name: String("task"),
			},
			&TaskConfig{
				Description:        String(""),
				Name:               String("task"),
				Providers:          []string{},
				DeprecatedServices: []string{},
				Module:             String(""),
				VarFiles:           []string{},
				Variables:          map[string]string{},
				Version:            String(""),
				TFVersion:          String(""),
				BufferPeriod:       DefaultBufferPeriodConfig(),
				Enabled:            Bool(true),
				Condition:          EmptyConditionConfig(),
				WorkingDir:         String("sync-tasks/task"),
				ModuleInputs:       DefaultModuleInputConfigs(),
			},
		},
		{
			"with_schedule_condition",
			&TaskConfig{
				Name:      String("task"),
				Condition: &ScheduleConditionConfig{},
			},
			&TaskConfig{
				Description:        String(""),
				Name:               String("task"),
				Providers:          []string{},
				DeprecatedServices: []string{},
				Module:             String(""),
				VarFiles:           []string{},
				Variables:          map[string]string{},
				Version:            String(""),
				TFVersion:          String(""),
				BufferPeriod: &BufferPeriodConfig{
					Enabled: Bool(false),
					Min:     TimeDuration(0 * time.Second),
					Max:     TimeDuration(0 * time.Second),
				},
				Enabled:      Bool(true),
				Condition:    &ScheduleConditionConfig{String("")},
				WorkingDir:   String("sync-tasks/task"),
				ModuleInputs: DefaultModuleInputConfigs(),
			},
		},
		{
			"with_services_module_input",
			&TaskConfig{
				Name:      String("task"),
				Condition: &ScheduleConditionConfig{},
				ModuleInputs: &ModuleInputConfigs{
					&ServicesModuleInputConfig{
						ServicesMonitorConfig{Regexp: String("^api$")}},
				},
			},
			&TaskConfig{
				Description:        String(""),
				Name:               String("task"),
				Providers:          []string{},
				DeprecatedServices: []string{},
				Module:             String(""),
				VarFiles:           []string{},
				Variables:          map[string]string{},
				Version:            String(""),
				TFVersion:          String(""),
				BufferPeriod: &BufferPeriodConfig{
					Enabled: Bool(false),
					Min:     TimeDuration(0 * time.Second),
					Max:     TimeDuration(0 * time.Second),
				},
				Enabled:    Bool(true),
				Condition:  &ScheduleConditionConfig{String("")},
				WorkingDir: String("sync-tasks/task"),
				ModuleInputs: &ModuleInputConfigs{&ServicesModuleInputConfig{
					ServicesMonitorConfig{
						Regexp:             String("^api$"),
						Names:              []string{},
						Datacenter:         String(""),
						Namespace:          String(""),
						Filter:             String(""),
						CTSUserDefinedMeta: map[string]string{},
					}}},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			tc.i.Finalize(DefaultBufferPeriodConfig(), DefaultWorkingDir)
			assert.Equal(t, tc.r, tc.i)
		})
	}
}

func TestTaskConfig_Finalize_DeprecatedSource(t *testing.T) {
	cases := []struct {
		name     string
		i        *TaskConfig
		expected string
	}{
		{
			"module_configured",
			&TaskConfig{
				Module: String("module/path"),
			},
			"module/path",
		},
		{
			"source_configured",
			&TaskConfig{
				DeprecatedSource: String("source/path"),
			},
			"source/path",
		},
		{
			"module_and_source_configured",
			&TaskConfig{
				Module:           String("module/path"),
				DeprecatedSource: String("source/path"),
			},
			"module/path",
		},
		{
			"none_configured",
			&TaskConfig{},
			"",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			tc.i.Finalize(DefaultBufferPeriodConfig(), DefaultWorkingDir)
			assert.Equal(t, tc.expected, *tc.i.Module)
		})
	}
}

func TestTaskConfig_Finalize_DeprecatedSourceInputs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		i        *TaskConfig
		expected *ModuleInputConfigs
	}{
		{
			"module_input_configured",
			&TaskConfig{
				ModuleInputs: &ModuleInputConfigs{
					&ConsulKVModuleInputConfig{
						ConsulKVMonitorConfig: ConsulKVMonitorConfig{
							Path: String("path"),
						},
					},
				},
			},
			&ModuleInputConfigs{
				&ConsulKVModuleInputConfig{
					ConsulKVMonitorConfig: ConsulKVMonitorConfig{
						Path:       String("path"),
						Recurse:    Bool(false),
						Datacenter: String(""),
						Namespace:  String(""),
					},
				},
			},
		},
		{
			"source_input_configured",
			&TaskConfig{
				DeprecatedSourceInputs: &ModuleInputConfigs{
					&ServicesModuleInputConfig{
						ServicesMonitorConfig: ServicesMonitorConfig{
							Regexp: String(".*"),
						},
					},
				},
			},
			&ModuleInputConfigs{
				&ServicesModuleInputConfig{
					ServicesMonitorConfig: ServicesMonitorConfig{
						Regexp:             String(".*"),
						Names:              []string{},
						Datacenter:         String(""),
						Namespace:          String(""),
						Filter:             String(""),
						CTSUserDefinedMeta: map[string]string{},
					},
				},
			},
		},
		{
			"both_configured",
			&TaskConfig{
				ModuleInputs: &ModuleInputConfigs{
					&ConsulKVModuleInputConfig{
						ConsulKVMonitorConfig: ConsulKVMonitorConfig{
							Path: String("path"),
						},
					},
				},
				DeprecatedSourceInputs: &ModuleInputConfigs{
					&ServicesModuleInputConfig{
						ServicesMonitorConfig: ServicesMonitorConfig{
							Regexp: String(".*"),
						},
					},
				},
			},
			&ModuleInputConfigs{
				&ConsulKVModuleInputConfig{
					ConsulKVMonitorConfig: ConsulKVMonitorConfig{
						Path:       String("path"),
						Recurse:    Bool(false),
						Datacenter: String(""),
						Namespace:  String(""),
					},
				},
				&ServicesModuleInputConfig{
					ServicesMonitorConfig: ServicesMonitorConfig{
						Regexp:             String(".*"),
						Names:              []string{},
						Datacenter:         String(""),
						Namespace:          String(""),
						Filter:             String(""),
						CTSUserDefinedMeta: map[string]string{},
					},
				},
			},
		},
		{
			"none_configured",
			&TaskConfig{},
			DefaultModuleInputConfigs(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.i.Finalize(DefaultBufferPeriodConfig(), DefaultWorkingDir)
			assert.Equal(t, tc.expected, tc.i.ModuleInputs)
		})
	}
}

func TestTaskConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		i       *TaskConfig
		isValid bool
	}{
		{
			"invalid: nil",
			nil,
			false,
		},
		{
			"invalid: empty",
			&TaskConfig{},
			false,
		},
		{
			"invalid: task name: missing",
			&TaskConfig{
				Condition: &ServicesConditionConfig{
					ServicesMonitorConfig: ServicesMonitorConfig{
						Names: []string{"api"},
					},
				},
				Module: String("path"),
			},
			false,
		},
		{
			"invalid: task name: invalid char",
			&TaskConfig{
				Name: String("cannot contain spaces"),
				Condition: &ServicesConditionConfig{
					ServicesMonitorConfig: ServicesMonitorConfig{
						Names: []string{"api"},
					},
				},
				Module: String("path"),
			},
			false,
		},
		{
			"invalid: task module: missing",
			&TaskConfig{
				Name: String("task"),
				Condition: &ServicesConditionConfig{
					ServicesMonitorConfig: ServicesMonitorConfig{
						Names: []string{"api"},
					},
				},
			},
			false,
		},
		{
			"invalid: TF version: unsupported version",
			&TaskConfig{
				Name: String("task"),
				Condition: &ServicesConditionConfig{
					ServicesMonitorConfig: ServicesMonitorConfig{
						Names: []string{"api"},
					},
				},
				Module:    String("path"),
				TFVersion: String("0.15.0"),
			},
			false,
		},
		{
			"invalid: provider: duplicate",
			&TaskConfig{
				Name: String("task"),
				Condition: &ServicesConditionConfig{
					ServicesMonitorConfig: ServicesMonitorConfig{
						Names: []string{"api"},
					},
				},
				Module:    String("path"),
				Providers: []string{"providerA", "providerA"},
			},
			false,
		},
		{
			"invalid: provider: duplicate with alias",
			&TaskConfig{
				Name: String("task"),
				Condition: &ServicesConditionConfig{
					ServicesMonitorConfig: ServicesMonitorConfig{
						Names: []string{"api"},
					},
				},
				Module:    String("path"),
				Providers: []string{"providerA", "providerA.alias"},
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			err := tc.i.Validate()
			if tc.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestTasksConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		i       TaskConfigs
		isValid bool
	}{
		{
			name:    "nil",
			i:       nil,
			isValid: false,
		}, {
			name: "one task",
			i: []*TaskConfig{
				{
					Name: String("task"),
					Condition: &ServicesConditionConfig{
						ServicesMonitorConfig: ServicesMonitorConfig{
							Names: []string{"serviceA", "serviceB"},
						},
					},
					Module:    String("path"),
					Providers: []string{"providerA", "providerB"},
				},
			},
			isValid: true,
		}, {
			name: "two tasks",
			i: []*TaskConfig{
				{
					Name: String("task"),
					Condition: &ServicesConditionConfig{
						ServicesMonitorConfig: ServicesMonitorConfig{
							Names: []string{"serviceA", "serviceB"},
						},
					},
					Module:    String("path"),
					Providers: []string{"providerA", "providerB"},
				},
				{
					Name: String("task2"),
					Condition: &ServicesConditionConfig{
						ServicesMonitorConfig: ServicesMonitorConfig{
							Names: []string{"serviceC"},
						},
					},
					Module:    String("path"),
					Providers: []string{"providerC"},
				},
			},
			isValid: true,
		}, {
			name: "duplicate task names",
			i: []*TaskConfig{
				{
					Name: String("task"),
					Condition: &ServicesConditionConfig{
						ServicesMonitorConfig: ServicesMonitorConfig{
							Names: []string{"api"},
						},
					},
					Module:    String("path"),
					Providers: []string{"providerA", "providerB"},
				}, {
					Name: String("task"),
					Condition: &ServicesConditionConfig{
						ServicesMonitorConfig: ServicesMonitorConfig{
							Names: []string{"api"},
						},
					},
					Module:    String("path"),
					Providers: []string{"providerA"},
				},
			},
			isValid: false,
		}, {
			name: "one invalid",
			i: []*TaskConfig{
				{
					Name: String("task"),
					Condition: &ServicesConditionConfig{
						ServicesMonitorConfig: ServicesMonitorConfig{
							Names: []string{"api"},
						},
					},
					Module:    String("path"),
					Providers: []string{"providerA", "providerB"},
				}, {
					Name: String("invalid"),
				},
			},
			isValid: false,
		}, {
			name: "duplicate provider instances",
			i: []*TaskConfig{
				{
					Name:      String("task"),
					Providers: []string{"provider.A", "provider.B"},
				},
			},
			isValid: false,
		}, {
			name: "unsupported TF version per task",
			i: []*TaskConfig{
				{
					Name: String("task"),
					Condition: &ServicesConditionConfig{
						ServicesMonitorConfig: ServicesMonitorConfig{
							Names: []string{"api"},
						},
					},
					Module:    String("path"),
					TFVersion: String("0.15.0"),
				},
			},
			isValid: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.i.Validate()
			if tc.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestTaskConfig_validateCondition(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		i       *TaskConfig
		isValid bool
	}{
		{
			"valid: only services list",
			&TaskConfig{
				DeprecatedServices: []string{"api"},
			},
			true,
		},
		{
			"valid: only cond-block",
			&TaskConfig{
				Condition: &ServicesConditionConfig{},
			},
			true,
		},
		{
			"valid: services list & non-service cond-block",
			&TaskConfig{
				DeprecatedServices: []string{"api"},
				Condition:          &ConsulKVConditionConfig{},
			},
			true,
		},
		{
			"invalid: no services list & no cond-block configured",
			&TaskConfig{},
			false,
		},
		{
			"invalid: services & services cond-block configured",
			&TaskConfig{
				DeprecatedServices: []string{"api"},
				Condition:          &ServicesConditionConfig{},
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			// add additional required task fields
			tc.i.Name = String("task")
			tc.i.Module = String("path")

			err := tc.i.validateCondition()
			if tc.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
