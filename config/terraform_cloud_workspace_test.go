// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerraformCloudWorkspaceConfig_Copy(t *testing.T) {
	t.Parallel()

	finalizedConf := &TerraformCloudWorkspaceConfig{}
	finalizedConf.Finalize()

	cases := []struct {
		name string
		c    *TerraformCloudWorkspaceConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&TerraformCloudWorkspaceConfig{},
		},
		{
			"finalized",
			finalizedConf,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.c.Copy()
			assert.Equal(t, tc.c, c)
		})
	}

	t.Run("fully_configured", func(t *testing.T) {
		o := &TerraformCloudWorkspaceConfig{
			ExecutionMode:    String("test-mode"),
			AgentPoolID:      String("test-id"),
			AgentPoolName:    String("test-name"),
			TerraformVersion: String("test-version"),
		}
		c := o.Copy()
		assert.Equal(t, o, c)
		assert.NotSame(t, o.ExecutionMode, c.ExecutionMode)
		assert.NotSame(t, o.AgentPoolID, c.AgentPoolID)
		assert.NotSame(t, o.AgentPoolName, c.AgentPoolName)
		assert.NotSame(t, o.TerraformVersion, c.TerraformVersion)
	})
}

func TestTerraformCloudWorkspaceConfig_Merge(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		a    *TerraformCloudWorkspaceConfig
		b    *TerraformCloudWorkspaceConfig
		r    *TerraformCloudWorkspaceConfig
	}{
		{
			"nil_a",
			nil,
			&TerraformCloudWorkspaceConfig{},
			&TerraformCloudWorkspaceConfig{},
		},
		{
			"nil_b",
			&TerraformCloudWorkspaceConfig{},
			nil,
			&TerraformCloudWorkspaceConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&TerraformCloudWorkspaceConfig{},
			&TerraformCloudWorkspaceConfig{},
			&TerraformCloudWorkspaceConfig{},
		},
		{
			"execution_mode_overrides",
			&TerraformCloudWorkspaceConfig{ExecutionMode: String("remote")},
			&TerraformCloudWorkspaceConfig{ExecutionMode: String("agent")},
			&TerraformCloudWorkspaceConfig{ExecutionMode: String("agent")},
		},
		{
			"execution_mode_empty_a",
			&TerraformCloudWorkspaceConfig{},
			&TerraformCloudWorkspaceConfig{ExecutionMode: String("agent")},
			&TerraformCloudWorkspaceConfig{ExecutionMode: String("agent")},
		},
		{
			"execution_mode_empty_b",
			&TerraformCloudWorkspaceConfig{ExecutionMode: String("agent")},
			&TerraformCloudWorkspaceConfig{},
			&TerraformCloudWorkspaceConfig{ExecutionMode: String("agent")},
		},
		{
			"agent_pool_id_overrides",
			&TerraformCloudWorkspaceConfig{AgentPoolID: String("apool-1")},
			&TerraformCloudWorkspaceConfig{AgentPoolID: String("apool-2")},
			&TerraformCloudWorkspaceConfig{AgentPoolID: String("apool-2")},
		},
		{
			"agent_pool_id_empty_a",
			&TerraformCloudWorkspaceConfig{},
			&TerraformCloudWorkspaceConfig{AgentPoolID: String("apool-2")},
			&TerraformCloudWorkspaceConfig{AgentPoolID: String("apool-2")},
		},
		{
			"agent_pool_id_empty_b",
			&TerraformCloudWorkspaceConfig{AgentPoolID: String("apool-1")},
			&TerraformCloudWorkspaceConfig{},
			&TerraformCloudWorkspaceConfig{AgentPoolID: String("apool-1")},
		},
		{
			"agent_pool_name_overrides",
			&TerraformCloudWorkspaceConfig{AgentPoolName: String("agent_pool_a")},
			&TerraformCloudWorkspaceConfig{AgentPoolName: String("agent_pool_b")},
			&TerraformCloudWorkspaceConfig{AgentPoolName: String("agent_pool_b")},
		},
		{
			"agent_pool_name_empty_a",
			&TerraformCloudWorkspaceConfig{},
			&TerraformCloudWorkspaceConfig{AgentPoolName: String("agent_pool_b")},
			&TerraformCloudWorkspaceConfig{AgentPoolName: String("agent_pool_b")},
		},
		{
			"agent_pool_name_empty_b",
			&TerraformCloudWorkspaceConfig{AgentPoolName: String("agent_pool_a")},
			&TerraformCloudWorkspaceConfig{},
			&TerraformCloudWorkspaceConfig{AgentPoolName: String("agent_pool_a")},
		},
		{
			"terraform_version_overrides",
			&TerraformCloudWorkspaceConfig{TerraformVersion: String("1.0.0")},
			&TerraformCloudWorkspaceConfig{TerraformVersion: String("1.1.1")},
			&TerraformCloudWorkspaceConfig{TerraformVersion: String("1.1.1")},
		},
		{
			"terraform_version_empty_a",
			&TerraformCloudWorkspaceConfig{},
			&TerraformCloudWorkspaceConfig{TerraformVersion: String("1.1.1")},
			&TerraformCloudWorkspaceConfig{TerraformVersion: String("1.1.1")},
		},
		{
			"terraform_version_empty_b",
			&TerraformCloudWorkspaceConfig{TerraformVersion: String("1.0.0")},
			&TerraformCloudWorkspaceConfig{},
			&TerraformCloudWorkspaceConfig{TerraformVersion: String("1.0.0")},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			assert.Equal(t, tc.r, r)
		})
	}
}

