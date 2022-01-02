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
			"fully_configured",
			&ServicesSourceInputConfig{
				ServicesMonitorConfig{
					Regexp: String("^web.*"),
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
			"regexp_overrides",
			&ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("same")}},
			&ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("different")}},
			&ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("different")}},
		},
		{
			"regexp_empty_one",
			&ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("same")}},
			&ServicesSourceInputConfig{},
			&ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("same")}},
		},
		{
			"regexp_empty_two",
			&ServicesSourceInputConfig{},
			&ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("same")}},
			&ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("same")}},
		},
		{
			"regexp_empty_same",
			&ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("same")}},
			&ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("same")}},
			&ServicesSourceInputConfig{ServicesMonitorConfig{Regexp: String("same")}},
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

	var ssi *ServicesSourceInputConfig

	cases := []struct {
		name string
		s    []string
		i    *ServicesSourceInputConfig
		r    *ServicesSourceInputConfig
	}{
		{
			"empty",
			[]string{},
			&ServicesSourceInputConfig{},
			&ServicesSourceInputConfig{
				ServicesMonitorConfig{
					Regexp: String(""),
				},
			},
		},
		{
			"services_ignored",
			[]string{"api"},
			&ServicesSourceInputConfig{},
			&ServicesSourceInputConfig{
				ServicesMonitorConfig{
					Regexp: String(""),
				},
			},
		},
		{
			"services_nil",
			[]string{"api"},
			ssi,
			ssi,
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
			"happy_path",
			false,
			&ServicesSourceInputConfig{
				ServicesMonitorConfig{
					Regexp: String(".*"),
				},
			},
		},
		{
			"nil_happy_path",
			false,
			nil,
		},
		{
			"invalid_regexp",
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
					Regexp: String("^api$"),
				},
			},
			"&ServicesSourceInputConfig{" +
				"&ServicesMonitorConfig{" +
				"Regexp:^api$, " +
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
			actual := tc.ssv.String()
			require.Equal(t, tc.expected, actual)
		})
	}
}
