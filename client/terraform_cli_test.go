package client

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	mocks "github.com/hashicorp/consul-terraform-sync/mocks/client"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hashicorp/terraform-exec/tfexec"
)

func NewTestTerraformCLI(config *TerraformCLIConfig, tfMock *mocks.TerraformExec) *TerraformCLI {
	if tfMock == nil {
		tfvars := tfexec.VarFile("terraform.tfvars")
		ptfvars := tfexec.VarFile("providers.tfvars")

		m := new(mocks.TerraformExec)
		m.On("SetEnv", mock.Anything).Return(nil)
		m.On("Init", mock.Anything).Return(nil)
		m.On("Apply", mock.Anything, tfvars, ptfvars).Return(nil)
		m.On("Plan", mock.Anything, tfvars, ptfvars).Return(true, nil)
		m.On("WorkspaceNew", mock.Anything, mock.Anything).Return(nil)
		tfMock = m
	}

	client := &TerraformCLI{
		tf:         tfMock,
		workingDir: "test/working/dir",
		workspace:  "test-workspace",
	}

	if config == nil {
		return client
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
				ExecPath:   "path/to/tf",
				WorkingDir: "",
				Workspace:  "default",
			},
		},
		{
			"happy path",
			false,
			&TerraformCLIConfig{
				ExecPath:   "path/to/tf",
				WorkingDir: "./",
				Workspace:  "my-workspace",
			},
		},
		{
			"variable files",
			false,
			&TerraformCLIConfig{
				ExecPath:   "path/to/tf",
				WorkingDir: "./",
				Workspace:  "my-workspace",
				VarFiles:   []string{"variables.tf", "/path/to/variables.tf"},
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

			for _, vf := range actual.varFiles {
				assert.True(t, filepath.IsAbs(vf), "Expected absolute path for variable files")
			}
		})
	}
}

func TestTerraformCLIInit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		config      *TerraformCLIConfig
		initErr     error
		wsErr       error
	}{
		{
			"happy path",
			false,
			&TerraformCLIConfig{},
			nil,
			nil,
		},
		{
			"init err",
			true,
			&TerraformCLIConfig{},
			errors.New("init error"),
			nil,
		},
		{
			"workspace-new error: unknown error",
			true,
			&TerraformCLIConfig{},
			nil,
			errors.New("workspace-new error"),
		},
		{
			"workspace-new: already exists",
			false,
			&TerraformCLIConfig{},
			nil,
			&tfexec.ErrWorkspaceExists{Name: "workspace-name"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := new(mocks.TerraformExec)
			m.On("Init", mock.Anything).Return(tc.initErr).Once()
			m.On("WorkspaceNew", mock.Anything, mock.Anything).Return(tc.wsErr)
			m.On("WorkspaceSelect", mock.Anything, mock.Anything).Return(nil)

			client := NewTestTerraformCLI(tc.config, m)
			ctx := context.Background()
			err := client.Init(ctx)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			m.AssertExpectations(t)
		})
	}

	t.Run("workspace artifact with empty state", func(t *testing.T) {
		// Edge case to handle https://github.com/hashicorp/terraform/issues/21393
		initErr := errors.New(`Initializing the backend...

The currently selected workspace (test-workspace) does not exist.
This is expected behavior when the selected workspace did not have an
existing non-empty state. Please enter a number to select a workspace:

1. default

Enter a value:

Error: Failed to select workspace: input not a valid number`)

		m := new(mocks.TerraformExec)
		var initCount int
		m.On("Init", mock.Anything).Return(func(context.Context, ...tfexec.InitOption) error {
			initCount++
			if initCount == 1 {
				return initErr
			}
			return nil
		}).Twice()
		m.On("WorkspaceNew", mock.Anything, mock.Anything).Return(nil)
		m.On("WorkspaceSelect", mock.Anything, mock.Anything).Return(nil)

		client := NewTestTerraformCLI(&TerraformCLIConfig{}, m)
		ctx := context.Background()
		err := client.Init(ctx)
		assert.NoError(t, err)
		m.AssertExpectations(t)
	})
}

func TestTerraformCLIApply(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		config      *TerraformCLIConfig
	}{
		{
			"happy path",
			false,
			&TerraformCLIConfig{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewTestTerraformCLI(tc.config, nil)
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
	}{
		{
			"happy path",
			false,
			&TerraformCLIConfig{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewTestTerraformCLI(tc.config, nil)
			ctx := context.Background()
			_, err := client.Plan(ctx)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestTerraformCLIValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectedErr string
		jsonOut     string
	}{
		{
			"happy path",
			"",
			`{"valid": true}`,
		},
		{
			"missing services error",
			`
module for task "test-workspace" is missing the "services" variable
`,
			`{
	"format_version": "0.1",
	"valid": false,
	"error_count": 1,
	"warning_count": 0,
	"diagnostics": [
		{
			"severity": "error",
			"summary": "Unsupported argument",
			"detail": "An argument named \"services\" is not expected here.",
			"range": {
				"filename": "main.tf",
				"start": {
					"line": 31,
					"column": 3,
					"byte": 845
				},
				"end": {
					"line": 31,
					"column": 11,
					"byte": 853
				}
			},
			"snippet": {
				"context": "module \"example-task\"",
				"code": "  services = var.services",
				"start_line": 31,
				"highlight_start_offset": 2,
				"highlight_end_offset": 10,
				"values": []
			}
		}
	]
}`, // Terraform v0.15 output
		},
		{
			"missing catalog_services error",
			`
module for task "test-workspace" is missing the "catalog_services" variable, add to module or set "source_includes_var" to false
`,
			`{
	"valid": false,
	"error_count": 1,
	"warning_count": 0,
	"diagnostics": [
		{
			"severity": "error",
			"summary": "Unsupported argument",
			"detail": "An argument named \"catalog_services\" is not expected here.",
			"range": {
				"filename": "main.tf",
				"start": {
					"line": 32,
					"column": 3,
					"byte": 887
				},
				"end": {
					"line": 32,
					"column": 19,
					"byte": 903
				}
			}
		}
	]
}`, // Terraform v0.13/v0.14 output
		},
		{
			"warning",
			"",
			`{
	"valid": true,
	"diagnostics": [
		{
			"severity": "warning",
			"summary": "Version constraints inside provider configuration blocks are deprecated",
			"detail": "Terraform 0.13 and earlier allowed provider....",
			"range": {
				"filename": "main.tf",
				"start": {
					"line": 26
				}
			},
			"snippet": {
				"context": "provider \"local\"",
				"code": "  version = \"2.1.0\"",
				"start_line": 26
			}
		}
	]
}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := new(mocks.TerraformExec)
			var validateOut *tfjson.ValidateOutput
			json.Unmarshal([]byte(tc.jsonOut), &validateOut)
			m.On("Validate", mock.Anything).Return(validateOut, nil)
			client := NewTestTerraformCLI(&TerraformCLIConfig{}, m)

			ctx := context.Background()
			err := client.Validate(ctx)

			if tc.expectedErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedErr, err.Error())
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
			assert.Contains(t, tc.tf.GoString(), tc.tf.workingDir)
			assert.Contains(t, tc.tf.GoString(), tc.tf.workspace)
		})
	}
}
