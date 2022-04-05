package client

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/stretchr/testify/assert"
)

func Test_isConsulEnterprise(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		info             ConsulAgentConfig
		expectEnterprise bool
		expectError      bool
		error            error
	}{
		{
			name: "oss",
			info: ConsulAgentConfig{
				"Config": {"Version": "v1.9.5"},
			},
			expectEnterprise: false,
			expectError:      false,
		},
		{
			name: "oss dev",
			info: ConsulAgentConfig{
				"Config": {"Version": "v1.9.5-dev"},
			},
			expectEnterprise: false,
			expectError:      false,
		},
		{
			name: "ent",
			info: ConsulAgentConfig{
				"Config": {"Version": "v1.9.5+ent"},
			},
			expectEnterprise: true,
			expectError:      false,
		},
		{
			name: "ent dev",
			info: ConsulAgentConfig{
				"Config": {"Version": "v1.9.5+ent-dev"},
			},
			expectEnterprise: true,
			expectError:      false,
		},
		{
			name: "missing",
			info: ConsulAgentConfig{
				"Config": {},
			},
			expectEnterprise: false,
			expectError:      true,
			error:            errors.New("unable to parse map[Config][Version], keys do not exist"),
		},
		{
			name: "malformed",
			info: ConsulAgentConfig{
				"Config": {"Version": "***"},
			},
			expectEnterprise: false,
			expectError:      true,
		},
		{
			name: "bad key",
			info: ConsulAgentConfig{
				"NoConfig": {"Version": "***"},
			},
			expectEnterprise: false,
			expectError:      true,
			error:            errors.New("unable to parse map[Config][Version], keys do not exist"),
		},
		{
			name: "not string",
			info: ConsulAgentConfig{
				"Config": {"Version": []string{}},
			},
			expectEnterprise: false,
			expectError:      true,
			error:            errors.New("unable to parse map[Config][Version], keys do not map to string"),
		},
	}

	ctx := logging.WithContext(context.Background(), logging.NewNullLogger())

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isEnterprise, err := isConsulEnterprise(ctx, tc.info)
			if tc.expectError {
				assert.Error(t, err)
				if tc.error != nil {
					assert.Equal(t, tc.error, err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectEnterprise, isEnterprise)
			}
		})
	}
}
