package client

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/stretchr/testify/assert"
)

var _ terraformExec = (*mockTerraformExec)(nil)

type mockTerraformExec struct{}

func (m *mockTerraformExec) SetEnv(env map[string]string) error {
	return nil
}

func (m *mockTerraformExec) Init(ctx context.Context, opts ...tfexec.InitOption) error {
	return nil
}

func (m *mockTerraformExec) Apply(ctx context.Context, opts ...tfexec.ApplyOption) error {
	return nil
}

func (m *mockTerraformExec) Plan(ctx context.Context, opts ...tfexec.PlanOption) (bool, error) {
	return true, nil
}

func NewTestTerraformCLI(config *TerraformCLIConfig, mock *mockTerraformExec) *TerraformCLI {
	if mock == nil {
		mock = &mockTerraformExec{}
	}

	client := &TerraformCLI{
		tf:         mock,
		logLevel:   "INFO",
		workingDir: "test/working/dir",
		workspace:  "test-workspace",
	}

	if config == nil {
		return client
	}

	if config.LogLevel != "" {
		client.logLevel = config.LogLevel
	}
	if config.WorkingDir != "" {
		client.workingDir = config.WorkingDir
	}
	if config.Workspace != "" {
		client.workspace = config.Workspace
	}

	return client
}

func TestNewTerraformCLI(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		config      *TerraformCLIConfig
	}{
		{
			"error nil config",
			true,
			nil,
		},
		{
			"terraform-exec error: no working dir",
			true,
			&TerraformCLIConfig{
				LogLevel:   "INFO",
				ExecPath:   "path/to/tf",
				WorkingDir: "",
				Workspace:  "default",
			},
		},
		{
			"happy path",
			false,
			&TerraformCLIConfig{
				LogLevel:   "INFO",
				ExecPath:   "path/to/tf",
				WorkingDir: "./",
				Workspace:  "my-workspace",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := NewTerraformCLI(tc.config)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, actual)
		})
	}
}

func TestTerraformCLIInit(t *testing.T) {
	t.Skip("skipping this test until terraform-exec implements WorkspaceNew")
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		config      *TerraformCLIConfig
		tfMock      *mockTerraformExec
	}{
		{
			"happy path",
			false,
			&TerraformCLIConfig{},
			&mockTerraformExec{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewTestTerraformCLI(tc.config, tc.tfMock)
			ctx := context.Background()
			err := client.Init(ctx)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestTerraformCLIApply(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		config      *TerraformCLIConfig
		tfMock      *mockTerraformExec
	}{
		{
			"happy path",
			false,
			&TerraformCLIConfig{},
			&mockTerraformExec{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewTestTerraformCLI(tc.config, tc.tfMock)
			ctx := context.Background()
			err := client.Apply(ctx)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestTerraformCLIPlan(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		config      *TerraformCLIConfig
		tfMock      *mockTerraformExec
	}{
		{
			"happy path",
			false,
			&TerraformCLIConfig{},
			&mockTerraformExec{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewTestTerraformCLI(tc.config, tc.tfMock)
			ctx := context.Background()
			err := client.Plan(ctx)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestTerraformCLIGoString(t *testing.T) {
	cases := []struct {
		name string
		tf   *TerraformCLI
	}{
		{
			"nil Terraform cli client",
			nil,
		},
		{
			"happy path",
			&TerraformCLI{
				logLevel:   "INFO",
				workingDir: "path/to/wd",
				workspace:  "ws",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.tf == nil {
				assert.Contains(t, tc.tf.GoString(), "nil")
				return
			}

			assert.Contains(t, tc.tf.GoString(), "&TerraformCLI")
			assert.Contains(t, tc.tf.GoString(), tc.tf.logLevel)
			assert.Contains(t, tc.tf.GoString(), tc.tf.workingDir)
			assert.Contains(t, tc.tf.GoString(), tc.tf.workspace)
		})
	}
}
