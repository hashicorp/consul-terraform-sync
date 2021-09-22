package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (

	// Success
	testSourceInputServicesSuccess = `
task {
	name = "source_input_task"
	source = "..."
	source_input "services" {
		regexp = ".*"
	}
	condition "schedule" {
		cron = "* * * * * * *"
	}
}`
	testSourceInputServicesUnconfiguredSuccess = `
task {
	name = "condition_task"
	source = "..."
	services = ["api"]
	source_input "services" {
	}
	condition "schedule" {
		cron = "* * * * * * *"
	}
}`

	// Errors
	testSourceInputServicesUnsupportedFieldError = `
task {
	name = "condition_task"
	source = "..."
	services = ["api"]
	source_input "services" {
		nonexistent_field = true
	}
	condition "schedule" {
		cron = "* * * * * * *"
	}
}`

	testFileName = "config.hcl"
)

func TestSourceInput_DefaultSourceInputConfig(t *testing.T) {
	e := &ServicesSourceInputConfig{
		ServicesMonitorConfig{
			Regexp: String(""),
		},
	}
	a := DefaultSourceInputConfig()
	require.Equal(t, e, a)
}

func TestSourceInput_DecodeConfig_Success(t *testing.T) {
	// Specifically test decoding source_input configs
	cases := []struct {
		name     string
		expected SourceInputConfig
		filename string
		config   string
	}{
		{
			name: "happy path",
			expected: &ServicesSourceInputConfig{
				ServicesMonitorConfig{
					Regexp: String(".*"),
				},
			},
			config: testSourceInputServicesSuccess,
		},
		{
			name: "un-configured",
			expected: &ServicesSourceInputConfig{
				ServicesMonitorConfig{
					Regexp: String(""),
				},
			},
			config: testSourceInputServicesUnconfiguredSuccess,
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

			// confirm source_input decoding
			tasks := *config.Tasks
			require.Equal(t, 1, len(tasks))
			require.Equal(t, tc.expected, tasks[0].SourceInput)
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
			name:     "unsupported field",
			expected: nil,
			config:   testSourceInputServicesUnsupportedFieldError,
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
