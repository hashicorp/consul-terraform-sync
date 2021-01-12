package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceConfig_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ServiceConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&ServiceConfig{},
		},
		{
			"same_enabled",
			&ServiceConfig{
				Description: String("description"),
				Name:        String("name"),
				Namespace:   String("namespace"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value",
				},
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

func TestServiceConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ServiceConfig
		b    *ServiceConfig
		r    *ServiceConfig
	}{
		{
			"nil_a",
			nil,
			&ServiceConfig{},
			&ServiceConfig{},
		},
		{
			"nil_b",
			&ServiceConfig{},
			nil,
			&ServiceConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&ServiceConfig{},
			&ServiceConfig{},
			&ServiceConfig{},
		},
		{
			"description_overrides",
			&ServiceConfig{Description: String("description")},
			&ServiceConfig{Description: String("describe")},
			&ServiceConfig{Description: String("describe")},
		},
		{
			"description_empty_one",
			&ServiceConfig{Description: String("description")},
			&ServiceConfig{},
			&ServiceConfig{Description: String("description")},
		},
		{
			"description_empty_two",
			&ServiceConfig{},
			&ServiceConfig{Description: String("description")},
			&ServiceConfig{Description: String("description")},
		},
		{
			"description_same",
			&ServiceConfig{Description: String("description")},
			&ServiceConfig{Description: String("description")},
			&ServiceConfig{Description: String("description")},
		},
		{
			"name_overrides",
			&ServiceConfig{Name: String("name")},
			&ServiceConfig{Name: String("service")},
			&ServiceConfig{Name: String("service")},
		},
		{
			"name_empty_one",
			&ServiceConfig{Name: String("name")},
			&ServiceConfig{},
			&ServiceConfig{Name: String("name")},
		},
		{
			"name_empty_two",
			&ServiceConfig{},
			&ServiceConfig{Name: String("name")},
			&ServiceConfig{Name: String("name")},
		},
		{
			"name_same",
			&ServiceConfig{Name: String("name")},
			&ServiceConfig{Name: String("name")},
			&ServiceConfig{Name: String("name")},
		},
		{
			"namespace_overrides",
			&ServiceConfig{Namespace: String("namespace")},
			&ServiceConfig{Namespace: String("")},
			&ServiceConfig{Namespace: String("")},
		},
		{
			"namespace_empty_one",
			&ServiceConfig{Namespace: String("namespace")},
			&ServiceConfig{},
			&ServiceConfig{Namespace: String("namespace")},
		},
		{
			"namespace_empty_two",
			&ServiceConfig{},
			&ServiceConfig{Namespace: String("namespace")},
			&ServiceConfig{Namespace: String("namespace")},
		},
		{
			"namespace_same",
			&ServiceConfig{Namespace: String("namespace")},
			&ServiceConfig{Namespace: String("namespace")},
			&ServiceConfig{Namespace: String("namespace")},
		},
		{
			"cts_user_defined_meta_overrides",
			&ServiceConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
			&ServiceConfig{CTSUserDefinedMeta: map[string]string{"key": "new-value"}},
			&ServiceConfig{CTSUserDefinedMeta: map[string]string{"key": "new-value"}},
		},
		{
			"cts_user_defined_meta_empty_one",
			&ServiceConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
			&ServiceConfig{},
			&ServiceConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
		},
		{
			"cts_user_defined_meta_empty_two",
			&ServiceConfig{},
			&ServiceConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
			&ServiceConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
		},
		{
			"cts_user_defined_meta_same",
			&ServiceConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
			&ServiceConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
			&ServiceConfig{CTSUserDefinedMeta: map[string]string{"key": "value"}},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestServiceConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *ServiceConfig
		r    *ServiceConfig
	}{
		{
			"empty",
			&ServiceConfig{},
			&ServiceConfig{
				Datacenter:         String(""),
				Description:        String(""),
				ID:                 String(""),
				Name:               String(""),
				Namespace:          String(""),
				Tag:                String(""),
				CTSUserDefinedMeta: map[string]string{},
			},
		},
		{
			"with_name",
			&ServiceConfig{
				Name: String("service"),
			},
			&ServiceConfig{
				Datacenter:         String(""),
				Description:        String(""),
				ID:                 String("service"),
				Name:               String("service"),
				Namespace:          String(""),
				Tag:                String(""),
				CTSUserDefinedMeta: map[string]string{},
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

func TestServiceConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		i       *ServiceConfig
		isValid bool
	}{
		{
			"nil",
			nil,
			false,
		},
		{
			"empty",
			&ServiceConfig{},
			false,
		},
		{
			"valid",
			&ServiceConfig{
				Name: String("task"),
			},
			true,
		},
		{
			"missing name",
			&ServiceConfig{Description: String("description")},
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

func TestServiceConfigs_CTSUserDefinedMeta(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		conf        *ServiceConfigs
		serviceList []string
		expected    map[string]map[string]string
	}{
		{
			"nil",
			nil,
			[]string{"a"},
			nil,
		}, {
			"empty",
			&ServiceConfigs{},
			[]string{"a"},
			make(map[string]map[string]string),
		}, {
			"meta",
			&ServiceConfigs{{
				Name: String("a"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value",
				},
			}},
			[]string{"a"},
			map[string]map[string]string{
				"a": {"key": "value"},
			},
		}, {
			"no meta",
			&ServiceConfigs{{
				Name: String("a"),
			}},
			[]string{"a"},
			map[string]map[string]string{},
		}, {
			"meta by id",
			&ServiceConfigs{{
				ID:   String("a-with-meta"),
				Name: String("a"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value",
				},
			}},
			[]string{"a-with-meta"},
			map[string]map[string]string{
				"a": {"key": "value"},
			},
		}, {
			"service id with identical service",
			&ServiceConfigs{{
				ID:   String("a-with-meta"),
				Name: String("a"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value",
				},
			}, {
				Name: String("a"),
				CTSUserDefinedMeta: map[string]string{
					"key": "this should not be selected",
				},
			}},
			[]string{"a-with-meta"},
			map[string]map[string]string{
				"a": {"key": "value"},
			},
		}, {
			"service name with identical service",
			&ServiceConfigs{{
				Name: String("a"),
				CTSUserDefinedMeta: map[string]string{
					"a": "b",
				},
			}, {
				ID:   String("a-with-meta"),
				Name: String("a"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value",
				},
			}},
			[]string{"a"},
			map[string]map[string]string{
				"a": {"a": "b"},
			},
		}, {
			"multiple",
			&ServiceConfigs{{
				Name: String("a"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value a",
				},
			}, {
				Name: String("b"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value b",
				},
			}, {
				Name: String("c"),
				CTSUserDefinedMeta: map[string]string{
					"key": "value c",
				},
			}},
			[]string{"a", "b"},
			map[string]map[string]string{
				"a": {"key": "value a"},
				"b": {"key": "value b"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.conf.CTSUserDefinedMeta(tc.serviceList)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
