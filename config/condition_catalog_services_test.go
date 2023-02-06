// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCatalogServicesConditionConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &CatalogServicesConditionConfig{}
	finalizedConf.Finalize()

	cases := []struct {
		name string
		a    *CatalogServicesConditionConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&CatalogServicesConditionConfig{},
		},
		{
			"finalized",
			finalizedConf,
		},
		{
			"happy_path",
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{
					Regexp:           String(".*"),
					UseAsModuleInput: Bool(true),
					Datacenter:       String("dc2"),
					Namespace:        String("ns2"),
					NodeMeta: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
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

func TestCatalogServicesConditionConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *CatalogServicesConditionConfig
		b    *CatalogServicesConditionConfig
		r    *CatalogServicesConditionConfig
	}{
		{
			"nil_a",
			nil,
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{},
		},
		{
			"nil_b",
			&CatalogServicesConditionConfig{},
			nil,
			&CatalogServicesConditionConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"happy_path",
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{
					Regexp:           String("regexp"),
					Datacenter:       String("datacenter_overriden"),
					Namespace:        nil,
					UseAsModuleInput: Bool(true),
					NodeMeta: map[string]string{
						"foo1": "bar1",
					},
				},
			},
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{
					Regexp:           nil,
					Datacenter:       String("datacenter"),
					Namespace:        String("namespace"),
					UseAsModuleInput: Bool(false),
					NodeMeta: map[string]string{
						"foo2": "bar2",
					},
				},
			},
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{
					Regexp:           String("regexp"),
					Datacenter:       String("datacenter"),
					Namespace:        String("namespace"),
					UseAsModuleInput: Bool(false),
					NodeMeta: map[string]string{
						"foo1": "bar1",
						"foo2": "bar2",
					},
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

func TestCatalogServicesConditionConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *CatalogServicesConditionConfig
		r    *CatalogServicesConditionConfig
	}{
		{
			"nil",
			nil,
			nil,
		},
		{
			"happy_path",
			&CatalogServicesConditionConfig{},
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{
					Regexp:           nil,
					UseAsModuleInput: Bool(true),
					Datacenter:       String(""),
					Namespace:        String(""),
					NodeMeta:         map[string]string{},
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

func TestCatalogServicesConditionConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *CatalogServicesConditionConfig
	}{
		{
			"nil",
			false,
			nil,
		},
		{
			"valid",
			false,
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{
					Regexp:           String(".*"),
					UseAsModuleInput: Bool(true),
					Datacenter:       String("dc2"),
					Namespace:        String("ns2"),
					NodeMeta: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
			},
		},
		{
			"invalid",
			true,
			&CatalogServicesConditionConfig{
				CatalogServicesMonitorConfig{},
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
