package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthConfig_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *AuthConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&AuthConfig{},
		},
		{
			"copy",
			&AuthConfig{
				Enabled:  Bool(true),
				Username: String("username"),
				Password: String("password"),
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

func TestAuthConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *AuthConfig
		b    *AuthConfig
		r    *AuthConfig
	}{
		{
			"nil_a",
			nil,
			&AuthConfig{},
			&AuthConfig{},
		},
		{
			"nil_b",
			&AuthConfig{},
			nil,
			&AuthConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&AuthConfig{},
			&AuthConfig{},
			&AuthConfig{},
		},
		{
			"enabled_overrides",
			&AuthConfig{Enabled: Bool(true)},
			&AuthConfig{Enabled: Bool(false)},
			&AuthConfig{Enabled: Bool(false)},
		},
		{
			"enabled_empty_one",
			&AuthConfig{Enabled: Bool(true)},
			&AuthConfig{},
			&AuthConfig{Enabled: Bool(true)},
		},
		{
			"enabled_empty_two",
			&AuthConfig{},
			&AuthConfig{Enabled: Bool(true)},
			&AuthConfig{Enabled: Bool(true)},
		},
		{
			"enabled_same",
			&AuthConfig{Enabled: Bool(true)},
			&AuthConfig{Enabled: Bool(true)},
			&AuthConfig{Enabled: Bool(true)},
		},
		{
			"username_overrides",
			&AuthConfig{Username: String("username")},
			&AuthConfig{Username: String("")},
			&AuthConfig{Username: String("")},
		},
		{
			"username_empty_one",
			&AuthConfig{Username: String("username")},
			&AuthConfig{},
			&AuthConfig{Username: String("username")},
		},
		{
			"username_empty_two",
			&AuthConfig{},
			&AuthConfig{Username: String("username")},
			&AuthConfig{Username: String("username")},
		},
		{
			"username_same",
			&AuthConfig{Username: String("username")},
			&AuthConfig{Username: String("username")},
			&AuthConfig{Username: String("username")},
		},
		{
			"password_overrides",
			&AuthConfig{Password: String("password")},
			&AuthConfig{Password: String("")},
			&AuthConfig{Password: String("")},
		},
		{
			"password_empty_one",
			&AuthConfig{Password: String("password")},
			&AuthConfig{},
			&AuthConfig{Password: String("password")},
		},
		{
			"password_empty_two",
			&AuthConfig{},
			&AuthConfig{Password: String("password")},
			&AuthConfig{Password: String("password")},
		},
		{
			"password_same",
			&AuthConfig{Password: String("password")},
			&AuthConfig{Password: String("password")},
			&AuthConfig{Password: String("password")},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestAuthConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *AuthConfig
		r    *AuthConfig
	}{
		{
			"empty",
			&AuthConfig{},
			&AuthConfig{
				Enabled:  Bool(false),
				Username: String(""),
				Password: String(""),
			},
		},
		{
			"with_username",
			&AuthConfig{
				Username: String("username"),
			},
			&AuthConfig{
				Enabled:  Bool(true),
				Username: String("username"),
				Password: String(""),
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
