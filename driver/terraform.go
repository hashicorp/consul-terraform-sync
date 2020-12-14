package driver

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/consul-terraform-sync/client"
	"github.com/hashicorp/consul-terraform-sync/handler"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl"
	"github.com/pkg/errors"
)

const (
	// Types of clients that are alternatives to the default Terraform CLI client
	developmentClient = "development"
	testClient        = "test"

	// Permissions for created directories and files
	workingDirPerms = os.FileMode(0750) // drwxr-x---
	filePerms       = os.FileMode(0640) // -rw-r-----

	errSuggestion = "remove Terraform from the configured path or specify a new path to safely install a compatible version."
)

var (
	_ Driver = (*Terraform)(nil)

	errUnsupportedTerraformVersion = fmt.Errorf("unsupported Terraform version: %s", errSuggestion)
	errIncompatibleTerraformBinary = fmt.Errorf("incompatible Terraform binary: %s", errSuggestion)
)

// Terraform is an Sync driver that uses the Terraform CLI to interface with
// low-level network infrastructure.
type Terraform struct {
	task              Task
	backend           map[string]interface{}
	requiredProviders map[string]interface{}

	workingDir string
	client     client.Client
	postApply  handler.Handler

	inited bool
}

// TerraformConfig configures the Terraform driver
type TerraformConfig struct {
	Task              Task
	Log               bool
	PersistLog        bool
	Path              string
	WorkingDir        string
	Backend           map[string]interface{}
	RequiredProviders map[string]interface{}

	// empty/unknown string will default to TerraformCLI client
	ClientType string
}

// NewTerraform configures and initializes a new Terraform driver for a task.
// The underlying Terraform CLI client and out-of-band handlers are prepared.
func NewTerraform(config *TerraformConfig) (*Terraform, error) {
	if _, err := os.Stat(config.WorkingDir); os.IsNotExist(err) {
		if err := os.Mkdir(config.WorkingDir, workingDirPerms); err != nil {
			log.Printf("[ERR] (driver.terraform) error creating task work directory: %s", err)
			return nil, err
		}
	}

	tfClient, err := newClient(&clientConfig{
		task:       config.Task,
		clientType: config.ClientType,
		log:        config.Log,
		persistLog: config.PersistLog,
		path:       config.Path,
		workingDir: config.WorkingDir,
	})
	if err != nil {
		log.Printf("[ERR] (driver.terraform) init client type '%s' error: %s", config.ClientType, err)
		return nil, err
	}
	if env := config.Task.Providers.Env(); len(env) > 0 {
		tfClient.SetEnv(env)
	}

	handler, err := getTerraformHandlers(config.Task)
	if err != nil {
		return nil, err
	}

	return &Terraform{
		task:              config.Task,
		backend:           config.Backend,
		requiredProviders: config.RequiredProviders,
		workingDir:        config.WorkingDir,
		client:            tfClient,
		postApply:         handler,
	}, nil
}

// Version returns the Terraform CLI version for the Terraform driver.
func (tf *Terraform) Version() string {
	return TerraformVersion
}

// InitTask initializes the task by creating the Terraform root module and related
// files to execute on.
func (tf *Terraform) InitTask(force bool) error {
	task := tf.task

	services := make([]tftmpl.Service, len(task.Services))
	for i, s := range task.Services {
		services[i] = tftmpl.Service{
			Datacenter:  s.Datacenter,
			Description: s.Description,
			Name:        s.Name,
			Namespace:   s.Namespace,
			Tag:         s.Tag,
		}
	}

	var vars hcltmpl.Variables
	for _, vf := range task.VarFiles {
		tfvars, err := tftmpl.LoadModuleVariables(vf)
		if err != nil {
			return err
		}

		if len(vars) == 0 {
			vars = tfvars
			continue
		}

		for k, v := range tfvars {
			vars[k] = v
		}
	}

	input := tftmpl.RootModuleInputData{
		Backend:      tf.backend,
		Providers:    task.Providers.ProviderBlocks(),
		ProviderInfo: task.ProviderInfo,
		Services:     services,
		Task: tftmpl.Task{
			Description: task.Description,
			Name:        task.Name,
			Source:      task.Source,
			Version:     task.Version,
		},
		Variables: vars,
	}
	input.Init()

	if err := tftmpl.InitRootModule(&input, tf.workingDir, filePerms, force); err != nil {
		return err
	}

	return nil
}

// InspectTask inspects for any differences pertaining to the task between
// the state of Consul and network infrastructure using the Terraform plan command
func (tf *Terraform) InspectTask(ctx context.Context) error {
	taskName := tf.task.Name

	if err := tf.init(ctx); err != nil {
		log.Printf("[ERR] (driver.terraform) error initializing workspace, "+
			"skipping plan for '%s'", taskName)
		return err
	}

	log.Printf("[TRACE] (driver.terraform) plan '%s'", taskName)
	if err := tf.client.Plan(ctx); err != nil {
		return errors.Wrap(err, fmt.Sprintf("error tf-plan for '%s'", taskName))
	}

	return nil
}

// ApplyTask applies the task changes.
func (tf *Terraform) ApplyTask(ctx context.Context) error {
	taskName := tf.task.Name

	if err := tf.init(ctx); err != nil {
		log.Printf("[ERR] (driver.terraform) error initializing workspace, "+
			"skipping apply for '%s'", taskName)
		return err
	}

	log.Printf("[TRACE] (driver.terraform) apply '%s'", taskName)
	if err := tf.client.Apply(ctx); err != nil {
		return errors.Wrap(err, fmt.Sprintf("error tf-apply for '%s'", taskName))
	}

	if tf.postApply != nil {
		log.Printf("[TRACE] (driver.terraform) post-apply out-of-band actions "+
			"for '%s'", taskName)
		if err := tf.postApply.Do(nil); err != nil {
			return err
		}
	}

	return nil
}

// init initializes the Terraform workspace if needed
func (tf *Terraform) init(ctx context.Context) error {
	taskName := tf.task.Name

	if tf.inited {
		log.Printf("[TRACE] (driver.terraform) workspace for task already "+
			"initialized, skipping for '%s'", taskName)
		return nil
	}

	log.Printf("[TRACE] (driver.terraform) initializing workspace '%s'", taskName)
	if err := tf.client.Init(ctx); err != nil {
		return errors.Wrap(err, fmt.Sprintf("error tf-init for '%s'", taskName))
	}
	tf.inited = true
	return nil
}

// getTerraformHandlers returns the first handler in a chain of handlers
// for a Terraform driver.
//
// Returned handler may be nil even if returned err is nil. This happens when
// no providers have a handler.
func getTerraformHandlers(task Task) (handler.Handler, error) {
	counter := 0
	var next handler.Handler
	for _, p := range task.Providers {
		h, err := handler.TerraformProviderHandler(p.Name(), p.ProviderBlock().RawConfig())
		if err != nil {
			log.Printf(
				"[ERR] (driver.terraform) could not initialize handler for "+
					"provider '%s': %s", p.Name(), err)
			return nil, err
		}
		if h != nil {
			counter++
			log.Printf(
				"[INFO] (driver.terraform) retrieved handler for provider '%s'", p.Name())
			h.SetNext(next)
			next = h
		}
	}
	log.Printf("[INFO] (driver.terraform) retrieved %d Terraform handlers for task '%s'",
		counter, task.Name)
	return next, nil
}
