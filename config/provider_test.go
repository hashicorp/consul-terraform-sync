package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderConfigs_Copy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ProviderConfigs
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&ProviderConfigs{},
		},
		{
			"same_enabled",
			&ProviderConfigs{
				{
					"null": map[string]interface{}{
						"attr":  "value",
						"count": 10,
					},
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

func TestProviderConfigs_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *ProviderConfigs
		b    *ProviderConfigs
		r    *ProviderConfigs
	}{
		{
			"nil_a",
			nil,
			&ProviderConfigs{},
			&ProviderConfigs{},
		},
		{
			"nil_b",
			&ProviderConfigs{},
			nil,
			&ProviderConfigs{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&ProviderConfigs{},
			&ProviderConfigs{},
			&ProviderConfigs{},
		},
		{
			"appends",
			&ProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}},
			&ProviderConfigs{{
				"template": map[string]interface{}{
					"attr":  "t",
					"count": 5,
				},
			}},
			&ProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}, {
				"template": map[string]interface{}{
					"attr":  "t",
					"count": 5,
				},
			}},
		},
		{
			"empty_one",
			&ProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}},
			&ProviderConfigs{},
			&ProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}},
		},
		{
			"empty_two",
			&ProviderConfigs{},
			&ProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}},
			&ProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestProviderConfigs_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *ProviderConfigs
		r    *ProviderConfigs
	}{
		{
			"empty",
			&ProviderConfigs{},
			&ProviderConfigs{},
		},
		{
			"with_name",
			&ProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}},
			&ProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			tc.i.Finalize()
			assert.Equal(t, tc.r, tc.i)
		})
	}
}

func TestProviderConfigs_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		i       *ProviderConfigs
		isValid bool
	}{
		{
			"nil",
			nil,
			false,
		},
		{
			"empty",
			&ProviderConfigs{},
			true,
		},
		{
			"valid",
			&ProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}},
			true,
		},
		{
			"empty provider map",
			&ProviderConfigs{{}},
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
