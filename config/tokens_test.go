package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokensConfig_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *TokensConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&TokensConfig{},
		},
		{
			"copy",
			&TokensConfig{
				Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb"),
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

func TestTokensConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *TokensConfig
		b    *TokensConfig
		r    *TokensConfig
	}{
		{
			"nil_a",
			nil,
			&TokensConfig{},
			&TokensConfig{},
		},
		{
			"nil_b",
			&TokensConfig{},
			nil,
			&TokensConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&TokensConfig{},
			&TokensConfig{},
			&TokensConfig{},
		},
		{
			"root_token_overrides",
			&TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")},
			&TokensConfig{Root: String("")},
			&TokensConfig{Root: String("")},
		},
		{
			"root_token_empty_one",
			&TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")},
			&TokensConfig{},
			&TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")},
		},
		{
			"root_token_empty_two",
			&TokensConfig{},
			&TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")},
			&TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")},
		},
		{
			"root_token_same",
			&TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")},
			&TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")},
			&TokensConfig{Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb")},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestTokensConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *TokensConfig
		r    *TokensConfig
	}{
		{
			"empty",
			&TokensConfig{},
			&TokensConfig{Root: String("")},
		},
		{
			"with_root_token",
			&TokensConfig{
				Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb"),
			},
			&TokensConfig{
				Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb"),
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

func TestTokensConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *TokensConfig
	}{
		{
			"happy_path",
			false,
			&TokensConfig{
				Root: String("da666809-98ca-0e94-a99c-893c4bf5f9eb"),
			},
		},
		{
			"invalid_root_token",
			true,
			&TokensConfig{},
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
