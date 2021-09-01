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
				Regexp:            String(".*"),
				SourceIncludesVar: Bool(true),
				Datacenter:        String("dc2"),
				Namespace:         String("ns2"),
				NodeMeta: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
			"config.hcl",
			`
task {
	name = "condition_task"
	source = "..."
	services = ["api"]
	condition "catalog-services" {
		regexp = ".*"
		source_includes_var = true
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
			"catalog-services: unconfigured",
			false,
			&CatalogServicesConditionConfig{
				Regexp:            String("^api$"),
				SourceIncludesVar: Bool(false),
				Datacenter:        String(""),
				Namespace:         String(""),
				NodeMeta:          map[string]string{},
			},
			"config.hcl",
			`
task {
	name = "condition_task"
	source = "..."
	services = ["api"]
	condition "catalog-services" {
	}
}`,
		},
		{
			"no condition: defaults to services condition",
			false,
			&ServicesConditionConfig{
				Regexp: String(""),
			},
			"config.hcl",
			`
task {
	name = "condition_task"
	source = "..."
	services = ["api"]
}`,
		},
		{
			"services: happy path",
			false,
			&ServicesConditionConfig{
				Regexp: String(".*"),
			},
			"config.hcl",
			`
task {
	name = "services_condition_task"
	source = "..."
	condition "services" {
		regexp = ".*"
	}
}`,
		},
		{
			"services: unconfigured",
			false,
			&ServicesConditionConfig{
				Regexp: String(""),
			},
			"config.hcl",
			`
task {
	name = "condition_task"
	source = "..."
	services = ["api"]
	condition "services" {
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
	source = "..."
	services = ["api"]
	condition "services" {
		nonexistent_field = true
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
	source = "..."
	services = ["api"]
	condition "catalog-services" {
		nonexistent_field = true
	}
}`,
		},
		{
			"nodes: empty",
			false,
			&NodesConditionConfig{
				Datacenter: String(""),
				services:   []string{},
			},
			"config.hcl",
			`
task {
	name = "condition_task"
	source = "..."
	services = []
	condition "nodes" {}
}
`,
		},
		{
			"nodes: happy path",
			false,
			&NodesConditionConfig{
				Datacenter: String("dc2"),
				services:   []string{},
			},
			"config.hcl",
			`
task {
	name = "condition_task"
	source = "..."
	services = []
	condition "nodes" {
		datacenter = "dc2"
	}
}
`,
		},
		{
			"nodes: unsupported field",
			true,
			nil,
			"config.hcl",
			`
task {
	name = "condition_task"
	source = "..."
	services = []
	condition "nodes" {
		nonexistent_field = true
	}
}
`,
		},
		{
			"error: two conditions",
			true,
			nil,
			"config.hcl",
			`
task {
	name = "condition_task"
	source = "..."
	services = ["api"]
	condition "catalog-services" {
	}
	condition "catalog-services" {
		regexp = ".*"
		source_includes_var = false
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
	services = ["api"]
	source = "..."
	condition "nonexistent-condition" {
	}
}`,
		},
		{
			"json happy path",
			false,
			&CatalogServicesConditionConfig{
				Regexp:            String(".*"),
				SourceIncludesVar: Bool(true),
				Datacenter:        String("dc2"),
				Namespace:         String("ns2"),
				NodeMeta: map[string]string{
					"key1": "value1",
					"key2": "value2",
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
		  "source": "Y",
		  "condition": {
			"catalog-services": {
			  "regexp": ".*",
			  "source_includes_var": true,
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
