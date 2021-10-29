package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestACLConfig_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ACLConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&ACLConfig{},
		},
		{
			"copy",
			&ACLConfig{
				Enabled:        Bool(true),
				BootstrapToken: String("+aHuWFH0bNzERaJpwdAPteD5EYzEQSWWNUxFsiVWt4ADIbHDU95ytJoYfHd/M22Q"),
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

func TestACLConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ACLConfig
		b    *ACLConfig
		r    *ACLConfig
	}{
		{
			"nil_a",
			nil,
			&ACLConfig{},
			&ACLConfig{},
		},
		{
			"nil_b",
			&ACLConfig{},
			nil,
			&ACLConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&ACLConfig{},
			&ACLConfig{},
			&ACLConfig{},
		},
		{
			"enabled_overrides",
			&ACLConfig{Enabled: Bool(true)},
			&ACLConfig{Enabled: Bool(false)},
			&ACLConfig{Enabled: Bool(false)},
		},
		{
			"enabled_empty_one",
			&ACLConfig{Enabled: Bool(true)},
			&ACLConfig{},
			&ACLConfig{Enabled: Bool(true)},
		},
		{
			"enabled_empty_two",
			&ACLConfig{},
			&ACLConfig{Enabled: Bool(true)},
			&ACLConfig{Enabled: Bool(true)},
		},
		{
			"enabled_same",
			&ACLConfig{Enabled: Bool(true)},
			&ACLConfig{Enabled: Bool(true)},
			&ACLConfig{Enabled: Bool(true)},
		},
		{
			"bootstrap_token_overrides",
			&ACLConfig{BootstrapToken: String("+aHuWFH0bNzERaJpwdAPteD5EYzEQSWWNUxFsiVWt4ADIbHDU95ytJoYfHd/M22Q")},
			&ACLConfig{BootstrapToken: String("")},
			&ACLConfig{BootstrapToken: String("")},
		},
		{
			"bootstrap_token_empty_one",
			&ACLConfig{BootstrapToken: String("+aHuWFH0bNzERaJpwdAPteD5EYzEQSWWNUxFsiVWt4ADIbHDU95ytJoYfHd/M22Q")},
			&ACLConfig{},
			&ACLConfig{BootstrapToken: String("+aHuWFH0bNzERaJpwdAPteD5EYzEQSWWNUxFsiVWt4ADIbHDU95ytJoYfHd/M22Q")},
		},
		{
			"bootstrap_token_empty_two",
			&ACLConfig{},
			&ACLConfig{BootstrapToken: String("+aHuWFH0bNzERaJpwdAPteD5EYzEQSWWNUxFsiVWt4ADIbHDU95ytJoYfHd/M22Q")},
			&ACLConfig{BootstrapToken: String("+aHuWFH0bNzERaJpwdAPteD5EYzEQSWWNUxFsiVWt4ADIbHDU95ytJoYfHd/M22Q")},
		},
		{
			"bootstrap_token_same",
			&ACLConfig{BootstrapToken: String("+aHuWFH0bNzERaJpwdAPteD5EYzEQSWWNUxFsiVWt4ADIbHDU95ytJoYfHd/M22Q")},
			&ACLConfig{BootstrapToken: String("+aHuWFH0bNzERaJpwdAPteD5EYzEQSWWNUxFsiVWt4ADIbHDU95ytJoYfHd/M22Q")},
			&ACLConfig{BootstrapToken: String("+aHuWFH0bNzERaJpwdAPteD5EYzEQSWWNUxFsiVWt4ADIbHDU95ytJoYfHd/M22Q")},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestACLConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *ACLConfig
		r    *ACLConfig
	}{
		{
			"empty",
			&ACLConfig{},
			&ACLConfig{
				Enabled:        Bool(defaultACLIsEnabled),
				BootstrapToken: String(""),
			},
		},
		{
			"with_enabled",
			&ACLConfig{
				Enabled: Bool(true),
			},
			&ACLConfig{
				Enabled:        Bool(true),
				BootstrapToken: String(""),
			},
		},
		{
			"with_bootstrap_token",
			&ACLConfig{
				BootstrapToken: String("+aHuWFH0bNzERaJpwdAPteD5EYzEQSWWNUxFsiVWt4ADIbHDU95ytJoYfHd/M22Q"),
			},
			&ACLConfig{
				Enabled:        Bool(defaultACLIsEnabled),
				BootstrapToken: String("+aHuWFH0bNzERaJpwdAPteD5EYzEQSWWNUxFsiVWt4ADIbHDU95ytJoYfHd/M22Q"),
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

func TestACLConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *ACLConfig
	}{
		{
			"happy_path",
			false,
			&ACLConfig{
				Enabled:        Bool(defaultACLIsEnabled),
				BootstrapToken: String("+aHuWFH0bNzERaJpwdAPteD5EYzEQSWWNUxFsiVWt4ADIbHDU95ytJoYfHd/M22Q"),
			},
		},
		{
			"invalid_enabled",
			true,
			&ACLConfig{
				BootstrapToken: String("+aHuWFH0bNzERaJpwdAPteD5EYzEQSWWNUxFsiVWt4ADIbHDU95ytJoYfHd/M22Q"),
			},
		},
		{
			"invalid_bootstrap_token",
			true,
			&ACLConfig{
				Enabled: Bool(defaultACLIsEnabled),
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
