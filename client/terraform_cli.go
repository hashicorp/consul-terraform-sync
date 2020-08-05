package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-exec/tfexec"
)

// workspaceEnv is the environment variable to set a Terraform workspace
const workspaceEnv = "TF_WORKSPACE"

var _ Client = (*TerraformCli)(nil)

// TerraformCli is the client that wraps around terraform-exec
// to execute Terraform cli commands
type TerraformCli struct {
	tf         terraformExec
	logLevel   string
	workingDir string
	workspace  string
}

// TerraformCliConfig configures the Terraform client
type TerraformCliConfig struct {
	LogLevel   string
	ExecPath   string
	WorkingDir string
	Workspace  string
}

// NewTerraformCli creates a terraform-exec client and configures and
// initializes a new Terraform client
func NewTerraformCli(config *TerraformCliConfig) (*TerraformCli, error) {
	if config == nil {
		return nil, errors.New("TerraformCliConfig cannot be nil - no meaningful default values")
	}

	tf, err := tfexec.NewTerraform(config.WorkingDir, config.ExecPath)
	if err != nil {
		return nil, err
	}
	if config.Workspace != "" {
		env := make(map[string]string)
		env[workspaceEnv] = config.Workspace
		tf.SetEnv(env)
	}

	return &TerraformCli{
		tf:         tf,
		logLevel:   config.LogLevel,
		workingDir: config.WorkingDir,
		workspace:  config.Workspace,
	}, nil
}

// Init executes the cli command a `terraform init`
func (t *TerraformCli) Init(ctx context.Context) error {
	return t.tf.Init(ctx)
}

// Apply executes the cli command `terraform apply` for a given workspace
func (t *TerraformCli) Apply(ctx context.Context) error {
	return t.tf.Apply(ctx)
}

// Plan executes the cli command a `terraform plan` for a given workspace
func (t *TerraformCli) Plan(ctx context.Context) error {
	return t.tf.Plan(ctx)
}

// GoString defines the printable version of this struct.
func (t *TerraformCli) GoString() string {
	if t == nil {
		return "(*TerraformCli)(nil)"
	}

	return fmt.Sprintf("&TerraformCli{"+
		"LogLevel:%s, "+
		"WorkingDir:%s, "+
		"WorkSpace:%s, "+
		"}",
		t.logLevel,
		t.workingDir,
		t.workspace,
	)
}
