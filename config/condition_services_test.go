package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServicesConditionConfig_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ServicesConditionConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&ServicesConditionConfig{},
		},
		{
			"happy_path",
			&ServicesConditionConfig{
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

func TestServicesConditionConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ServicesConditionConfig
		b    *ServicesConditionConfig
		r    *ServicesConditionConfig
	}{
		{
			"nil_a",
			nil,
			&ServicesConditionConfig{},
			&ServicesConditionConfig{},
		},
		{
			"nil_b",
			&ServicesConditionConfig{},
			nil,
			&ServicesConditionConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&ServicesConditionConfig{},
			&ServicesConditionConfig{},
			&ServicesConditionConfig{},
		},
		{
			"happy_path",
			&ServicesConditionConfig{
				ServicesMonitorConfig{
					Regexp:             String("regexp"),
					Datacenter:         String("datacenter_overriden"),
					Namespace:          nil,
					Filter:             nil,
					CTSUserDefinedMeta: map[string]string{},
				},
			},
			&ServicesConditionConfig{
				ServicesMonitorConfig{
					Regexp:             nil,
					Datacenter:         String("datacenter"),
					Namespace:          String("namespace"),
					Filter:             nil,
					CTSUserDefinedMeta: map[string]string{},
				},
			},
			&ServicesConditionConfig{
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

func TestServicesConditionConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		s    []string
		i    *ServicesConditionConfig
		r    *ServicesConditionConfig
	}{
		{
			"nil",
			[]string{},
			nil,
			nil,
		},
		{
			"happy_path",
			[]string{},
			&ServicesConditionConfig{},
			&ServicesConditionConfig{
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
			tc.i.Finalize(tc.s)
			assert.Equal(t, tc.r, tc.i)
		})
	}
}

func TestServicesConditionConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *ServicesConditionConfig
	}{
		{
			"valid",
			false,
			&ServicesConditionConfig{
				ServicesMonitorConfig{
					Regexp: String(".*"),
				},
			},
		},
		{
			"invalid",
			true,
			&ServicesConditionConfig{
				ServicesMonitorConfig{
					Regexp: String("*"),
				},
			},
		},
		{
			"nil",
			false,
			nil,
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

func TestServicesCondition_GoString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		i        *ServicesConditionConfig
		expected string
	}{
		{
			"nil",
			nil,
			"(*ServicesConditionConfig)(nil)",
		},
		{
			"fully_configured",
			&ServicesConditionConfig{
				ServicesMonitorConfig{
					Regexp:     String("^api$"),
					Datacenter: String("dc"),
					Namespace:  String("namespace"),
					Filter:     String("filter"),
					CTSUserDefinedMeta: map[string]string{
						"key": "value",
					},
				},
			},
			"&ServicesConditionConfig{&ServicesMonitorConfig{Regexp:^api$, Names:[], " +
				"Datacenter:dc, Namespace:namespace, Filter:filter, " +
				"CTSUserDefinedMeta:map[key:value]}}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.i.GoString()
			assert.Equal(t, tc.expected, actual)
		})
	}
}
