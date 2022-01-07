package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (

	// Success
	testSourceInputServicesSuccess = `
task {
	name = "module_input_task"
	module = "..."
	module_input "services" {
		regexp = ".*"
		datacenter = "dc2"
		namespace = "ns2"
		filter = "some-filter"
		cts_user_defined_meta {
			key = "value"
		}
	}
	condition "schedule" {
		cron = "* * * * * * *"
	}
}`

	testSourceInputConsulKVSuccess = `
task {
	name = "condition_task"
	module = "..."
	services = ["api"]
	condition "schedule" {
		cron = "* * * * * * *"
	}
	module_input "consul-kv" {
		path = "key-path"
		namespace = "ns2"
		datacenter = "dc2"
		recurse = true
	}
}`

	// Errors
	testSourceInputServicesUnsupportedFieldError = `
task {
	name = "condition_task"
	module = "..."
	services = ["api"]
	module_input "services" {
		nonexistent_field = true
	}
	condition "schedule" {
		cron = "* * * * * * *"
	}
}`
	testSourceInputConsulKVUnsupportedFieldError = `
task {
	name = "condition_task"
	module = "..."
	services = ["api"]
	condition "schedule" {
		cron = "* * * * * * *"
	}
	module_input "consul-kv" {
		path = "key-path"
        use_as_module_input = true
		namespace = "ns2"
		datacenter = "dc2"
		recurse = true
	}
}`

	testFileName = "config.hcl"
)

func TestSourceInput_DecodeConfig_Success(t *testing.T) {
	// Specifically test decoding module_input configs
	cases := []struct {
		name     string
		expected SourceInputConfig
		filename string
		config   string
	}{
		{
			name: "services happy path",
			expected: &ServicesSourceInputConfig{
				ServicesMonitorConfig{
					Regexp:             String(".*"),
					Names:              []string{},
					Datacenter:         String("dc2"),
					Namespace:          String("ns2"),
					Filter:             String("some-filter"),
					CTSUserDefinedMeta: map[string]string{"key": "value"},
				},
			},
			config: testSourceInputServicesSuccess,
		},
		{
			name: "consul-kv: happy path",
			expected: &ConsulKVSourceInputConfig{
				ConsulKVMonitorConfig{
					Path:       String("key-path"),
					Datacenter: String("dc2"),
					Namespace:  String("ns2"),
					Recurse:    Bool(true),
				},
			},
			config: testSourceInputConsulKVSuccess,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// replicate decoding process used by cts cli
			config, err := decodeConfig([]byte(tc.config), testFileName)
			require.NoError(t, err)
			config.Finalize()
			err = config.Validate()
			require.NoError(t, err)

			// confirm module_input decoding
			tasks := *config.Tasks
			require.Equal(t, 1, len(tasks))
			require.Equal(t, tc.expected, tasks[0].ModuleInput)
		})
	}
}

func TestSourceInput_DecodeConfig_Error(t *testing.T) {
	// specifically test decoding condition configs
	cases := []struct {
		name     string
		expected *Config
		config   string
	}{
		{
			name:     "services unsupported field",
			expected: nil,
			config:   testSourceInputServicesUnsupportedFieldError,
		},
		{
			name:     "consul kv unsupported field",
			expected: nil,
			config:   testSourceInputConsulKVUnsupportedFieldError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := decodeConfig([]byte(tc.config), testFileName)
			require.Error(t, err)
			require.Equal(t, tc.expected, config)
		})
	}
}
