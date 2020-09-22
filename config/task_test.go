package config

import (
	"fmt"
	"testing"

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
				Description: String("description"),
				Name:        String("name"),
				Providers:   []string{"provider"},
				Services:    []string{"service"},
				Source:      String("source"),
				Version:     String("0.0.0"),
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
				BufferPeriod: DefaultTaskBufferPeriodConfig(),
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
				BufferPeriod: DefaultTaskBufferPeriodConfig(),
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			tc.i.Finalize()
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
			"nil",
			nil,
			false,
		},
		{
			"empty",
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
			},
			true,
		},
		{
			"missing name",
			&TaskConfig{Services: []string{"service"}, Source: String("source")},
			false,
		},
		{
			"missing service",
			&TaskConfig{Name: String("task"), Source: String("source")},
			false,
		},
		{
			"missing source",
			&TaskConfig{Name: String("task"), Services: []string{"service"}},
			false,
		},
		{
			"duplicate provider",
			&TaskConfig{
				Name:      String("task"),
				Services:  []string{"serviceA", "serviceB"},
				Source:    String("source"),
				Providers: []string{"providerA", "providerA"},
			},
			false,
		},
		{
			"duplicate provider with alias",
			&TaskConfig{
				Name:      String("task"),
				Services:  []string{"serviceA", "serviceB"},
				Source:    String("source"),
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
