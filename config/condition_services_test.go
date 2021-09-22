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
			"fully_configured",
			&ServicesConditionConfig{
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
			"regexp_overrides",
			&ServicesConditionConfig{ServicesMonitorConfig{Regexp: String("same")}},
			&ServicesConditionConfig{ServicesMonitorConfig{Regexp: String("different")}},
			&ServicesConditionConfig{ServicesMonitorConfig{Regexp: String("different")}},
		},
		{
			"regexp_empty_one",
			&ServicesConditionConfig{ServicesMonitorConfig{Regexp: String("same")}},
			&ServicesConditionConfig{},
			&ServicesConditionConfig{ServicesMonitorConfig{Regexp: String("same")}},
		},
		{
			"regexp_empty_two",
			&ServicesConditionConfig{},
			&ServicesConditionConfig{ServicesMonitorConfig{Regexp: String("same")}},
			&ServicesConditionConfig{ServicesMonitorConfig{Regexp: String("same")}},
		},
		{
			"regexp_empty_same",
			&ServicesConditionConfig{ServicesMonitorConfig{Regexp: String("same")}},
			&ServicesConditionConfig{ServicesMonitorConfig{Regexp: String("same")}},
			&ServicesConditionConfig{ServicesMonitorConfig{Regexp: String("same")}},
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
			"empty",
			[]string{},
			&ServicesConditionConfig{},
			&ServicesConditionConfig{
				ServicesMonitorConfig{
					Regexp: String(""),
				},
			},
		},
		{
			"services_ignored",
			[]string{"api"},
			&ServicesConditionConfig{},
			&ServicesConditionConfig{
				ServicesMonitorConfig{
					Regexp: String(""),
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
			"happy_path",
			false,
			&ServicesConditionConfig{
				ServicesMonitorConfig{
					Regexp: String(".*"),
				},
			},
		},
		{
			"invalid_regexp",
			true,
			&ServicesConditionConfig{
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
