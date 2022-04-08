package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderConfigs_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &TerraformProviderConfigs{}
	finalizedConf.Finalize()

	cases := []struct {
		name string
		a    *TerraformProviderConfigs
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&TerraformProviderConfigs{},
		},
		{
			"finalized",
			finalizedConf,
		},
		{
			"same_enabled",
			&TerraformProviderConfigs{
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

func TestTerraformProviderConfigs_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *TerraformProviderConfigs
		b    *TerraformProviderConfigs
		r    *TerraformProviderConfigs
	}{
		{
			"nil_a",
			nil,
			&TerraformProviderConfigs{},
			&TerraformProviderConfigs{},
		},
		{
			"nil_b",
			&TerraformProviderConfigs{},
			nil,
			&TerraformProviderConfigs{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&TerraformProviderConfigs{},
			&TerraformProviderConfigs{},
			&TerraformProviderConfigs{},
		},
		{
			"appends",
			&TerraformProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}},
			&TerraformProviderConfigs{{
				"template": map[string]interface{}{
					"attr":  "t",
					"count": 5,
				},
			}},
			&TerraformProviderConfigs{{
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
			&TerraformProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}},
			&TerraformProviderConfigs{},
			&TerraformProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}},
		},
		{
			"empty_two",
			&TerraformProviderConfigs{},
			&TerraformProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}},
			&TerraformProviderConfigs{{
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
		i    *TerraformProviderConfigs
		r    *TerraformProviderConfigs
	}{
		{
			"empty",
			&TerraformProviderConfigs{},
			&TerraformProviderConfigs{},
		},
		{
			"with_name",
			&TerraformProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}},
			&TerraformProviderConfigs{{
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
		i       *TerraformProviderConfigs
		isValid bool
	}{
		{
			"nil",
			nil,
			false,
		},
		{
			"empty",
			&TerraformProviderConfigs{},
			true,
		},
		{
			"valid",
			&TerraformProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}},
			true,
		},
		{
			"empty provider map",
			&TerraformProviderConfigs{{}},
			false,
		}, {
			"multiple",
			&TerraformProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}, {
				"template": map[string]interface{}{
					"foo": "bar",
				},
			}},
			true,
		}, {
			"alias",
			&TerraformProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}, {
				"null": map[string]interface{}{
					"alias": "negative",
					"attr":  "abc",
					"count": -2,
				},
			}},
			true,
		}, {
			"duplicate",
			&TerraformProviderConfigs{{
				"null": map[string]interface{}{
					"attr":  "n",
					"count": 10,
				},
			}, {
				"null": map[string]interface{}{
					"attr":  "abc",
					"count": -2,
				},
			}},
			false,
		}, {
			"duplicate alias",
			&TerraformProviderConfigs{{
				"null": map[string]interface{}{
					"alias": "alias",
					"attr":  "n",
					"count": 10,
				},
			}, {
				"null": map[string]interface{}{
					"alias": "alias",
					"attr":  "abc",
					"count": -2,
				},
			}},
			false,
		}, {
			"task_env",
			&TerraformProviderConfigs{{
				"null": map[string]interface{}{
					"task_env": map[string]interface{}{
						"NULL_TOKEN": "{{ env \"MY_CTS_NULL_TOKEN\" }}",
						"NULL_BOOL":  "true",
						"NULL_NUM":   "10",
					},
				},
			}},
			true,
		}, {
			"task_env invalid",
			&TerraformProviderConfigs{{
				"null": map[string]interface{}{
					"task_env": map[string]interface{}{
						"NULL_BOOL": true,
					},
				},
			}},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.i.Validate()
			if tc.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestProviderConfigs_GoString(t *testing.T) {
	t.Parallel()

	// only testing provider cases with one argument since map order is random
	cases := []struct {
		name     string
		i        *TerraformProviderConfigs
		expected string
	}{
		{
			"nil",
			nil,
			`(*TerraformProviderConfigs)(nil)`,
		},
		{
			"empty",
			&TerraformProviderConfigs{},
			`{}`,
		},
		{
			"single config",
			&TerraformProviderConfigs{{
				"null": map[string]interface{}{
					"count": "10",
				},
			}},
			fmt.Sprintf(`{&map[null:%s]}`, redactMessage),
		},
		{
			"multiple configs, same provider",
			&TerraformProviderConfigs{{
				"null": map[string]interface{}{
					"attr": "n",
				},
			}, {
				"null": map[string]interface{}{
					"alias": "negative",
				},
			}},
			fmt.Sprintf(`{&map[null:%s], &map[null:%s]}`,
				redactMessage, redactMessage),
		},
		{
			"multiple configs, different provider",
			&TerraformProviderConfigs{{
				"firewall": map[string]interface{}{
					"hostname": "127.0.0.10",
					"username": "username",
					"password": "password123",
				},
			}, {
				"loadbalancer": map[string]interface{}{
					"address": "10.10.10.10",
					"api_key": "abcd123",
				},
			}},
			fmt.Sprintf(`{&map[firewall:%s], &map[loadbalancer:%s]}`,
				redactMessage, redactMessage),
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			actual := tc.i.GoString()
			assert.Equal(t, tc.expected, actual)
		})
	}
}
