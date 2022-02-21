package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCondition_DecodeConfig(t *testing.T) {
	// specifically test decoding condition configs
	cases := []struct {
		name      string
		expectErr bool
		expected  ConditionConfig
		filename  string
		config    string
	}{
		{
			"catalog-services: happy path",
			false,
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{
					Regexp:           String(".*"),
					UseAsModuleInput: Bool(true),
					Datacenter:       String("dc2"),
					Namespace:        String("ns2"),
					NodeMeta: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
			},
			"config.hcl",
			`
task {
	name = "condition_task"
	module = "..."
	condition "catalog-services" {
		regexp = ".*"
		use_as_module_input = true
		namespace = "ns2"
		datacenter = "dc2"
		node_meta {
		  "key1" = "value1"
		  "key2" = "value2"
		}
	}
}`,
		},
		{
			"no condition - deprecated",
			false,
			EmptyConditionConfig(),
			"config.hcl",
			`
task {
	name = "condition_task"
	module = "..."
	services = ["api"]
}`,
		},
		{
			"services: happy path",
			false,
			&ServicesConditionConfig{
				ServicesMonitorConfig: ServicesMonitorConfig{
					Regexp:     String(".*"),
					Names:      []string{},
					Datacenter: String("dc"),
					Namespace:  String("namespace"),
					Filter:     String("filter"),
					CTSUserDefinedMeta: map[string]string{
						"key": "value",
					},
				},
				UseAsModuleInput: Bool(true),
			},
			"config.hcl",
			`
task {
	name = "services_condition_task"
	module = "..."
	condition "services" {
		regexp = ".*"
		datacenter = "dc"
		namespace = "namespace"
		filter = "filter"
		cts_user_defined_meta {
			key = "value"
		}
	}
}`,
		},
		{
			"services: unsupported field",
			true,
			nil,
			"config.hcl",
			`
task {
	name = "condition_task"
	module = "..."
	condition "services" {
		nonexistent_field = true
	}
}`,
		},
		{
			"schedule: happy path",
			false,
			&ScheduleConditionConfig{
				Cron: String("* * * * * * *"),
			},
			"config.hcl",
			`
task {
	name = "schedule_condition_task"
	module = "..."
	condition "schedule" {
		cron = "* * * * * * *"
	}
}`,
		},
		{
			"catalog-services: unsupported field",
			true,
			nil,
			"config.hcl",
			`
task {
	name = "condition_task"
	module = "..."
	condition "catalog-services" {
		nonexistent_field = true
	}
}`,
		},
		{
			"error: two conditions",
			true,
			nil,
			"config.hcl",
			`
task {
	name = "condition_task"
	module = "..."
	condition "catalog-services" {
	}
	condition "catalog-services" {
		regexp = ".*"
		use_as_module_input = false
	}
}`,
		},
		{
			"consul-kv: happy path",
			false,
			&ConsulKVConditionConfig{
				ConsulKVMonitorConfig: ConsulKVMonitorConfig{
					Path:       String("key-path"),
					Datacenter: String("dc2"),
					Namespace:  String("ns2"),
					Recurse:    Bool(true),
				},
				UseAsModuleInput: Bool(true),
			},
			"config.hcl",
			`
task {
	name = "condition_task"
	module = "..."
	condition "consul-kv" {
		path = "key-path"
		use_as_module_input = true
		namespace = "ns2"
		datacenter = "dc2"
		recurse = true
	}
}`,
		},
		{
			"error: nonexistent condition type",
			true,
			nil,
			"config.hcl",
			`
task {
	name = "condition_task"
	module = "..."
	condition "nonexistent-condition" {
	}
}`,
		},
		{
			"json happy path",
			false,
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{
					Regexp:           String(".*"),
					UseAsModuleInput: Bool(true),
					Datacenter:       String("dc2"),
					Namespace:        String("ns2"),
					NodeMeta: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
			},
			"config.json",
			`
{
	"task": [
		{
		  "name": "task",
		  "description": "automate services for X to do Y",
		  "services": ["serviceA", "serviceB", "serviceC"],
		  "providers": ["X"],
		  "module": "Y",
		  "condition": {
			"catalog-services": {
			  "regexp": ".*",
			  "use_as_module_input": true,
			  "datacenter": "dc2",
			  "namespace": "ns2",
			  "node_meta": {
				"key1": "value1",
				"key2": "value2"
			  }
			}
		  }
		}
]}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// replicate decoding process used by cts cli
			config, err := decodeConfig([]byte(tc.config), tc.filename)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			config.Finalize()
			err = config.Validate()
			require.NoError(t, err)

			// confirm that condition decoding
			tasks := *config.Tasks
			require.Equal(t, 1, len(tasks))
			require.Equal(t, tc.expected, tasks[0].Condition)
		})
	}
}
