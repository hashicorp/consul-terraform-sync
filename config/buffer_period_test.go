package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBufferPeriodConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &BufferPeriodConfig{}
	finalizedConf.Finalize(DefaultBufferPeriodConfig())

	cases := []struct {
		name string
		a    *BufferPeriodConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&BufferPeriodConfig{},
		},
		{
			"finalized",
			finalizedConf,
		},
		{
			"same_enabled",
			&BufferPeriodConfig{
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

func TestBufferPeriodConfig_Merge(t *testing.T) {
	cases := []struct {
		name string
		a    *BufferPeriodConfig
		b    *BufferPeriodConfig
		r    *BufferPeriodConfig
	}{
		{
			"nil_a",
			nil,
			&BufferPeriodConfig{},
			&BufferPeriodConfig{},
		},
		{
			"nil_b",
			&BufferPeriodConfig{},
			nil,
			&BufferPeriodConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&BufferPeriodConfig{},
			&BufferPeriodConfig{},
			&BufferPeriodConfig{},
		},
		{
			"enabled_overrides",
			&BufferPeriodConfig{Enabled: Bool(true)},
			&BufferPeriodConfig{Enabled: Bool(false)},
			&BufferPeriodConfig{Enabled: Bool(false)},
		},
		{
			"enabled_empty_one",
			&BufferPeriodConfig{Enabled: Bool(true)},
			&BufferPeriodConfig{},
			&BufferPeriodConfig{Enabled: Bool(true)},
		},
		{
			"enabled_empty_two",
			&BufferPeriodConfig{},
			&BufferPeriodConfig{Enabled: Bool(true)},
			&BufferPeriodConfig{Enabled: Bool(true)},
		},
		{
			"enabled_same",
			&BufferPeriodConfig{Enabled: Bool(true)},
			&BufferPeriodConfig{Enabled: Bool(true)},
			&BufferPeriodConfig{Enabled: Bool(true)},
		},
		{
			"min_overrides",
			&BufferPeriodConfig{Min: TimeDuration(10 * time.Second)},
			&BufferPeriodConfig{Min: TimeDuration(0 * time.Second)},
			&BufferPeriodConfig{Min: TimeDuration(0 * time.Second)},
		},
		{
			"min_empty_one",
			&BufferPeriodConfig{Min: TimeDuration(10 * time.Second)},
			&BufferPeriodConfig{},
			&BufferPeriodConfig{Min: TimeDuration(10 * time.Second)},
		},
		{
			"min_empty_two",
			&BufferPeriodConfig{},
			&BufferPeriodConfig{Min: TimeDuration(10 * time.Second)},
			&BufferPeriodConfig{Min: TimeDuration(10 * time.Second)},
		},
		{
			"min_same",
			&BufferPeriodConfig{Min: TimeDuration(10 * time.Second)},
			&BufferPeriodConfig{Min: TimeDuration(10 * time.Second)},
			&BufferPeriodConfig{Min: TimeDuration(10 * time.Second)},
		},
		{
			"max_overrides",
			&BufferPeriodConfig{Max: TimeDuration(20 * time.Second)},
			&BufferPeriodConfig{Max: TimeDuration(0 * time.Second)},
			&BufferPeriodConfig{Max: TimeDuration(0 * time.Second)},
		},
		{
			"max_empty_one",
			&BufferPeriodConfig{Max: TimeDuration(20 * time.Second)},
			&BufferPeriodConfig{},
			&BufferPeriodConfig{Max: TimeDuration(20 * time.Second)},
		},
		{
			"max_empty_two",
			&BufferPeriodConfig{},
			&BufferPeriodConfig{Max: TimeDuration(20 * time.Second)},
			&BufferPeriodConfig{Max: TimeDuration(20 * time.Second)},
		},
		{
			"max_same",
			&BufferPeriodConfig{Max: TimeDuration(20 * time.Second)},
			&BufferPeriodConfig{Max: TimeDuration(20 * time.Second)},
			&BufferPeriodConfig{Max: TimeDuration(20 * time.Second)},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestBufferPeriodConfig_Finalize(t *testing.T) {
	cases := []struct {
		name string
		i    *BufferPeriodConfig
		p    *BufferPeriodConfig
		r    *BufferPeriodConfig
	}{
		{
			"empty input, parent enabled",
			&BufferPeriodConfig{},
			DefaultBufferPeriodConfig(),
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(5 * time.Second),
				Max:     TimeDuration(20 * time.Second),
			},
		},
		{
			"empty input, parent disabled",
			&BufferPeriodConfig{},
			&BufferPeriodConfig{
				Enabled: Bool(false),
				Min:     TimeDuration(0 * time.Second),
				Max:     TimeDuration(0 * time.Second),
			},
			&BufferPeriodConfig{
				Enabled: Bool(false),
				Min:     TimeDuration(0 * time.Second),
				Max:     TimeDuration(0 * time.Second),
			},
		},
		{
			"disabled input",
			&BufferPeriodConfig{
				Enabled: Bool(false),
			},
			DefaultBufferPeriodConfig(),
			&BufferPeriodConfig{
				Enabled: Bool(false),
				Min:     TimeDuration(0 * time.Second),
				Max:     TimeDuration(0 * time.Second),
			},
		},
		{
			"enabled input, enabled parent",
			&BufferPeriodConfig{
				Enabled: Bool(true),
			},
			DefaultBufferPeriodConfig(),
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(5 * time.Second),
				Max:     TimeDuration(20 * time.Second)},
		},
		{
			"enabled input, disabled parent",
			&BufferPeriodConfig{
				Enabled: Bool(true),
			},
			&BufferPeriodConfig{
				Enabled: Bool(false),
				Min:     TimeDuration(0 * time.Second),
				Max:     TimeDuration(0 * time.Second),
			},
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(5 * time.Second),
				Max:     TimeDuration(20 * time.Second),
			},
		},
		{
			"only min input",
			&BufferPeriodConfig{
				Min: TimeDuration(10 * time.Second),
			},
			DefaultBufferPeriodConfig(),
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(10 * time.Second),
				Max:     TimeDuration(40 * time.Second),
			},
		},
		{
			"only max input",
			&BufferPeriodConfig{
				Max: TimeDuration(50 * time.Second),
			},
			DefaultBufferPeriodConfig(),
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(5 * time.Second),
				Max:     TimeDuration(50 * time.Second),
			},
		},
		{
			"input enabled and parent enabled fully configured",
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(20 * time.Second),
				Max:     TimeDuration(50 * time.Second),
			},
			DefaultBufferPeriodConfig(),
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(20 * time.Second),
				Max:     TimeDuration(50 * time.Second),
			},
		},
		{
			"input enabled and parent disabled fully configured",
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(20 * time.Second),
				Max:     TimeDuration(50 * time.Second),
			},
			&BufferPeriodConfig{
				Enabled: Bool(false),
				Min:     TimeDuration(1 * time.Second),
				Max:     TimeDuration(5 * time.Second),
			},
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(20 * time.Second),
				Max:     TimeDuration(50 * time.Second),
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			tc.i.Finalize(tc.p)
			assert.Equal(t, tc.r, tc.i)
		})
	}
}

func TestBufferPeriodConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		i       *BufferPeriodConfig
		isValid bool
	}{
		{
			"nil",
			nil,
			true,
		},
		{
			"empty",
			&BufferPeriodConfig{},
			true,
		},
		{
			"valid",
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(time.Duration(10 * time.Second)),
				Max:     TimeDuration(time.Duration(60 * time.Second)),
			},
			true,
		},
		{
			"min negative",
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(time.Duration(-10 * time.Second)),
				Max:     TimeDuration(time.Duration(5 * time.Second)),
			},
			false,
		},
		{
			"max negative",
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(time.Duration(5 * time.Second)),
				Max:     TimeDuration(time.Duration(-10 * time.Second)),
			},
			false,
		},
		{
			"min > max",
			&BufferPeriodConfig{
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
