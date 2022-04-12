package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServicesModuleInputConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &ServicesModuleInputConfig{}
	finalizedConf.Finalize()

	cases := []struct {
		name string
		a    *ServicesModuleInputConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&ServicesModuleInputConfig{},
		},
		{
			"finalized",
			finalizedConf,
		},
		{
			"happy_path",
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
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.a.Copy()
			if tc.a == nil {
				// returned nil interface has nil type, which is unequal to tc.a
				assert.Nil(t, r)
			} else {
				assert.Equal(t, tc.a, r)
			}
		})
	}
}

func TestServicesModuleInputConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ServicesModuleInputConfig
		b    *ServicesModuleInputConfig
		r    *ServicesModuleInputConfig
	}{
		{
			"nil_a",
			nil,
			&ServicesModuleInputConfig{},
			&ServicesModuleInputConfig{},
		},
		{
			"nil_b",
			&ServicesModuleInputConfig{},
			nil,
			&ServicesModuleInputConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&ServicesModuleInputConfig{},
			&ServicesModuleInputConfig{},
			&ServicesModuleInputConfig{},
		},
		{
			"happy_path",
			&ServicesModuleInputConfig{
				ServicesMonitorConfig{
					Regexp:             String("regexp"),
					Datacenter:         String("datacenter_overriden"),
					Namespace:          nil,
					Filter:             nil,
					CTSUserDefinedMeta: map[string]string{},
				},
			},
			&ServicesModuleInputConfig{
				ServicesMonitorConfig{
					Regexp:             nil,
					Datacenter:         String("datacenter"),
					Namespace:          String("namespace"),
					Filter:             nil,
					CTSUserDefinedMeta: map[string]string{},
				},
			},
			&ServicesModuleInputConfig{
				ServicesMonitorConfig{
					Regexp:             String("regexp"),
					Datacenter:         String("datacenter"),
					Namespace:          String("namespace"),
					Filter:             nil,
					CTSUserDefinedMeta: map[string]string{},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			if tc.r == nil {
				// returned nil interface has nil type, which is unequal to tc.r
				assert.Nil(t, r)
			} else {
				assert.Equal(t, tc.r, r)
			}
		})
	}
}

func TestServicesModuleInputConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *ServicesModuleInputConfig
		r    *ServicesModuleInputConfig
	}{
		{
			"nil",
			nil,
			nil,
		},
		{
			"happy_path",
			&ServicesModuleInputConfig{},
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.i.Finalize()
			assert.Equal(t, tc.r, tc.i)
		})
	}
}

func TestServicesModuleInputConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *ServicesModuleInputConfig
	}{
		{
			"valid",
			false,
			&ServicesModuleInputConfig{
				ServicesMonitorConfig{
					Regexp: String(".*"),
				},
			},
		},
		{
			"nil",
			false,
			nil,
		},
		{
			"invalid",
			true,
			&ServicesModuleInputConfig{
				ServicesMonitorConfig{
					Regexp: String("*"),
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.c.Validate()
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGoString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		ssv      *ServicesModuleInputConfig
		expected string
	}{
		{
			"configured services module_input",
			&ServicesModuleInputConfig{
				ServicesMonitorConfig{
					Regexp:     String("^api$"),
					Datacenter: String("dc2"),
					Namespace:  String("ns2"),
					Filter:     String("some-filter"),
					CTSUserDefinedMeta: map[string]string{
						"key": "value",
					},
				},
			},
			"&ServicesModuleInputConfig{" +
				"&ServicesMonitorConfig{" +
				"Regexp:^api$, " +
				"Names:[], " +
				"Datacenter:dc2, " +
				"Namespace:ns2, " +
				"Filter:some-filter, " +
				"CTSUserDefinedMeta:map[key:value]" +
				"}" +
				"}",
		},
		{
			"nil services module_input",
			nil,
			"(*ServicesModuleInputConfig)(nil)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.ssv.GoString()
			require.Equal(t, tc.expected, actual)
		})
	}
}
