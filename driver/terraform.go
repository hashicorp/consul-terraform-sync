package driver

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/consul-nia/templates/tftmpl"
	"github.com/pkg/errors"
)

const (
	// Number of times to retry in addition to initial attempt
	defaultRetry = 2

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

// Terraform is an NIA driver that uses the Terraform CLI to interface with
// low-level network infrastructure.
type Terraform struct {
	log               bool
	persistLog        bool
	path              string
	workingDir        string
	worker            *worker
	backend           map[string]interface{}
	requiredProviders map[string]interface{}

	version    string
	clientType string
}

// TerraformConfig configures the Terraform driver
type TerraformConfig struct {
	Log               bool
	PersistLog        bool
	Path              string
	WorkingDir        string
	Backend           map[string]interface{}
	RequiredProviders map[string]interface{}

	// empty/unknown string will default to TerraformCLI client
	ClientType string
}

// NewTerraform configures and initializes a new Terraform driver
func NewTerraform(config *TerraformConfig) *Terraform {
	return &Terraform{
		log:               config.Log,
		persistLog:        config.PersistLog,
		path:              config.Path,
		workingDir:        config.WorkingDir,
		backend:           config.Backend,
		requiredProviders: config.RequiredProviders,
		clientType:        config.ClientType,
	}
}

// Init initializes the Terraform local environment. The Terraform binary is
// installed to the configured path.
func (tf *Terraform) Init(ctx context.Context) error {
	if _, err := os.Stat(tf.workingDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tf.workingDir, workingDirPerms); err != nil {
			log.Printf("[ERR] (driver.terraform) error creating base work directory: %s", err)
			return err
		}
	}

	if isTFInstalled(tf.path) {
		tfVersion, compatible, err := isTFCompatible(ctx, tf.workingDir, tf.path)
		if err != nil {
			if strings.Contains(err.Error(), "exec format error") {
				return errIncompatibleTerraformBinary
			}
			return err
		}

		if !compatible {
			return errUnsupportedTerraformVersion
		}

		tf.version = tfVersion.String()
		log.Printf("[INFO] (driver.terraform) skipping install, terraform %s "+
			"already exists at path %s/terraform", tf.version, tf.path)
		return nil
	}

	log.Printf("[INFO] (driver.terraform) installing terraform to path '%s'", tf.path)
	if err := tf.install(ctx); err != nil {
		log.Printf("[ERR] (driver.terraform) error installing terraform: %s", err)
		return err
	}
	log.Printf("[INFO] (driver.terraform) successfully installed terraform")
	return nil
}

// Version returns the Terraform CLI version for the Terraform driver.
func (tf *Terraform) Version() string {
	return tf.version
}

// InitTask initializes the task by creating the Terraform root module and
// client to execute task.
func (tf *Terraform) InitTask(task Task, force bool) error {
	modulePath := filepath.Join(tf.workingDir, task.Name)
	if _, err := os.Stat(modulePath); os.IsNotExist(err) {
		if err := os.Mkdir(modulePath, workingDirPerms); err != nil {
			log.Printf("[ERR] (driver.terraform) error creating task work directory: %s", err)
			return err
		}
	}

	services := make([]*tftmpl.Service, len(task.Services))
	for i, s := range task.Services {
		services[i] = &tftmpl.Service{
			Datacenter:  s.Datacenter,
			Description: s.Description,
			Name:        s.Name,
			Namespace:   s.Namespace,
			Tag:         s.Tag,
		}
	}

	var vars tftmpl.Variables
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
		Providers:    task.Providers,
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

	if err := tftmpl.InitRootModule(&input, modulePath, filePerms, force); err != nil {
		return err
	}

	worker, err := newWorker(&workerConfig{
		task:       task,
		clientType: tf.clientType,
		log:        tf.log,
		persistLog: tf.persistLog,
		path:       tf.path,
		workingDir: tf.workingDir,
	})
	if err != nil {
		log.Printf("[ERR] (driver.terraform) init worker error: %s", err)
		return err
	}
	tf.worker = worker
	return nil
}

// ApplyTask applies the task changes.
func (tf *Terraform) ApplyTask(ctx context.Context) error {
	w := tf.worker
	taskName := w.task.Name

	if w.inited {
		log.Printf("[TRACE] (driver.terraform) already inited, skip for '%s'", taskName)
	} else {
		log.Printf("[TRACE] (driver.terraform) init '%s'", taskName)
		if err := w.init(ctx); err != nil {
			log.Printf("[ERR] (driver.terraform) init (skip apply) for '%s'", taskName)
			return errors.Wrap(err, fmt.Sprintf("Error tf-init for '%s'", taskName))
		}
	}

	log.Printf("[TRACE] (driver.terraform) apply '%s'", taskName)
	if err := w.apply(ctx); err != nil {
		return errors.Wrap(err, fmt.Sprintf("Error tf-apply for '%s'", taskName))
	}
	return nil
}
