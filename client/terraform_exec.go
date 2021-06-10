package client

import (
	"context"
	"io"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-json"
)

//go:generate mockery --name=terraformExec  --structname=TerraformExec --output=../mocks/client

var _ terraformExec = (*tfexec.Terraform)(nil)

// terraformExec describes the interface for terraform-exec, the SDK for
// Terraform CLI: https://github.com/hashicorp/terraform-exec
type terraformExec interface {
	SetEnv(env map[string]string) error
	SetStdout(w io.Writer)
	Init(ctx context.Context, opts ...tfexec.InitOption) error
	Apply(ctx context.Context, opts ...tfexec.ApplyOption) error
	Plan(ctx context.Context, opts ...tfexec.PlanOption) (bool, error)
	WorkspaceNew(ctx context.Context, workspace string, opts ...tfexec.WorkspaceNewCmdOption) error
	WorkspaceSelect(ctx context.Context, workspace string) error
	Validate(ctx context.Context) (*tfjson.ValidateOutput, error)
}
