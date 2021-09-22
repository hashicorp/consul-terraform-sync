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
				Source:      String("source"),
				Version:     String("0.0.0"),
				Enabled:     Bool(true),
				Condition: &CatalogServicesConditionConfig{
					CatalogServicesMonitorConfig{
						Regexp:            String(".*"),
						SourceIncludesVar: Bool(true),
						Datacenter:        String("dc2"),
						Namespace:         String("ns2"),
						NodeMeta: map[string]string{
							"key": "value",
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
			&TaskConfig{Source: String("source")},
			&TaskConfig{Source: String("")},
			&TaskConfig{Source: String("")},
		},
		{
			"source_empty_one",
			&TaskConfig{Source: String("source")},
			&TaskConfig{},
			&TaskConfig{Source: String("source")},
		},
		{
			"source_empty_two",
			&TaskConfig{},
			&TaskConfig{Source: String("source")},
			&TaskConfig{Source: String("source")},
		},
		{
			"source_same",
			&TaskConfig{Source: String("source")},
			&TaskConfig{Source: String("source")},
			&TaskConfig{Source: String("source")},
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
			&TaskConfig{SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("")}}},
			&TaskConfig{SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("")}}},
		},
		{
			"source_input_empty_one",
			&TaskConfig{SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{},
			&TaskConfig{SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
		},
		{
			"source_input_empty_two",
			&TaskConfig{},
			&TaskConfig{SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
		},
		{
			"source_input_same",
			&TaskConfig{SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
			&TaskConfig{SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}}},
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
				Source:       String(""),
				VarFiles:     []string{},
				Version:      String(""),
				TFVersion:    String(""),
				BufferPeriod: DefaultBufferPeriodConfig(),
				Enabled:      Bool(true),
				Condition:    DefaultConditionConfig(),
				WorkingDir:   String("sync-tasks"),
				SourceInput:  DefaultSourceInputConfig(),
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
				Source:       String(""),
				VarFiles:     []string{},
				Version:      String(""),
				TFVersion:    String(""),
				BufferPeriod: DefaultBufferPeriodConfig(),
				Enabled:      Bool(true),
				Condition:    DefaultConditionConfig(),
				WorkingDir:   String("sync-tasks/task"),
				SourceInput:  DefaultSourceInputConfig(),
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
				Source:      String(""),
				VarFiles:    []string{},
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
				SourceInput: DefaultSourceInputConfig(),
			},
		},
		{
			"with_services_source_input",
			&TaskConfig{
				Name:        String("task"),
				Condition:   &ScheduleConditionConfig{},
				SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{String("^api$")}},
			},
			&TaskConfig{
				Description: String(""),
				Name:        String("task"),
				Providers:   []string{},
				Services:    []string{},
				Source:      String(""),
				VarFiles:    []string{},
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
				SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{String("^api$")}},
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
			"valid",
			&TaskConfig{
				Name:      String("task"),
				Services:  []string{"serviceA", "serviceB"},
				Source:    String("source"),
				Providers: []string{"providerA", "providerB"},
				Condition: DefaultConditionConfig(),
			},
			true,
		},
		{
			"valid: missing condition",
			&TaskConfig{
				Name:        String("task"),
				Services:    []string{"serviceA", "serviceB"},
				Source:      String("source"),
				Providers:   []string{"providerA", "providerB"},
				SourceInput: DefaultSourceInputConfig(),
			},
			true,
		},
		{
			"valid: missing services with catalog-services condition regexp",
			&TaskConfig{
				Name:      String("task"),
				Source:    String("source"),
				Providers: []string{"providerA", "providerB"},
				Condition: &CatalogServicesConditionConfig{
					CatalogServicesMonitorConfig{
						Regexp: String(".*"),
					},
				},
				SourceInput: DefaultSourceInputConfig(),
			},
			true,
		},
		{
			"valid: no services with services condition regexp",
			&TaskConfig{
				Name:      String("task"),
				Source:    String("source"),
				Providers: []string{"providerA", "providerB"},
				Condition: &ServicesConditionConfig{
					ServicesMonitorConfig{
						Regexp: String(".*"),
					},
				},
				SourceInput: DefaultSourceInputConfig(),
			},
			true,
		},
		{
			"valid: service with schedule condition",
			&TaskConfig{
				Name:        String("task"),
				Source:      String("source"),
				Services:    []string{"serviceA", "serviceB"},
				Condition:   &ScheduleConditionConfig{String("* * * * * * *")},
				SourceInput: DefaultSourceInputConfig(),
			},
			true,
		},
		{
			"valid: service with schedule condition and non empty source_input",
			&TaskConfig{
				Name:        String("task"),
				Source:      String("source"),
				Condition:   &ScheduleConditionConfig{String("* * * * * * *")},
				SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}},
			},
			true,
		},
		{
			"invalid: missing name",
			&TaskConfig{Services: []string{"service"}, Source: String("source")},
			false,
		},
		{
			"invalid: invalid name",
			&TaskConfig{
				Name:     String("cannot contain spaces"),
				Services: []string{"service"},
				Source:   String("source"),
			},
			false,
		},
		{
			"invalid: no service with no condition",
			&TaskConfig{Name: String("task"), Source: String("source")},
			false,
		},
		{
			"invalid: no service with services condition",
			&TaskConfig{
				Name:   String("task"),
				Source: String("source"),
				Condition: &ServicesConditionConfig{
					ServicesMonitorConfig{
						Regexp: String(""),
					},
				},
				SourceInput: DefaultSourceInputConfig(),
			},
			false,
		},
		{
			"invalid: missing service with catalog-service condition and no regexp",
			&TaskConfig{
				Name:        String("task"),
				Source:      String("source"),
				Condition:   &CatalogServicesConditionConfig{},
				SourceInput: DefaultSourceInputConfig(),
			},
			false,
		},
		{
			"invalid: missing service with schedule condition",
			&TaskConfig{
				Name:        String("task"),
				Source:      String("source"),
				Condition:   &ScheduleConditionConfig{String("* * * * * * *")},
				SourceInput: DefaultSourceInputConfig(),
			},
			false,
		},
		{
			"invalid: invalid catalog-service condition (bad regexp)",
			&TaskConfig{
				Name:        String("task"),
				Source:      String("source"),
				Condition:   &CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String("*")}},
				SourceInput: DefaultSourceInputConfig(),
			},
			false,
		},
		{
			"invalid: missing source",
			&TaskConfig{Name: String("task"), Services: []string{"service"}},
			false,
		},
		{
			"invalid: unsupported TF version per task",
			&TaskConfig{
				Name:        String("task"),
				Services:    []string{"service"},
				Source:      String("source"),
				TFVersion:   String("0.15.0"),
				SourceInput: DefaultSourceInputConfig(),
			},
			false,
		},
		{
			"invalid: duplicate provider",
			&TaskConfig{
				Name:        String("task"),
				Services:    []string{"serviceA", "serviceB"},
				Source:      String("source"),
				Providers:   []string{"providerA", "providerA"},
				SourceInput: DefaultSourceInputConfig(),
			},
			false,
		},
		{
			"invalid: duplicate provider with alias",
			&TaskConfig{
				Name:        String("task"),
				Services:    []string{"serviceA", "serviceB"},
				Source:      String("source"),
				Providers:   []string{"providerA", "providerA.alias"},
				SourceInput: DefaultSourceInputConfig(),
			},
			false,
		},
		{
			"invalid: invalid service condition (bad regexp)",
			&TaskConfig{
				Name:        String("task"),
				Source:      String("source"),
				Condition:   &ServicesConditionConfig{ServicesMonitorConfig{Regexp: String("*")}},
				SourceInput: DefaultSourceInputConfig(),
			},
			false,
		},
		{
			"invalid: services and service condition regexp both provided",
			&TaskConfig{
				Name:        String("task"),
				Source:      String("source"),
				Services:    []string{"serviceA", "serviceB"},
				Condition:   &ServicesConditionConfig{ServicesMonitorConfig{Regexp: String("^service.*")}},
				SourceInput: DefaultSourceInputConfig(),
			},
			false,
		},
		{
			"missing services with consul-kv condition",
			&TaskConfig{
				Name:   String("task"),
				Source: String("source"),
				Condition: &ConsulKVConditionConfig{
					ConsulKVMonitorConfig: ConsulKVMonitorConfig{
						Path: String("path"),
					},
				},
				SourceInput: DefaultSourceInputConfig(),
			},
			false,
		},
		{
			"invalid: schedule condition provided with no services and empty source_input",
			&TaskConfig{
				Name:        String("task"),
				Source:      String("source"),
				Condition:   &ScheduleConditionConfig{String("* * * * * * *")},
				SourceInput: DefaultSourceInputConfig(),
			},
			false,
		},
		{
			"invalid: schedule condition provided with no services and nil source_input regex",
			&TaskConfig{
				Name:      String("task"),
				Source:    String("source"),
				Condition: &ScheduleConditionConfig{String("* * * * * * *")},
				SourceInput: &ServicesSourceInputConfig{
					ServicesMonitorConfig{
						Regexp: nil,
					},
				},
			},
			false,
		},
		{
			"invalid: services provided along with services source_input",
			&TaskConfig{
				Name:        String("task"),
				Source:      String("source"),
				Services:    []string{"serviceA", "serviceB"},
				Condition:   &ScheduleConditionConfig{String("* * * * * * *")},
				SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}},
			},
			false,
		},
		{
			"invalid: configured source_input provided with non schedule condition",
			&TaskConfig{
				Name:        String("task"),
				Source:      String("source"),
				Condition:   &ServicesConditionConfig{ServicesMonitorConfig{Regexp: String(".*")}},
				SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String(".*")}},
			},
			false,
		},
		{
			"invalid: source_input bad regex",
			&TaskConfig{
				Name:        String("task"),
				Source:      String("source"),
				Condition:   &ServicesConditionConfig{ServicesMonitorConfig{Regexp: String(".*")}},
				SourceInput: &ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("*")}},
			},
			false,
		},
		{
			"invalid: missing services with consul-kv source input",
			&TaskConfig{
				Name:      String("task"),
				Source:    String("source"),
				Condition: &ScheduleConditionConfig{String("* * * * * * *")},
				SourceInput: &ConsulKVSourceInputConfig{
					ConsulKVMonitorConfig{
						Path: String("path"),
					},
				},
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
					Name:        String("task"),
					Services:    []string{"serviceA", "serviceB"},
					Source:      String("source"),
					Providers:   []string{"providerA", "providerB"},
					SourceInput: DefaultSourceInputConfig(),
				},
			},
			isValid: true,
		}, {
			name: "two tasks",
			i: []*TaskConfig{
				{
					Name:        String("task"),
					Services:    []string{"serviceA", "serviceB"},
					Source:      String("source"),
					Providers:   []string{"providerA", "providerB"},
					SourceInput: DefaultSourceInputConfig(),
				},
				{
					Name:        String("task2"),
					Services:    []string{"serviceC"},
					Source:      String("sourceC"),
					Providers:   []string{"providerC"},
					SourceInput: DefaultSourceInputConfig(),
				},
			},
			isValid: true,
		}, {
			name: "duplicate task names",
			i: []*TaskConfig{
				{
					Name:        String("task"),
					Services:    []string{"serviceA", "serviceB"},
					Source:      String("source"),
					Providers:   []string{"providerA", "providerB"},
					SourceInput: DefaultSourceInputConfig(),
				}, {
					Name:        String("task"),
					Services:    []string{"serviceA"},
					Source:      String("source2"),
					Providers:   []string{"providerA"},
					SourceInput: DefaultSourceInputConfig(),
				},
			},
			isValid: false,
		}, {
			name: "one invalid",
			i: []*TaskConfig{
				{
					Name:        String("task"),
					Services:    []string{"serviceA", "serviceB"},
					Source:      String("source"),
					Providers:   []string{"providerA", "providerB"},
					SourceInput: DefaultSourceInputConfig(),
				}, {
					Name: String("invalid"),
				},
			},
			isValid: false,
		}, {
			name: "duplicate provider instances",
			i: []*TaskConfig{
				{
					Name:        String("task"),
					Providers:   []string{"provider.A", "provider.B"},
					SourceInput: DefaultSourceInputConfig(),
				},
			},
			isValid: false,
		}, {
			name: "unsupported TF version per task",
			i: []*TaskConfig{
				{
					Name:        String("task"),
					Services:    []string{"serviceA", "serviceB"},
					Source:      String("source"),
					TFVersion:   String("0.15.0"),
					SourceInput: DefaultSourceInputConfig(),
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
				Source:    String("source"),
				Condition: &CatalogServicesConditionConfig{},
			},
			false,
		},
		{
			"valid: services with catalog-service condition missing regexp",
			&TaskConfig{
				Name:      String("task_a"),
				Source:    String("source"),
				Services:  []string{"serviceA"},
				Condition: &CatalogServicesConditionConfig{},
			},
			true,
		},
		{
			"valid: no services with catalog-service condition's regexp is empty string",
			&TaskConfig{
				Name:      String("task_a"),
				Source:    String("source"),
				Services:  []string{"serviceA"},
				Condition: &CatalogServicesConditionConfig{CatalogServicesMonitorConfig{Regexp: String("")}},
			},
			true,
		},
		{
			"valid: no source_input included with schedule condition",
			&TaskConfig{
				Name:      String("task_a"),
				Source:    String("source"),
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
