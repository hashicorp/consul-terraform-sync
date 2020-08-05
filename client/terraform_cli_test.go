package client

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/stretchr/testify/assert"
)

var _ terraformExec = (*mockTerraformExec)(nil)

type mockTerraformExec struct{}

func (m *mockTerraformExec) Init(ctx context.Context, opts ...tfexec.InitOption) error {
	return nil
}

func (m *mockTerraformExec) Apply(ctx context.Context, opts ...tfexec.ApplyOption) error {
	return nil
}

func (m *mockTerraformExec) Plan(ctx context.Context, opts ...tfexec.PlanOption) error {
	return nil
}

func NewTestTerraformCli(config *TerraformCliConfig, mock *mockTerraformExec) *TerraformCli {
	if mock == nil {
		mock = &mockTerraformExec{}
	}

	client := &TerraformCli{
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

func TestNewTerraformCli(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		config      *TerraformCliConfig
	}{
		{
			"error nil config",
			true,
			nil,
		},
		{
			"terraform-exec error: no working dir",
			true,
			&TerraformCliConfig{
				LogLevel:   "INFO",
				ExecPath:   "path/to/tf",
				WorkingDir: "",
				Workspace:  "default",
			},
		},
		{
			"terraform-exec error: no tf binary in exec path",
			true,
			&TerraformCliConfig{
				LogLevel:   "INFO",
				ExecPath:   "path/to/tf",
				WorkingDir: "./",
				Workspace:  "my-workspace",
			},
		},
		// No happy path unit test possible because it requires the
		// terraform binary to exist. This would need to be tested
		// with an integration test
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := NewTerraformCli(tc.config)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, actual)
		})
	}
}

func TestTerraformCliInit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		config      *TerraformCliConfig
		tfMock      *mockTerraformExec
	}{
		{
			"happy path",
			false,
			&TerraformCliConfig{},
			&mockTerraformExec{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewTestTerraformCli(tc.config, tc.tfMock)
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

func TestTerraformCliApply(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		config      *TerraformCliConfig
		tfMock      *mockTerraformExec
	}{
		{
			"happy path",
			false,
			&TerraformCliConfig{},
			&mockTerraformExec{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewTestTerraformCli(tc.config, tc.tfMock)
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

func TestTerraformCliPlan(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		config      *TerraformCliConfig
		tfMock      *mockTerraformExec
	}{
		{
			"happy path",
			false,
			&TerraformCliConfig{},
			&mockTerraformExec{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewTestTerraformCli(tc.config, tc.tfMock)
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

func TestTerraformCliGoString(t *testing.T) {
	cases := []struct {
		name string
		tf   *TerraformCli
	}{
		{
			"nil Terraform cli client",
			nil,
		},
		{
			"happy path",
			&TerraformCli{
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

			assert.Contains(t, tc.tf.GoString(), "&TerraformCli")
			assert.Contains(t, tc.tf.GoString(), tc.tf.logLevel)
			assert.Contains(t, tc.tf.GoString(), tc.tf.workingDir)
			assert.Contains(t, tc.tf.GoString(), tc.tf.workspace)
		})
	}
}
