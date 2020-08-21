package driver

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/consul-nia/client"
	"github.com/hashicorp/consul-nia/templates/tftmpl"
)

const (
	terraformVersion = "0.13.0-rc1"
	releasesURL      = "https://releases.hashicorp.com"

	// Types of clients that are alternatives to the default Terraform CLI client
	developmentClient = "development"
	testClient        = "test"
)

var _ Driver = (*Terraform)(nil)

// Terraform is an NIA driver that uses the Terraform CLI to interface with
// low-level network infrastructure.
type Terraform struct {
	logLevel          string
	path              string
	dataDir           string
	workingDir        string
	skipVerify        bool
	workers           []*worker
	backend           map[string]interface{}
	requiredProviders map[string]interface{}

	version    string
	clientType string
}

// TerraformConfig configures the Terraform driver
type TerraformConfig struct {
	LogLevel          string
	Path              string
	DataDir           string
	WorkingDir        string
	SkipVerify        bool
	Backend           map[string]interface{}
	RequiredProviders map[string]interface{}

	// empty/unknown string will default to TerraformCLI client
	ClientType string
}

// NewTerraform configures and initializes a new Terraform driver
func NewTerraform(config *TerraformConfig) *Terraform {
	return &Terraform{
		logLevel:          config.LogLevel,
		path:              config.Path,
		dataDir:           config.DataDir,
		workingDir:        config.WorkingDir,
		skipVerify:        config.SkipVerify,
		backend:           config.Backend,
		requiredProviders: config.RequiredProviders,

		// TODO: the version is currently hard-coded. NIA should discover
		// the latest patch version within the minor version.
		version:    terraformVersion,
		clientType: config.ClientType,
	}
}

// Init initializes the Terraform local environment. The Terraform binary is
// installed to the configured path.
func (tf *Terraform) Init() error {
	if !terraformInstalled(tf.path) {
		log.Printf("[INFO] (driver.terraform) installing terraform (%s) to path '%s'", tf.version, tf.path)
		if err := tf.install(); err != nil {
			log.Printf("[ERR] (driver.terraform) error installing terraform: %s", err)
			return err
		}
		log.Printf("[INFO] (driver.terraform) successfully installed terraform")
	} else {
		log.Printf("[INFO] (driver.terraform) skipping install, terraform "+
			"already exists at path %s/terraform", tf.path)
	}

	return nil
}

// Version returns the Terraform CLI version for the Terraform driver.
func (tf *Terraform) Version() string {
	return tf.version
}

// InitTask initializes the Terraform root module for the task.
func (tf *Terraform) InitTask(task Task, force bool) error {
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
	if task.VariablesFile != "" {
		var err error
		vars, err = tftmpl.LoadModuleVariables(task.VariablesFile)
		if err != nil {
			return err
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
	return tftmpl.InitRootModule(&input, tf.workingDir, force)
}

// InitWorker given a task, identifies a unit of work and creates a worker for it.
// Worker is added to the driver. Currently assumes a task has a single instance of
// a provider and is therefore equivalent to a unit of work.
// TODO: multiple provider instances
func (tf *Terraform) InitWorker(task Task) error {
	client, err := tf.initClient(task)
	if err != nil {
		log.Printf("[ERR] (driver.terraform) init client type '%s' error: %s", tf.clientType, err)
		return err
	}

	tf.workers = append(tf.workers, &worker{
		client: client,
		work:   &work{task},
	})
	return nil
}

// initClient initializes a specific type of client given a task
func (tf *Terraform) initClient(task Task) (client.Client, error) {
	var c client.Client
	var err error

	switch tf.clientType {
	case developmentClient:
		log.Printf("[TRACE] (driver.terraform) creating development client for task '%s'", task.Name)
		c, err = client.NewPrinter(&client.PrinterConfig{
			LogLevel:   tf.logLevel,
			ExecPath:   tf.path,
			WorkingDir: fmt.Sprintf("%s/%s", tf.workingDir, task.Name),
			Workspace:  task.Name,
		})
	case testClient:
		log.Printf("[TRACE] (driver.terraform) creating mock client for task '%s'", task.Name)
		c = client.NewMockClient()
	default:
		log.Printf("[TRACE] (driver.terraform) creating terraform cli client for task '%s'", task.Name)
		c, err = client.NewTerraformCLI(&client.TerraformCLIConfig{
			LogLevel:   tf.logLevel,
			ExecPath:   tf.path,
			WorkingDir: fmt.Sprintf("%s/%s", tf.workingDir, task.Name),
			Workspace:  task.Name,
		})
	}

	return c, err
}

// InitWork initializes the client for all of the driver's workers
func (tf *Terraform) InitWork(ctx context.Context) error {
	var errs []string

	for _, r := range tf.workers {
		if err := r.client.Init(ctx); err != nil {
			log.Printf("[ERR] (driver.terraform) init work %s error: %s", r.client.GoString(), err)
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		// TODO: handle errors better when we do concurrency
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	return nil
}

// ApplyWork applies changes for all of the driver's workers
func (tf *Terraform) ApplyWork(ctx context.Context) error {
	var errs []string

	for _, r := range tf.workers {
		log.Printf("[TRACE] (driver.terraform) apply work for worker %s", r.client.GoString())
		if err := r.client.Apply(ctx); err != nil {
			log.Printf("[ERR] (driver.terraform) apply work %s error: %s", r.client.GoString(), err)
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		// TODO: handle errors better when we do concurrency
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	return nil
}
