package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServicesSourceInputConfig_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ServicesSourceInputConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&ServicesSourceInputConfig{},
		},
		{
			"happy_path",
			&ServicesSourceInputConfig{
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

func TestServicesSourceInputConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ServicesSourceInputConfig
		b    *ServicesSourceInputConfig
		r    *ServicesSourceInputConfig
	}{
		{
			"nil_a",
			nil,
			&ServicesSourceInputConfig{},
			&ServicesSourceInputConfig{},
		},
		{
			"nil_b",
			&ServicesSourceInputConfig{},
			nil,
			&ServicesSourceInputConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&ServicesSourceInputConfig{},
			&ServicesSourceInputConfig{},
			&ServicesSourceInputConfig{},
		},
		{
			"happy_path",
			&ServicesSourceInputConfig{
				ServicesMonitorConfig{
					Regexp:             String("regexp"),
					Datacenter:         String("datacenter_overriden"),
					Namespace:          nil,
					Filter:             nil,
					CTSUserDefinedMeta: map[string]string{},
				},
			},
			&ServicesSourceInputConfig{
				ServicesMonitorConfig{
					Regexp:             nil,
					Datacenter:         String("datacenter"),
					Namespace:          String("namespace"),
					Filter:             nil,
					CTSUserDefinedMeta: map[string]string{},
				},
			},
			&ServicesSourceInputConfig{
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

func TestServicesSourceInputConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		s    []string
		i    *ServicesSourceInputConfig
		r    *ServicesSourceInputConfig
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
			&ServicesSourceInputConfig{},
			&ServicesSourceInputConfig{
				ServicesMonitorConfig{
					Regexp:             nil,
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

func TestServicesSourceInputConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *ServicesSourceInputConfig
	}{
		{
			"valid",
			false,
			&ServicesSourceInputConfig{
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
			&ServicesSourceInputConfig{
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
		ssv      *ServicesSourceInputConfig
		expected string
	}{
		{
			"configured services source_input",
			&ServicesSourceInputConfig{
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
			"&ServicesSourceInputConfig{" +
				"&ServicesMonitorConfig{" +
				"Regexp:^api$, " +
				"Datacenter:dc2, " +
				"Namespace:ns2, " +
				"Filter:some-filter, " +
				"CTSUserDefinedMeta:map[key:value]" +
				"}" +
				"}",
		},
		{
			"nil services source_input",
			nil,
			"(*ServicesSourceInputConfig)(nil)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.ssv.GoString()
			require.Equal(t, tc.expected, actual)
		})
	}
}
