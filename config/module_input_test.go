package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (

	// Success
	testModuleInputServicesSuccess = `
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

	testModuleInputConsulKVSuccess = `
task {
	name = "module_input_task"
	module = "..."
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
	testModuleInputsSuccess = `
task {
	name = "module_input_task"
	module = "..."
	condition "catalog-services" {
		regexp = ".*"
	}
	module_input "services" {
		names = ["api"]
	}
	module_input "consul-kv" {
		path = "my/path"
	}
}`

	// Errors
	testModuleInputServicesUnsupportedFieldError = `
task {
	name = "module_input_task"
	module = "..."
	module_input "services" {
		nonexistent_field = true
	}
	condition "schedule" {
		cron = "* * * * * * *"
	}
}`
	testModuleInputConsulKVUnsupportedFieldError = `
task {
	name = "module_input_task"
	module = "..."
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

func TestModuleInput_DecodeConfig_Success(t *testing.T) {
	// Specifically test decoding module_input configs
	cases := []struct {
		name     string
		expected *ModuleInputConfigs
		config   string
	}{
		{
			name: "services",
			expected: &ModuleInputConfigs{
				&ServicesModuleInputConfig{
					ServicesMonitorConfig{
						Regexp:             String(".*"),
						Names:              []string{},
						Datacenter:         String("dc2"),
						Namespace:          String("ns2"),
						Filter:             String("some-filter"),
						CTSUserDefinedMeta: map[string]string{"key": "value"},
					},
				},
			},
			config: testModuleInputServicesSuccess,
		},
		{
			name: "consul-kv",
			expected: &ModuleInputConfigs{
				&ConsulKVModuleInputConfig{
					ConsulKVMonitorConfig{
						Path:       String("key-path"),
						Datacenter: String("dc2"),
						Namespace:  String("ns2"),
						Recurse:    Bool(true),
					},
				},
			},
			config: testModuleInputConsulKVSuccess,
		},
		{
			name: "multiple unique module_inputs",
			expected: &ModuleInputConfigs{
				&ServicesModuleInputConfig{
					ServicesMonitorConfig{
						Names:              []string{"api"},
						Datacenter:         String(""),
						Namespace:          String(""),
						Filter:             String(""),
						CTSUserDefinedMeta: map[string]string{},
					},
				},
				&ConsulKVModuleInputConfig{
					ConsulKVMonitorConfig{
						Path:       String("my/path"),
						Recurse:    Bool(false),
						Datacenter: String(""),
						Namespace:  String(""),
					},
				},
			},
			config: testModuleInputsSuccess,
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
			require.Equal(t, tc.expected, tasks[0].ModuleInputs)
		})
	}
}

func TestModuleInput_DecodeConfig_Error(t *testing.T) {
	// specifically test decoding condition configs
	cases := []struct {
		name     string
		expected *Config
		config   string
	}{
		{
			name:     "services unsupported field",
			expected: nil,
			config:   testModuleInputServicesUnsupportedFieldError,
		},
		{
			name:     "consul kv unsupported field",
			expected: nil,
			config:   testModuleInputConsulKVUnsupportedFieldError,
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

func TestModuleInputConfigs_Len(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		i        *ModuleInputConfigs
		expected int
	}{
		{
			"nil",
			nil,
			0,
		},
		{
			"happy_path",
			&ModuleInputConfigs{
				&ServicesModuleInputConfig{},
				&ConsulKVModuleInputConfig{},
			},
			2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.i.Len()
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestModuleInputConfigs_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		i        *ModuleInputConfigs
		expected *ModuleInputConfigs
	}{
		{
			"nil",
			nil,
			nil,
		},
		{
			"empty",
			&ModuleInputConfigs{},
			&ModuleInputConfigs{},
		},
		{
			"happy_path",
			&ModuleInputConfigs{
				&ServicesModuleInputConfig{
					ServicesMonitorConfig{
						Regexp:     String("^web.*"),
						Datacenter: String("dc"),
						Namespace:  String("namespace"),
						Filter:     String("filter"),
						CTSUserDefinedMeta: map[string]string{
							"key": "value",
						},
					},
				},
				&ConsulKVModuleInputConfig{
					ConsulKVMonitorConfig{
						Path:       String("key-path"),
						Recurse:    Bool(true),
						Datacenter: String("dc2"),
						Namespace:  String("ns2"),
					},
				},
			},
			&ModuleInputConfigs{
				&ServicesModuleInputConfig{
					ServicesMonitorConfig{
						Regexp:     String("^web.*"),
						Datacenter: String("dc"),
						Namespace:  String("namespace"),
						Filter:     String("filter"),
						CTSUserDefinedMeta: map[string]string{
							"key": "value",
						},
					},
				},
				&ConsulKVModuleInputConfig{
					ConsulKVMonitorConfig{
						Path:       String("key-path"),
						Recurse:    Bool(true),
						Datacenter: String("dc2"),
						Namespace:  String("ns2"),
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.i.Copy()
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestModuleInputConfigs_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ModuleInputConfigs
		b    *ModuleInputConfigs
		r    *ModuleInputConfigs
	}{
		{
			"nil_a",
			nil,
			&ModuleInputConfigs{},
			&ModuleInputConfigs{},
		},
		{
			"nil_b",
			&ModuleInputConfigs{},
			nil,
			&ModuleInputConfigs{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&ModuleInputConfigs{},
			&ModuleInputConfigs{},
			&ModuleInputConfigs{},
		},
		{
			"happy_path_different_type",
			&ModuleInputConfigs{&ServicesModuleInputConfig{}},
			&ModuleInputConfigs{&ConsulKVModuleInputConfig{}},
			&ModuleInputConfigs{
				&ServicesModuleInputConfig{},
				&ConsulKVModuleInputConfig{},
			},
		},
		{
			"happy_path_same_type",
			// will error with Validation()
			&ModuleInputConfigs{&ServicesModuleInputConfig{}},
			&ModuleInputConfigs{&ServicesModuleInputConfig{}},
			&ModuleInputConfigs{
				&ServicesModuleInputConfig{},
				&ServicesModuleInputConfig{},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestModuleInputConfigs_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *ModuleInputConfigs
		r    *ModuleInputConfigs
	}{
		{
			"nil",
			nil,
			nil,
		},
		{
			"empty",
			&ModuleInputConfigs{},
			&ModuleInputConfigs{},
		},
		{
			"happy_path",
			&ModuleInputConfigs{
				&ServicesModuleInputConfig{},
			},
			&ModuleInputConfigs{
				&ServicesModuleInputConfig{
					ServicesMonitorConfig{
						Regexp:             nil,
						Names:              []string{},
						Datacenter:         String(""),
						Namespace:          String(""),
						Filter:             String(""),
						CTSUserDefinedMeta: map[string]string{},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.i.Finalize()
			assert.Equal(t, tc.r, tc.i)
		})
	}
}

func TestModuleInputConfigs_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		services     []string
		condition    ConditionConfig
		moduleInputs *ModuleInputConfigs
		valid        bool
	}{
		{
			name:         "valid: nil",
			moduleInputs: nil,
			valid:        true,
		},
		{
			name:         "valid: empty module_inputs",
			moduleInputs: &ModuleInputConfigs{},
			valid:        true,
		},
		{
			name: "valid: happy path",
			moduleInputs: &ModuleInputConfigs{
				&ServicesModuleInputConfig{},
				&ConsulKVModuleInputConfig{},
			},
			valid: true,
		},
		{
			name:      "valid: happy path with cond-block",
			condition: &CatalogServicesConditionConfig{},
			moduleInputs: &ModuleInputConfigs{
				&ServicesModuleInputConfig{},
				&ConsulKVModuleInputConfig{},
			},
			valid: true,
		},
		{
			name:     "valid: happy path with services list",
			services: []string{"api"},
			moduleInputs: &ModuleInputConfigs{
				&ConsulKVModuleInputConfig{},
			},
			valid: true,
		},
		{
			name: "invalid: module_inputs not unique",
			moduleInputs: &ModuleInputConfigs{
				&ServicesModuleInputConfig{},
				&ConsulKVModuleInputConfig{},
				&ConsulKVModuleInputConfig{},
			},
			valid: false,
		},
		{
			name:     "invalid: services & services module_input configured",
			services: []string{"api"},
			moduleInputs: &ModuleInputConfigs{
				&ServicesModuleInputConfig{},
			},
			valid: false,
		},
		{
			name:      "invalid: cond & module_input same type",
			condition: &ConsulKVConditionConfig{},
			moduleInputs: &ModuleInputConfigs{
				&ConsulKVModuleInputConfig{},
			},
			valid: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.moduleInputs.Validate(tc.services, tc.condition)
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestModuleInputConfigs_GoString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		i        *ModuleInputConfigs
		expected string
	}{
		{
			"nil",
			nil,
			"(*ModuleInputConfigs)(nil)",
		},
		{
			"configured",
			&ModuleInputConfigs{
				&ServicesModuleInputConfig{
					ServicesMonitorConfig: ServicesMonitorConfig{
						Regexp: String("^api$"),
					},
				},
				&ConsulKVModuleInputConfig{
					ConsulKVMonitorConfig: ConsulKVMonitorConfig{
						Path: String("my/path"),
					},
				},
			},
			"{&ServicesModuleInputConfig{&ServicesMonitorConfig{Regexp:^api$, Names:[], " +
				"Datacenter:, Namespace:, Filter:, CTSUserDefinedMeta:map[]}}, " +
				"&ConsulKVModuleInputConfig{&ConsulKVMonitorConfig{Path:my/path, " +
				"Recurse:false, Datacenter:, Namespace:, }}}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.i.GoString()
			assert.Equal(t, tc.expected, actual)
		})
	}
}
