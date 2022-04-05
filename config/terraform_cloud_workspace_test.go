package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerraformCloudWorkspaceConfig_Copy(t *testing.T) {
	t.Parallel()

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
			"fully_configured",
			&TerraformCloudWorkspaceConfig{
				ExecutionMode: String("agent"),
				AgentPoolID:   String("apool-123"),
				AgentPoolName: String("test_agent_pool"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.c.Copy()
			assert.Equal(t, tc.c, c)
		})
	}
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
				ExecutionMode: String(""),
				AgentPoolID:   String(""),
				AgentPoolName: String(""),
			},
		},
		{
			"fully_configured",
			&TerraformCloudWorkspaceConfig{
				ExecutionMode: String("test_mode"),
				AgentPoolID:   String("apool-1"),
				AgentPoolName: String("test")},
			&TerraformCloudWorkspaceConfig{
				ExecutionMode: String("test_mode"),
				AgentPoolID:   String("apool-1"),
				AgentPoolName: String("test"),
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
			"remote",
			&TerraformCloudWorkspaceConfig{
				ExecutionMode: String("remote"),
			},
			"&TerraformCloudWorkspaceConfig{" +
				"AgentPoolID:, " +
				"AgentPoolName:," +
				"ExecutionMode:remote" +
				"}",
		},
		{
			"fully_configured",
			&TerraformCloudWorkspaceConfig{
				ExecutionMode: String("agent"),
				AgentPoolID:   String("apool-1"),
				AgentPoolName: String("test_agent_pool"),
			},
			"&TerraformCloudWorkspaceConfig{" +
				"AgentPoolID:apool-1, " +
				"AgentPoolName:test_agent_pool," +
				"ExecutionMode:agent" +
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
		ExecutionMode: String(""),
		AgentPoolID:   String(""),
		AgentPoolName: String(""),
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
				ExecutionMode: String("agent"),
				AgentPoolID:   String("apool-1"),
				AgentPoolName: String("test_agent_pool"),
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
				ExecutionMode: String(""),
				AgentPoolID:   String(""),
				AgentPoolName: String(""),
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
