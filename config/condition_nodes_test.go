package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodesConditionConfig_Copy(t *testing.T) {
	cases := []struct {
		name string
		a    *NodesConditionConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&NodesConditionConfig{},
		},
		{
			"fully_configured",
			&NodesConditionConfig{
				Datacenter: String("dc2"),
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

func TestNodesConditionConfig_Merge(t *testing.T) {
	cases := []struct {
		name string
		a    *NodesConditionConfig
		b    *NodesConditionConfig
		r    *NodesConditionConfig
	}{
		{
			"nil_a",
			nil,
			&NodesConditionConfig{},
			&NodesConditionConfig{},
		},
		{
			"nil_b",
			&NodesConditionConfig{},
			nil,
			&NodesConditionConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&NodesConditionConfig{},
			&NodesConditionConfig{},
			&NodesConditionConfig{},
		},
		{
			"datacenter_overrides",
			&NodesConditionConfig{Datacenter: String("same")},
			&NodesConditionConfig{Datacenter: String("different")},
			&NodesConditionConfig{Datacenter: String("different")},
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

func TestNodesConditionConfig_Finalize(t *testing.T) {
	cases := []struct {
		name string
		s    []string
		i    *NodesConditionConfig
		r    *NodesConditionConfig
	}{
		{
			"empty",
			[]string{},
			&NodesConditionConfig{},
			&NodesConditionConfig{
				Datacenter: String(""),
				services:   []string{},
			},
		},
		{
			"pass_in_services",
			[]string{"api"},
			&NodesConditionConfig{},
			&NodesConditionConfig{
				Datacenter: String(""),
				services:   []string{"api"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.i.Finalize(tc.s)
			assert.Equal(t, tc.r, tc.i)
		})
	}
}

func TestNodesConditionConfig_Validate(t *testing.T) {
	cases := []struct {
		name      string
		expectErr bool
		services  []string
	}{
		{
			"nil",
			false,
			nil,
		},
		{
			"empty",
			false,
			[]string{},
		},
		{
			"services set",
			true,
			[]string{"api"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := &NodesConditionConfig{}
			n.Finalize(tc.services)
			err := n.Validate()
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