func TestTerraformCloudWorkspaceConfig_Finalize(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		c    *TerraformCloudWorkspaceConfig
		r    *TerraformCloudWorkspaceConfig
	}{
		{
			"nil",
			nil,
			nil,
		},
		{
			"empty",
			&TerraformCloudWorkspaceConfig{},
			&TerraformCloudWorkspaceConfig{
				ExecutionMode:    String(""),
				AgentPoolID:      String(""),
				AgentPoolName:    String(""),
				TerraformVersion: String(""),
			},
		},
		{
			"fully_configured",
			&TerraformCloudWorkspaceConfig{
				ExecutionMode:    String("test_mode"),
				AgentPoolID:      String("apool-1"),
				AgentPoolName:    String("test"),
				TerraformVersion: String("1.1.1"),
			},
			&TerraformCloudWorkspaceConfig{
				ExecutionMode:    String("test_mode"),
				AgentPoolID:      String("apool-1"),
				AgentPoolName:    String("test"),
				TerraformVersion: String("1.1.1"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.c.Finalize()
			assert.Equal(t, tc.r, tc.c)
		})
	}
}

func TestTerraformCloudWorkspaceConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expectErr bool
		c         *TerraformCloudWorkspaceConfig
	}{
		{
			"nil",
			false,
			nil,
		},
		{
			"valid_remote",
			false,
			&TerraformCloudWorkspaceConfig{
				ExecutionMode: String("remote"),
			},
		},
		{
			"valid_agent_pool_id",
			false,
			&TerraformCloudWorkspaceConfig{
				ExecutionMode: String("agent"),
				AgentPoolID:   String("apool-1"),
			},
		},
		{
			"valid_agent_pool_name",
			false,
			&TerraformCloudWorkspaceConfig{
				ExecutionMode: String("agent"),
				AgentPoolName: String("test_agent_pool"),
			},
		},
		{
			"valid_both_agent_pool_id_and_name",
			false, // warns but does not error
			&TerraformCloudWorkspaceConfig{
				ExecutionMode: String("agent"),
				AgentPoolID:   String("apool-1"),
				AgentPoolName: String("test_agent_pool"),
			},
		},
		{
			"empty",
			false, // shouldn't error because optional
			&TerraformCloudWorkspaceConfig{},
		},
		{
			"invalid_execution_mode",
			true,
			&TerraformCloudWorkspaceConfig{
				ExecutionMode: String("local"),
			},
		},
		{
			"agent_with_no_agent_pool",
			true,
			&TerraformCloudWorkspaceConfig{
				ExecutionMode: String("agent"),
			},
		},
		{
			"remote_with_agent_pool_id",
			true,
			&TerraformCloudWorkspaceConfig{
				ExecutionMode: String("remote"),
				AgentPoolID:   String("apool-1"),
			},
		},
		{
			"remote_with_agent_pool_name",
			true,
			&TerraformCloudWorkspaceConfig{
				ExecutionMode: String("remote"),
				AgentPoolName: String("test_agent_pool"),
			},
		},
		{
			"valid_terraform_version",
			false,
			&TerraformCloudWorkspaceConfig{
				TerraformVersion: String("0.15.1"),
			},
		},
		{
			"invalid_terraform_version",
			true,
			&TerraformCloudWorkspaceConfig{
				TerraformVersion: String("invalid"),
			},
		},
		{
			"incompatible_terraform_version",
			true,
			&TerraformCloudWorkspaceConfig{
				TerraformVersion: String("0.12.0"),
			},
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

func TestTerraformCloudWorkspaceConfig_GoString(t *testing.T) {
	cases := []struct {
		name     string
		c        *TerraformCloudWorkspaceConfig
		expected string
	}{
		{
			"nil",
			nil,
			"(*TerraformCloudWorkspaceConfig)(nil)",
		},
		{
			"partially_configured",
			&TerraformCloudWorkspaceConfig{
				ExecutionMode: String("remote"),
			},
			"&TerraformCloudWorkspaceConfig{" +
				"AgentPoolID:, " +
				"AgentPoolName:, " +
				"ExecutionMode:remote, " +
				"TerraformVersion:" +
				"}",
		},
		{
			"fully_configured",
			&TerraformCloudWorkspaceConfig{
				ExecutionMode:    String("agent"),
				AgentPoolID:      String("apool-1"),
				AgentPoolName:    String("test_agent_pool"),
				TerraformVersion: String("1.0.0"),
			},
			"&TerraformCloudWorkspaceConfig{" +
				"AgentPoolID:apool-1, " +
				"AgentPoolName:test_agent_pool, " +
				"ExecutionMode:agent, " +
				"TerraformVersion:1.0.0" +
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

func TestTerraformCloudWorkspaceConfig_DefaultTerraformCloudWorkspaceConfig(t *testing.T) {
	r := DefaultTerraformCloudWorkspaceConfig()
	expected := &TerraformCloudWorkspaceConfig{
		ExecutionMode:    String(""),
		AgentPoolID:      String(""),
		AgentPoolName:    String(""),
		TerraformVersion: String(""),
	}
	assert.Equal(t, expected, r)
}

func TestTerraformCloudWorkspaceConfig_isEmpty(t *testing.T) {
	cases := []struct {
		name     string
		c        *TerraformCloudWorkspaceConfig
		expected bool
	}{
		{
			"configured",
			&TerraformCloudWorkspaceConfig{
				ExecutionMode:    String("agent"),
				AgentPoolID:      String("apool-1"),
				AgentPoolName:    String("test_agent_pool"),
				TerraformVersion: String("1.0.0"),
			},
			false,
		},
		{
			"partially_configured",
			&TerraformCloudWorkspaceConfig{
				TerraformVersion: String("1.0.0"),
			},
			false,
		},
		{
			"empty_nil_values",
			&TerraformCloudWorkspaceConfig{},
			true,
		},
		{
			"empty_default_values",
			&TerraformCloudWorkspaceConfig{
				ExecutionMode:    String(""),
				AgentPoolID:      String(""),
				AgentPoolName:    String(""),
				TerraformVersion: String(""),
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.c.IsEmpty()
			assert.Equal(t, tc.expected, r)
		})
	}
}
