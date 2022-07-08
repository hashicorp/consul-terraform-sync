package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntentionsModuleInputConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &IntentionsModuleInputConfig{}
	finalizedConf.Finalize()

	cases := []struct {
		name string
		a    *IntentionsModuleInputConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&IntentionsModuleInputConfig{},
		},
		{
			"finalized",
			finalizedConf,
		},
		{
			"happy_path",
			&IntentionsModuleInputConfig{
				IntentionsMonitorConfig{
					Datacenter: String("dc"),
					Namespace:  String("namespace"),
					SourceServices: &IntentionsServicesConfig{
						Regexp: String("^web.*"),
					},
					DestinationServices: &IntentionsServicesConfig{
						Regexp: String("^api.*"),
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

func TestIntentionsModuleInputConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *IntentionsModuleInputConfig
		b    *IntentionsModuleInputConfig
		r    *IntentionsModuleInputConfig
	}{
		{
			"nil_a",
			nil,
			&IntentionsModuleInputConfig{},
			&IntentionsModuleInputConfig{},
		},
		{
			"nil_b",
			&IntentionsModuleInputConfig{},
			nil,
			&IntentionsModuleInputConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&IntentionsModuleInputConfig{},
			&IntentionsModuleInputConfig{},
			&IntentionsModuleInputConfig{},
		},
		{
			"happy_path",
			&IntentionsModuleInputConfig{
				IntentionsMonitorConfig{
					Datacenter: String("datacenter_overidden"),
					Namespace:  nil,
					SourceServices: &IntentionsServicesConfig{
						Regexp: String("^web.*"),
					},
					DestinationServices: &IntentionsServicesConfig{
						Regexp: String("^api.*"),
					},
				},
			},
			&IntentionsModuleInputConfig{
				IntentionsMonitorConfig{
					Datacenter: String("datacenter"),
					Namespace:  String("namespace"),
					SourceServices: &IntentionsServicesConfig{
						Regexp: nil,
					},
					DestinationServices: &IntentionsServicesConfig{
						Regexp: nil,
					},
				},
			},
			&IntentionsModuleInputConfig{
				IntentionsMonitorConfig{
					Datacenter: String("datacenter"),
					Namespace:  String("namespace"),
					SourceServices: &IntentionsServicesConfig{
						Regexp: String("^web.*"),
					},
					DestinationServices: &IntentionsServicesConfig{
						Regexp: String("^api.*"),
					},
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

func TestIntentionsModuleInputConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *IntentionsModuleInputConfig
		r    *IntentionsModuleInputConfig
	}{
		{
			"nil",
			nil,
			nil,
		},
		{
			"happy_path",
			&IntentionsModuleInputConfig{},
			&IntentionsModuleInputConfig{
				IntentionsMonitorConfig{
					Datacenter:          String(""),
					Namespace:           String(""),
					SourceServices:      (*IntentionsServicesConfig)(nil),
					DestinationServices: (*IntentionsServicesConfig)(nil),
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

func TestIntentionsModuleInputConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *IntentionsModuleInputConfig
	}{
		{
			"valid",
			false,
			&IntentionsModuleInputConfig{
				IntentionsMonitorConfig{
					SourceServices: &IntentionsServicesConfig{
						Regexp: String(".*"),
					},
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
			&IntentionsModuleInputConfig{
				IntentionsMonitorConfig{
					SourceServices: &IntentionsServicesConfig{
						Regexp: String("*"),
					},
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

func TestIntentionsModuleInputConfig_TestGoString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		ssv      *IntentionsModuleInputConfig
		expected string
	}{
		{
			"configured services module_input",
			&IntentionsModuleInputConfig{
				IntentionsMonitorConfig{
					Datacenter: String("dc"),
					Namespace:  String("namespace"),
					SourceServices: &IntentionsServicesConfig{
						Regexp: String("^web.*"),
					},
					DestinationServices: &IntentionsServicesConfig{
						Regexp: String("^api.*"),
					},
				},
			},
			"&IntentionsModuleInputConfig{" +
				"&IntentionsMonitorConfig{" +
				"Datacenter:dc, " +
				"Namespace:namespace, " +
				"Source Services Regexp:^web.*, " +
				"Destination Services Regexp:^api.*" +
				"}" +
				"}",
		},
		{
			"nil intentions module_input",
			nil,
			"(*IntentionsModuleInputConfig)(nil)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.ssv.GoString()
			require.Equal(t, tc.expected, actual)
		})
	}
}
