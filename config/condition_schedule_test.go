package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScheduleConditionConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &ScheduleConditionConfig{}
	finalizedConf.Finalize()

	cases := []struct {
		name string
		a    *ScheduleConditionConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&ScheduleConditionConfig{},
		},
		{
			"finalized",
			finalizedConf,
		},
		{
			"fully_configured",
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String("* * * * * * *"),
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

func TestScheduleConditionConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ScheduleConditionConfig
		b    *ScheduleConditionConfig
		r    *ScheduleConditionConfig
	}{
		{
			"nil_a",
			nil,
			&ScheduleConditionConfig{},
			&ScheduleConditionConfig{},
		},
		{
			"nil_b",
			&ScheduleConditionConfig{},
			nil,
			&ScheduleConditionConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&ScheduleConditionConfig{},
			&ScheduleConditionConfig{},
			&ScheduleConditionConfig{},
		},
		{
			"cron_overrides",
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String("same"),
				},
			},
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String("different"),
				},
			},
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String("different"),
				},
			},
		},
		{
			"cron_empty_one",
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String("same"),
				},
			},
			&ScheduleConditionConfig{},
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String("same"),
				},
			},
		},
		{
			"cron_empty_two",
			&ScheduleConditionConfig{},
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String("same"),
				},
			},
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String("same"),
				},
			},
		},
		{
			"cron_empty_same",
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String("same"),
				},
			},
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String("same"),
				},
			},
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String("same"),
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

func TestScheduleConditionConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *ScheduleConditionConfig
		r    *ScheduleConditionConfig
	}{
		{
			"nil",
			nil,
			nil,
		},
		{
			"empty",
			&ScheduleConditionConfig{},
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String(""),
				},
			},
		},
		{
			"cron_configured",
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String("* * * * *"),
				},
			},
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String("* * * * *"),
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

func TestScheduleConditionConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *ScheduleConditionConfig
	}{
		{
			"nil",
			false,
			nil,
		},
		{
			"valid_cron",
			false,
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String("* * * * * * *"),
				},
			},
		},
		{
			"nil_cron",
			true,
			&ScheduleConditionConfig{},
		},
		{
			"invalid_cron",
			true,
			&ScheduleConditionConfig{
				ScheduleMonitorConfig: ScheduleMonitorConfig{
					Cron: String("invalid"),
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
