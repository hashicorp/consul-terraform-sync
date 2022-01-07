package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				Description: String("description"),
				Name:        String("name"),
				Providers:   []string{"provider"},
				Services:    []string{"service"},
				Module:      String("path"),
				Version:     String("0.0.0"),
				Enabled:     Bool(true),
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
				DeprecatedSourceInput: &ServicesSourceInputConfig{
					ServicesMonitorConfig: ServicesMonitorConfig{
						Regexp: String(".*"),
					},
				},
				ModuleInput: &ConsulKVSourceInputConfig{
					ConsulKVMonitorConfig: ConsulKVMonitorConfig{
						Path: String("path"),
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
			&TaskConfig{Services: []string{"a"}},
			&TaskConfig{Services: []string{"b"}},
			&TaskConfig{Services: []string{"a", "b"}},
		},
		{
			"services_empty_one",
			&TaskConfig{Services: []string{"service"}},
			&TaskConfig{},
			&TaskConfig{Services: []string{"service"}},
		},
		{
			"services_empty_two",
			&TaskConfig{},
			&TaskConfig{Services: []string{"service"}},
			&TaskConfig{Services: []string{"service"}},
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
			"source_input_overrides",
			&TaskConfig{DeprecatedSourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{DeprecatedSourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("")}}},
			&TaskConfig{DeprecatedSourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("")}}},
		},
		{
			"source_input_empty_one",
			&TaskConfig{DeprecatedSourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{},
			&TaskConfig{DeprecatedSourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
		},
		{
			"source_input_empty_two",
			&TaskConfig{},
			&TaskConfig{DeprecatedSourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{DeprecatedSourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
		},
		{
			"source_input_same",
			&TaskConfig{DeprecatedSourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{DeprecatedSourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{DeprecatedSourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
		},
		{
			"module_input_overrides",
			&TaskConfig{ModuleInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{ModuleInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("")}}},
			&TaskConfig{ModuleInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("")}}},
		},
		{
			"module_input_empty_one",
			&TaskConfig{ModuleInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{},
			&TaskConfig{ModuleInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
		},
		{
			"module_input_empty_two",
			&TaskConfig{},
			&TaskConfig{ModuleInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{ModuleInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
		},
		{
			"module_input_same",
			&TaskConfig{ModuleInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{ModuleInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{ModuleInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
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
				Description:  String(""),
				Name:         String(""),
				Providers:    []string{},
				Services:     []string{},
				Module:       String(""),
				VarFiles:     []string{},
				Variables:    map[string]string{},
				Version:      String(""),
				TFVersion:    String(""),
				BufferPeriod: DefaultBufferPeriodConfig(),
				Enabled:      Bool(true),
				Condition:    EmptyConditionConfig(),
				WorkingDir:   String("sync-tasks"),
				ModuleInput:  EmptyModuleInputConfig(),
			},
		},
		{
			"with_name",
			&TaskConfig{
				Name: String("task"),
			},
			&TaskConfig{
				Description:  String(""),
				Name:         String("task"),
				Providers:    []string{},
				Services:     []string{},
				Module:       String(""),
				VarFiles:     []string{},
				Variables:    map[string]string{},
				Version:      String(""),
				TFVersion:    String(""),
				BufferPeriod: DefaultBufferPeriodConfig(),
				Enabled:      Bool(true),
				Condition:    EmptyConditionConfig(),
				WorkingDir:   String("sync-tasks/task"),
				ModuleInput:  EmptyModuleInputConfig(),
			},
		},
		{
			"with_schedule_condition",
			&TaskConfig{
				Name:      String("task"),
				Condition: &ScheduleConditionConfig{},
			},
			&TaskConfig{
				Description: String(""),
				Name:        String("task"),
				Providers:   []string{},
				Services:    []string{},
				Module:      String(""),
				VarFiles:    []string{},
				Variables:   map[string]string{},
				Version:     String(""),
				TFVersion:   String(""),
				BufferPeriod: &BufferPeriodConfig{
					Enabled: Bool(false),
					Min:     TimeDuration(0 * time.Second),
					Max:     TimeDuration(0 * time.Second),
				},
				Enabled:     Bool(true),
				Condition:   &ScheduleConditionConfig{String("")},
				WorkingDir:  String("sync-tasks/task"),
				ModuleInput: EmptyModuleInputConfig(),
			},
		},
		{
			"with_services_module_input",
			&TaskConfig{
				Name:      String("task"),
				Condition: &ScheduleConditionConfig{},
				ModuleInput: &ServicesSourceInputConfig{
					ServicesMonitorConfig{Regexp: String("^api$")}},
			},
			&TaskConfig{
				Description: String(""),
				Name:        String("task"),
				Providers:   []string{},
				Services:    []string{},
				Module:      String(""),
				VarFiles:    []string{},
				Variables:   map[string]string{},
				Version:     String(""),
				TFVersion:   String(""),
				BufferPeriod: &BufferPeriodConfig{
					Enabled: Bool(false),
					Min:     TimeDuration(0 * time.Second),
					Max:     TimeDuration(0 * time.Second),
				},
				Enabled:    Bool(true),
				Condition:  &ScheduleConditionConfig{String("")},
				WorkingDir: String("sync-tasks/task"),
				ModuleInput: &ServicesSourceInputConfig{
					ServicesMonitorConfig{
						Regexp:             String("^api$"),
						Names:              []string{},
						Datacenter:         String(""),
						Namespace:          String(""),
						Filter:             String(""),
						CTSUserDefinedMeta: map[string]string{},
					}},
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

func TestTaskConfig_Finalize_DeprecatedSourceInput(t *testing.T) {
	cases := []struct {
		name     string
		i        *TaskConfig
		expected SourceInputConfig
	}{
		{
			"module_input_configured",
			&TaskConfig{
				ModuleInput: &ConsulKVSourceInputConfig{
					ConsulKVMonitorConfig: ConsulKVMonitorConfig{
						Path: String("path"),
					},
				},
			},
			&ConsulKVSourceInputConfig{
				ConsulKVMonitorConfig: ConsulKVMonitorConfig{
					Path:       String("path"),
					Recurse:    Bool(false),
					Datacenter: String(""),
					Namespace:  String(""),
				},
			},
		},
		{
			"source_input_configured",
			&TaskConfig{
				DeprecatedSourceInput: &ServicesSourceInputConfig{
					ServicesMonitorConfig: ServicesMonitorConfig{
						Regexp: String(".*"),
					},
				},
			},
			&ServicesSourceInputConfig{
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
		{
			"both_configured",
			&TaskConfig{
				ModuleInput: &ConsulKVSourceInputConfig{
					ConsulKVMonitorConfig: ConsulKVMonitorConfig{
						Path: String("path"),
					},
				},
				DeprecatedSourceInput: &ServicesSourceInputConfig{
					ServicesMonitorConfig: ServicesMonitorConfig{
						Regexp: String(".*"),
					},
				},
			},
			&ConsulKVSourceInputConfig{
				ConsulKVMonitorConfig: ConsulKVMonitorConfig{
					Path:       String("path"),
					Recurse:    Bool(false),
					Datacenter: String(""),
					Namespace:  String(""),
				},
			},
		},
		{
			"none_configured",
			&TaskConfig{},
			EmptyModuleInputConfig(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.i.Finalize(DefaultBufferPeriodConfig(), DefaultWorkingDir)
			// assert.NotEqual(t, tc.expected, tc.i.ModuleInput)
			assert.Equal(t, tc.expected, tc.i.ModuleInput)
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
			&TaskConfig{Services: []string{"service"}, Module: String("path")},
			false,
		},
		{
			"invalid: task name: invalid char",
			&TaskConfig{
				Name:     String("cannot contain spaces"),
				Services: []string{"service"},
				Module:   String("path"),
			},
			false,
		},
		{
			"invalid: task module: missing",
			&TaskConfig{Name: String("task"), Services: []string{"service"}},
			false,
		},
		{
			"invalid: TF version: unsupported version",
			&TaskConfig{
				Name:      String("task"),
				Services:  []string{"service"},
				Module:    String("path"),
				TFVersion: String("0.15.0"),
			},
			false,
		},
		{
			"invalid: provider: duplicate",
			&TaskConfig{
				Name:      String("task"),
				Services:  []string{"api"},
				Module:    String("path"),
				Providers: []string{"providerA", "providerA"},
			},
			false,
		},
		{
			"invalid: provider: duplicate with alias",
			&TaskConfig{
				Name:      String("task"),
				Services:  []string{"api"},
				Module:    String("path"),
				Providers: []string{"providerA", "providerA.alias"},
			},
			false,
		},
		{
			"valid: no cond: services configured",
			&TaskConfig{
				Name:     String("task"),
				Services: []string{"api"},
				Module:   String("path"),
			},
			true,
		},
		{
			"invalid: no cond: no services",
			&TaskConfig{Name: String("task"), Module: String("path")},
			false,
		},
		// catalog-services condition test cases
		{
			"invalid: cs cond: no cond.regexp & no services",
			&TaskConfig{
				Name:      String("task"),
				Module:    String("path"),
				Condition: &CatalogServicesConditionConfig{},
			},
			false,
		},
		{
			"valid: cs-cond: cond.regexp configured & no services",
			&TaskConfig{
				Name:   String("task"),
				Module: String("path"),
				Condition: &CatalogServicesConditionConfig{
					CatalogServicesMonitorConfig{
						Regexp: String(".*"),
					},
				},
			},
			true,
		},
		// services condition test case
		{
			"valid: services cond: no services",
			&TaskConfig{
				Name:   String("task"),
				Module: String("path"),
				Condition: &ServicesConditionConfig{
					ServicesMonitorConfig: ServicesMonitorConfig{
						Regexp: String(".*"),
					},
				},
			},
			true,
		},
		{
			"invalid: services cond: services configured",
			&TaskConfig{
				Name:     String("task"),
				Module:   String("path"),
				Services: []string{"api"},
				Condition: &ServicesConditionConfig{
					ServicesMonitorConfig: ServicesMonitorConfig{
						Regexp: String(""),
					},
				},
			},
			false,
		},
		// consul-kv condition test cases
		{
			"invalid: kv cond: no services",
			&TaskConfig{
				Name:   String("task"),
				Module: String("path"),
				Condition: &ConsulKVConditionConfig{
					ConsulKVMonitorConfig: ConsulKVMonitorConfig{
						Path: String("path"),
					},
				},
			},
			false,
		},
		// schedule condition test cases
		{
			"valid: sched cond: services configured",
			&TaskConfig{
				Name:      String("task"),
				Module:    String("path"),
				Services:  []string{"api"},
				Condition: &ScheduleConditionConfig{String("* * * * * * *")},
			},
			true,
		},
		{
			"valid: sched cond: service module_input configured & no services",
			&TaskConfig{
				Name:      String("task"),
				Module:    String("path"),
				Condition: &ScheduleConditionConfig{String("* * * * * * *")},
				ModuleInput: &ServicesSourceInputConfig{
					ServicesMonitorConfig{Regexp: String(".*")}},
			},
			true,
		},
		{
			"invalid: sched cond: no services & no module_input",
			&TaskConfig{
				Name:      String("task"),
				Module:    String("path"),
				Condition: &ScheduleConditionConfig{String("* * * * * * *")},
			},
			false,
		},
		{
			"invalid: sched cond: services module_input configured & services configured",
			&TaskConfig{
				Name:      String("task"),
				Module:    String("path"),
				Services:  []string{"api"},
				Condition: &ScheduleConditionConfig{String("* * * * * * *")},
				ModuleInput: &ServicesSourceInputConfig{
					ServicesMonitorConfig{Names: []string{"api"}},
				},
			},
			false,
		},
		{
			"invalid: sched cond: kv module_input configured & no services",
			&TaskConfig{
				Name:      String("task"),
				Module:    String("path"),
				Condition: &ScheduleConditionConfig{String("* * * * * * *")},
				ModuleInput: &ConsulKVSourceInputConfig{
					ConsulKVMonitorConfig{
						Path: String("path"),
					},
				},
			},
			false,
		},
		// non-schedule condition test-cases
		{
			"invalid: non-sched cond: module_input configured",
			&TaskConfig{
				Name:   String("task"),
				Module: String("path"),
				Condition: &ServicesConditionConfig{
					ServicesMonitorConfig: ServicesMonitorConfig{
						Regexp: String(".*")}},
				ModuleInput: &ServicesSourceInputConfig{
					ServicesMonitorConfig{Regexp: String(".*")}},
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
					Name:      String("task"),
					Services:  []string{"serviceA", "serviceB"},
					Module:    String("path"),
					Providers: []string{"providerA", "providerB"},
				},
			},
			isValid: true,
		}, {
			name: "two tasks",
			i: []*TaskConfig{
				{
					Name:      String("task"),
					Services:  []string{"serviceA", "serviceB"},
					Module:    String("path"),
					Providers: []string{"providerA", "providerB"},
				},
				{
					Name:      String("task2"),
					Services:  []string{"serviceC"},
					Module:    String("sourceC"),
					Providers: []string{"providerC"},
				},
			},
			isValid: true,
		}, {
			name: "duplicate task names",
			i: []*TaskConfig{
				{
					Name:      String("task"),
					Services:  []string{"serviceA", "serviceB"},
					Module:    String("path"),
					Providers: []string{"providerA", "providerB"},
				}, {
					Name:      String("task"),
					Services:  []string{"serviceA"},
					Module:    String("source2"),
					Providers: []string{"providerA"},
				},
			},
			isValid: false,
		}, {
			name: "one invalid",
			i: []*TaskConfig{
				{
					Name:      String("task"),
					Services:  []string{"serviceA", "serviceB"},
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
					Name:      String("task"),
					Services:  []string{"serviceA", "serviceB"},
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

func TestTaskConfig_FinalizeValidate(t *testing.T) {
	// Tests the full finalize then validate process, particularly how it handles the
	// relationship between task.services and task.condition
	//
	// Relationships:
	// - When catalog-service condition's regexp is nil or empty string, test
	//    how regexp is handled by Finalize() which impacts Validate()

	cases := []struct {
		name   string
		config *TaskConfig
		valid  bool
	}{
		{
			"invalid: no services with catalog-service condition missing regexp",
			&TaskConfig{
				Name:      String("task_a"),
				Module:    String("path"),
				Condition: &CatalogServicesConditionConfig{},
			},
			false,
		},
		{
			"valid: no services with catalog-service condition's regexp is empty string",
			&TaskConfig{
				Name:      String("task_a"),
				Module:    String("path"),
				Services:  []string{"serviceA"},
				Condition: &CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String("")}},
			},
			true,
		},
		{
			"valid: no module_input included with schedule condition",
			&TaskConfig{
				Name:      String("task_a"),
				Module:    String("path"),
				Services:  []string{"serviceA"},
				Condition: &ScheduleConditionConfig{String("* * * * * * *")},
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.config.Finalize(DefaultBufferPeriodConfig(), DefaultWorkingDir)
			err := tc.config.Validate()
			if tc.valid {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "condition",
					"error does not seem to be about condition configuration")
			}
		})
	}
}
