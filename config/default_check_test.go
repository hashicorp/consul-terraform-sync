// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCheckConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &DefaultCheckConfig{}
	finalizedConf.Finalize()

	cases := []struct {
		name string
		a    *DefaultCheckConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&DefaultCheckConfig{},
		},
		{
			"finalized",
			finalizedConf,
		},
		{
			"fully_configured",
			&DefaultCheckConfig{
				Enabled: Bool(false),
				Address: String("test"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.a.Copy()
			assert.Equal(t, tc.a, r)
		})
	}
}

func TestDefaultCheckConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *DefaultCheckConfig
		b    *DefaultCheckConfig
		r    *DefaultCheckConfig
	}{
		{
			"nil_a",
			nil,
			&DefaultCheckConfig{},
			&DefaultCheckConfig{},
		},
		{
			"nil_b",
			&DefaultCheckConfig{},
			nil,
			&DefaultCheckConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&DefaultCheckConfig{},
			&DefaultCheckConfig{},
			&DefaultCheckConfig{},
		},
		{
			"enabled_overrides",
			&DefaultCheckConfig{Enabled: Bool(false)},
			&DefaultCheckConfig{Enabled: Bool(true)},
			&DefaultCheckConfig{Enabled: Bool(true)},
		},
		{
			"enabled_empty_one",
			&DefaultCheckConfig{Enabled: Bool(false)},
			&DefaultCheckConfig{},
			&DefaultCheckConfig{Enabled: Bool(false)},
		},
		{
			"enabled_empty_two",
			&DefaultCheckConfig{},
			&DefaultCheckConfig{Enabled: Bool(false)},
			&DefaultCheckConfig{Enabled: Bool(false)},
		},
		{
			"enabled_same",
			&DefaultCheckConfig{Enabled: Bool(false)},
			&DefaultCheckConfig{Enabled: Bool(false)},
			&DefaultCheckConfig{Enabled: Bool(false)},
		},
		{
			"address_overrides",
			&DefaultCheckConfig{Address: String("127.0.0.1:123")},
			&DefaultCheckConfig{Address: String("test")},
			&DefaultCheckConfig{Address: String("test")},
		},
		{
			"address_empty_one",
			&DefaultCheckConfig{Address: String("127.0.0.1:123")},
			&DefaultCheckConfig{},
			&DefaultCheckConfig{Address: String("127.0.0.1:123")},
		},
		{
			"address_empty_two",
			&DefaultCheckConfig{},
			&DefaultCheckConfig{Address: String("127.0.0.1:123")},
			&DefaultCheckConfig{Address: String("127.0.0.1:123")},
		},
		{
			"address_same",
			&DefaultCheckConfig{Address: String("127.0.0.1:123")},
			&DefaultCheckConfig{Address: String("127.0.0.1:123")},
			&DefaultCheckConfig{Address: String("127.0.0.1:123")},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestDefaultCheckConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *DefaultCheckConfig
		r    *DefaultCheckConfig
	}{
		{
			"empty",
			&DefaultCheckConfig{},
			&DefaultCheckConfig{
				Enabled: Bool(true),
				Address: String(""),
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

func TestDefaultCheckConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		c         *DefaultCheckConfig
		expectErr bool
	}{
		{
			"nil",
			nil,
			false,
		},
		{
			"empty",
			&DefaultCheckConfig{},
			false,
		},
		{
			"defaults",
			&DefaultCheckConfig{
				Enabled: Bool(true),
				Address: String(""),
			},
			false,
		},
		{
			"configured",
			&DefaultCheckConfig{
				Enabled: Bool(true),
				Address: String("https://cts"),
			},
			false,
		},
		{
			"invalid_no_scheme",
			&DefaultCheckConfig{
				Enabled: Bool(true),
				Address: String("127.0.0.1"),
			},
			true,
		},
		{
			"invalid_format",
			&DefaultCheckConfig{
				Enabled: Bool(true),
				Address: String("http-not-a-uri"),
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.c.Validate()
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDefaultCheckConfig_GoString(t *testing.T) {
	cases := []struct {
		name     string
		c        *DefaultCheckConfig
		expected string
	}{
		{
			"nil",
			nil,
			"(*DefaultCheckConfig)(nil)",
		},
		{
			"fully_configured",
			&DefaultCheckConfig{
				Enabled: Bool(true),
				Address: String("test"),
			},
			"&DefaultCheckConfig{" +
				"Enabled:true, " +
				"Address:test" +
				"}",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.c.GoString()
			assert.Equal(t, tc.expected, r)
		})
	}
}
