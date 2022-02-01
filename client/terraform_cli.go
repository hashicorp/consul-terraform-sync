package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/terraform-exec/tfexec"
)

var (
	_ Client = (*TerraformCLI)(nil)

	wsFailedToSelectRegexp = regexp.MustCompile(`Failed to select workspace`)
)

const (
	tcliSubsystemName = "terraformcli"
)

// TerraformCLI is the client that wraps around terraform-exec
// to execute Terraform cli commands
type TerraformCLI struct {
	tf         terraformExec
	workingDir string
	workspace  string
	logger     logging.Logger
}

// TerraformCLIConfig configures the Terraform client
type TerraformCLIConfig struct {
	Log        bool
	PersistLog bool
	ExecPath   string
	WorkingDir string
	Workspace  string
}

// NewTerraformCLI creates a terraform-exec client and configures and
// initializes a new Terraform client
func NewTerraformCLI(config *TerraformCLIConfig) (*TerraformCLI, error) {
	if config == nil {
		return nil, errors.New("TerraformCLIConfig cannot be nil - no meaningful default values")
	}

	tfPath := filepath.Join(config.ExecPath, "terraform")
	tf, err := tfexec.NewTerraform(config.WorkingDir, tfPath)
	if err != nil {
		return nil, err
	}

	// tfexec does not support logging levels. This enables Terraform output to
	// log within Sync logs. This is useful for debugging and development
	// purposes. It may be difficult to work with log aggregators that expect
	// uniform log format.
	logger := logging.Global().Named(loggingSystemName).Named(tcliSubsystemName)
	if config.Log {
		logger.Info("Terraform logging is set, Terraform logs will output with Sync logs")
		lg := log.New(log.Writer(), "", log.Flags())
		tf.SetLogger(lg)
		tf.SetStdout(log.Writer())
		tf.SetStderr(log.Writer())
	} else {
		logger.Info("Terraform output is muted")
	}

	// This is equivalent to setting TF_LOG_PATH=$WORKDIR/terraform.log.
	// tfexec only supports TRACE log level which results in verbose logging.
	// Caution: Do not run in production, and may be useful for debugging and
	// development purposes. There is no log rotation and may quickly result
	// large files.
	if config.PersistLog {
		logPath := filepath.Join(config.WorkingDir, "terraform.log")
		tf.SetLogPath(logPath)
		logger.Info("persisting Terraform logs on disk", "logPath", logPath)
	}

	client := &TerraformCLI{
		tf:         tf,
		workingDir: config.WorkingDir,
		workspace:  config.Workspace,
		logger:     logger,
	}
	logger.Trace("created Terraform CLI client", "client", client.GoString())

	return client, nil
}

// SetEnv sets the environment for the Terraform workspace
func (t *TerraformCLI) SetEnv(env map[string]string) error {
	return t.tf.SetEnv(env)
}

// SetStdout sets the standard out for Terraform
func (t *TerraformCLI) SetStdout(w io.Writer) {
	t.tf.SetStdout(w)
}

// Init initializes by executing the cli command `terraform init` and
// `terraform workspace new <name>`
func (t *TerraformCLI) Init(ctx context.Context) error {
	var wsCreated bool

	// This is special handling for when the workspace has been detected in
	// .terraform/environment with a non-existing state. This case is common
	// when the state for the workspace has been deleted.
	// https://github.com/hashicorp/terraform/issues/21393
TF_INIT_AGAIN:
	if err := t.tf.Init(ctx); err != nil {
		var wsErr *tfexec.ErrNoWorkspace
		matched := wsFailedToSelectRegexp.MatchString(err.Error())
		if matched || errors.As(err, &wsErr) {
			t.logger.Info("workspace was detected without state, " +
				"creating new workspace and attempting Terraform init again")
			if err := t.tf.WorkspaceNew(ctx, t.workspace); err != nil {
				return err
			}

			if !wsCreated {
				wsCreated = true
				goto TF_INIT_AGAIN
			}
		}
		return err
	}

	logws := t.logger.With("workspace", t.workspace)
	if !wsCreated {
		err := t.tf.WorkspaceNew(ctx, t.workspace)
		if err != nil {
			var wsErr *tfexec.ErrWorkspaceExists
			if !errors.As(err, &wsErr) {
				logws.Error("unable to create workspace", "error", err)
				return err
			}
			logws.Debug("workspace already exists", "error", err)
		} else {
			logws.Trace("workspace created")
		}
	}

	if err := t.tf.WorkspaceSelect(ctx, t.workspace); err != nil {
		logws.Error("unable to change workspace", "error", err)
		return err
	}

	return nil
}

// Apply executes the cli command `terraform apply` for a given workspace
func (t *TerraformCLI) Apply(ctx context.Context) error {
	return t.tf.Apply(ctx)
}

// Plan executes the cli command `terraform plan` for a given workspace
func (t *TerraformCLI) Plan(ctx context.Context) (bool, error) {
	return t.tf.Plan(ctx)
}

// Validate verifies the generated configuration files
func (t *TerraformCLI) Validate(ctx context.Context) error {
	output, err := t.tf.Validate(ctx)
	if err != nil {
		return err
	}

	var sb strings.Builder
	for _, d := range output.Diagnostics {
		sb.WriteByte('\n')
		switch d.Detail {
		case `An argument named "services" is not expected here.`:
			fmt.Fprintf(&sb, `module for task "%s" is missing the "services" variable`, t.workspace)
		case `An argument named "catalog_services" is not expected here.`:
			fmt.Fprintf(
				&sb,
				`module for task "%s" is missing the "catalog_services" variable, add to module or set "use_as_module_input" to false`,
				t.workspace)
		default:
			fmt.Fprintf(&sb, "%s: %s\n", d.Severity, d.Summary)
			if d.Range != nil && d.Snippet != nil {
				if d.Snippet.Context != nil {
					fmt.Fprintf(&sb, "\non %s line %d, in %s\n",
						d.Range.Filename, d.Range.Start.Line, *d.Snippet.Context)
				} else {
					fmt.Fprintf(&sb, "\non %s line %d\n", d.Range.Filename, d.Range.Start.Line)
				}
				fmt.Fprintf(&sb, "%d:%s\n\n", d.Snippet.StartLine, d.Snippet.Code)
			}
			sb.WriteString(d.Detail)
		}
		sb.WriteByte('\n')
	}

	if !output.Valid {
		return fmt.Errorf(sb.String())
	}

	if sb.Len() > 0 {
		t.logger.Warn("Terraform validate returned warnings", "warnings", sb.String())
	}

	return nil
}

// GoString defines the printable version of this struct.
func (t *TerraformCLI) GoString() string {
	if t == nil {
		return "(*TerraformCLI)(nil)"
	}

	return fmt.Sprintf("&TerraformCLI{"+
		"WorkingDir:%s, "+
		"WorkSpace:%s, "+
		"}",
		t.workingDir,
		t.workspace,
	)
}
