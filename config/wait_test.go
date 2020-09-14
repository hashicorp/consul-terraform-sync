package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWaitConfig_Copy(t *testing.T) {
	cases := []struct {
		name string
		a    *WaitConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&WaitConfig{},
		},
		{
			"same_enabled",
			&WaitConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(10 * time.Second),
				Max:     TimeDuration(20 * time.Second),
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Copy()
			assert.Equal(t, tc.a, r)
		})
	}
}

func TestWaitConfig_Merge(t *testing.T) {
	cases := []struct {
		name string
		a    *WaitConfig
		b    *WaitConfig
		r    *WaitConfig
	}{
		{
			"nil_a",
			nil,
			&WaitConfig{},
			&WaitConfig{},
		},
		{
			"nil_b",
			&WaitConfig{},
			nil,
			&WaitConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&WaitConfig{},
			&WaitConfig{},
			&WaitConfig{},
		},
		{
			"enabled_overrides",
			&WaitConfig{Enabled: Bool(true)},
			&WaitConfig{Enabled: Bool(false)},
			&WaitConfig{Enabled: Bool(false)},
		},
		{
			"enabled_empty_one",
			&WaitConfig{Enabled: Bool(true)},
			&WaitConfig{},
			&WaitConfig{Enabled: Bool(true)},
		},
		{
			"enabled_empty_two",
			&WaitConfig{},
			&WaitConfig{Enabled: Bool(true)},
			&WaitConfig{Enabled: Bool(true)},
		},
		{
			"enabled_same",
			&WaitConfig{Enabled: Bool(true)},
			&WaitConfig{Enabled: Bool(true)},
			&WaitConfig{Enabled: Bool(true)},
		},
		{
			"min_overrides",
			&WaitConfig{Min: TimeDuration(10 * time.Second)},
			&WaitConfig{Min: TimeDuration(0 * time.Second)},
			&WaitConfig{Min: TimeDuration(0 * time.Second)},
		},
		{
			"min_empty_one",
			&WaitConfig{Min: TimeDuration(10 * time.Second)},
			&WaitConfig{},
			&WaitConfig{Min: TimeDuration(10 * time.Second)},
		},
		{
			"min_empty_two",
			&WaitConfig{},
			&WaitConfig{Min: TimeDuration(10 * time.Second)},
			&WaitConfig{Min: TimeDuration(10 * time.Second)},
		},
		{
			"min_same",
			&WaitConfig{Min: TimeDuration(10 * time.Second)},
			&WaitConfig{Min: TimeDuration(10 * time.Second)},
			&WaitConfig{Min: TimeDuration(10 * time.Second)},
		},
		{
			"max_overrides",
			&WaitConfig{Max: TimeDuration(20 * time.Second)},
			&WaitConfig{Max: TimeDuration(0 * time.Second)},
			&WaitConfig{Max: TimeDuration(0 * time.Second)},
		},
		{
			"max_empty_one",
			&WaitConfig{Max: TimeDuration(20 * time.Second)},
			&WaitConfig{},
			&WaitConfig{Max: TimeDuration(20 * time.Second)},
		},
		{
			"max_empty_two",
			&WaitConfig{},
			&WaitConfig{Max: TimeDuration(20 * time.Second)},
			&WaitConfig{Max: TimeDuration(20 * time.Second)},
		},
		{
			"max_same",
			&WaitConfig{Max: TimeDuration(20 * time.Second)},
			&WaitConfig{Max: TimeDuration(20 * time.Second)},
			&WaitConfig{Max: TimeDuration(20 * time.Second)},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestWaitConfig_Finalize(t *testing.T) {
	cases := []struct {
		name string
		i    *WaitConfig
		r    *WaitConfig
	}{
		{
			"empty",
			&WaitConfig{},
			&WaitConfig{
				Enabled: Bool(false),
				Min:     TimeDuration(5 * time.Second),
				Max:     TimeDuration(20 * time.Second),
			},
		},
		{
			"with_min",
			&WaitConfig{
				Min: TimeDuration(10 * time.Second),
			},
			&WaitConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(10 * time.Second),
				Max:     TimeDuration(40 * time.Second),
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			tc.i.Finalize()
			assert.Equal(t, tc.r, tc.i)
		})
	}
}

func TestWaitConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		i       *WaitConfig
		isValid bool
	}{
		{
			"nil",
			nil,
			true,
		},
		{
			"empty",
			&WaitConfig{},
			true,
		},
		{
			"valid",
			&WaitConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(time.Duration(10 * time.Second)),
				Max:     TimeDuration(time.Duration(60 * time.Second)),
			},
			true,
		},
		{
			"min negative",
			&WaitConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(time.Duration(-10 * time.Second)),
				Max:     TimeDuration(time.Duration(5 * time.Second)),
			},
			false,
		},
		{
			"max negative",
			&WaitConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(time.Duration(5 * time.Second)),
				Max:     TimeDuration(time.Duration(-10 * time.Second)),
			},
			false,
		},
		{
			"min > max",
			&WaitConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(time.Duration(5 * time.Second)),
				Max:     TimeDuration(time.Duration(2 * time.Second)),
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			err := tc.i.Validate()
			if tc.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
