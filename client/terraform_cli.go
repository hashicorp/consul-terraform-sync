package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl"
	"github.com/hashicorp/terraform-exec/tfexec"
)

var (
	_ Client = (*TerraformCLI)(nil)

	wsFailedToSelectRegexp = regexp.MustCompile(`Failed to select workspace`)
)

// TerraformCLI is the client that wraps around terraform-exec
// to execute Terraform cli commands
type TerraformCLI struct {
	tf         terraformExec
	workingDir string
	workspace  string
	varFiles   []string
}

// TerraformCLIConfig configures the Terraform client
type TerraformCLIConfig struct {
	Log        bool
	PersistLog bool
	ExecPath   string
	WorkingDir string
	Workspace  string
	VarFiles   []string
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
	if config.Log {
		log.Printf("[INFO] (client.terraformcli) Terraform logging is set, " +
			"Terraform logs will output with Sync logs")
		logger := log.New(log.Writer(), "", log.Flags())
		tf.SetLogger(logger)
		tf.SetStdout(log.Writer())
		tf.SetStderr(log.Writer())
	} else {
		log.Printf("[INFO] (client.terraformcli) Terraform output is muted")
	}

	// This is equivalent to setting TF_LOG_PATH=$WORKDIR/terraform.log.
	// tfexec only supports TRACE log level which results in verbose logging.
	// Caution: Do not run in production, and may be useful for debugging and
	// development purposes. There is no log rotation and may quickly result
	// large files.
	if config.PersistLog {
		logPath := filepath.Join(config.WorkingDir, "terraform.log")
		tf.SetLogPath(logPath)
		log.Printf("[INFO] (client.terraformcli) persiting Terraform logs on disk: %s", logPath)
	}

	// Expand any relative paths for variable files to absolute paths
	var varFiles []string
	for _, vf := range config.VarFiles {
		if !strings.HasPrefix(vf, "/") {
			wd, err := os.Getwd()
			if err != nil {
				log.Println("[ERR] (client.terraformcli) unable to retrieve current " +
					"working directory to determine path to variable files")
				log.Panic(err)
			}
			vfAbs := filepath.Join(wd, vf)
			varFiles = append(varFiles, vfAbs)
		} else {
			varFiles = append(varFiles, vf)
		}
	}

	client := &TerraformCLI{
		tf:         tf,
		workingDir: config.WorkingDir,
		workspace:  config.Workspace,
		varFiles:   varFiles,
	}
	log.Printf("[TRACE] (client.terraformcli) created Terraform CLI client %s", client.GoString())

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
			log.Printf("[INFO] (client.terraformcli) workspace was detected without state, " +
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

	if !wsCreated {
		err := t.tf.WorkspaceNew(ctx, t.workspace)
		if err != nil {
			var wsErr *tfexec.ErrWorkspaceExists
			if !errors.As(err, &wsErr) {
				log.Printf("[ERR] (client.terraformcli) unable to create workspace: %q", t.workspace)
				return err
			}
			log.Printf("[DEBUG] (client.terraformcli) workspace already exists: '%s'", t.workspace)
		} else {
			log.Printf("[TRACE] (client.terraformcli) workspace created: %q", t.workspace)
		}
	}

	if err := t.tf.WorkspaceSelect(ctx, t.workspace); err != nil {
		log.Printf("[ERR] (client.terraformcli) unable to change workspace: %q", t.workspace)
		return err
	}

	return nil
}

// Apply executes the cli command `terraform apply` for a given workspace
func (t *TerraformCLI) Apply(ctx context.Context) error {
	// Pass along all tfvars files including ones generated by Sync
	opts := []tfexec.ApplyOption{
		tfexec.VarFile(tftmpl.TFVarsFilename),
		tfexec.VarFile(tftmpl.ProvidersTFVarsFilename),
	}
	for _, vf := range t.varFiles {
		opts = append(opts, tfexec.VarFile(vf))
	}

	return t.tf.Apply(ctx, opts...)
}

// Plan executes the cli command `terraform plan` for a given workspace
func (t *TerraformCLI) Plan(ctx context.Context) (bool, error) {
	// Pass along all tfvars files including ones generated by Sync
	opts := []tfexec.PlanOption{
		tfexec.VarFile(tftmpl.TFVarsFilename),
		tfexec.VarFile(tftmpl.ProvidersTFVarsFilename),
	}
	for _, vf := range t.varFiles {
		opts = append(opts, tfexec.VarFile(vf))
	}

	return t.tf.Plan(ctx, opts...)
}

// GoString defines the printable version of this struct.
func (t *TerraformCLI) GoString() string {
	if t == nil {
		return "(*TerraformCLI)(nil)"
	}

	return fmt.Sprintf("&TerraformCLI{"+
		"WorkingDir:%s, "+
		"WorkSpace:%s, "+
		"VarFiles:%s"+
		"}",
		t.workingDir,
		t.workspace,
		t.varFiles,
	)
}
