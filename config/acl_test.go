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
				Enabled: Bool(true),
				Tokens: &TokensConfig{
					Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb"),
				},
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
			"token_overrides",
			&ACLConfig{Tokens: &TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")}},
			&ACLConfig{Tokens: &TokensConfig{Root: String("")}},
			&ACLConfig{Tokens: &TokensConfig{Root: String("")}},
		},
		{
			"token_empty_one",
			&ACLConfig{Tokens: &TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")}},
			&ACLConfig{},
			&ACLConfig{Tokens: &TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")}},
		},
		{
			"token_empty_two",
			&ACLConfig{Tokens: &TokensConfig{Root: String("")}},
			&ACLConfig{Tokens: &TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")}},
			&ACLConfig{Tokens: &TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")}},
		},
		{
			"token_same",
			&ACLConfig{Tokens: &TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")}},
			&ACLConfig{Tokens: &TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")}},
			&ACLConfig{Tokens: &TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")}},
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
				Enabled: Bool(defaultACLIsEnabled),
				Tokens:  &TokensConfig{Root: String("")},
			},
		},
		{
			"with_enabled",
			&ACLConfig{
				Enabled: Bool(true),
			},
			&ACLConfig{
				Enabled: Bool(true),
				Tokens:  &TokensConfig{Root: String("")},
			},
		},
		{
			"with_token",
			&ACLConfig{
				Tokens: &TokensConfig{
					Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb"),
				},
			},
			&ACLConfig{
				Enabled: Bool(defaultACLIsEnabled),
				Tokens: &TokensConfig{
					Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb"),
				},
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
				Enabled: Bool(defaultACLIsEnabled),
				Tokens: &TokensConfig{
					Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb"),
				},
			},
		},
		{
			"invalid_enabled",
			true,
			&ACLConfig{
				Tokens: &TokensConfig{
					Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb"),
				},
			},
		},
		{
			"invalid_token",
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
