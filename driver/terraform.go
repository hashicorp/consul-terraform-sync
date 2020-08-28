package driver

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/consul-nia/client"
	mocks "github.com/hashicorp/consul-nia/mocks/client"
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
	workers           map[string]*worker // task-name -> worker
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
		workers:           make(map[string]*worker),
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

	tf.workers[task.Name] = &worker{
		client: client,
		work:   &work{task: task},
	}
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
		c = new(mocks.Client)
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

// InitWork initializes a single task/workers client
func (tf *Terraform) InitTaskWork(taskName string, ctx context.Context) error {
	if w, ok := tf.workers[taskName]; ok {
		log.Printf("[TRACE] (driver.terraform) go init for work: %v", w.work)
		return w.client.Init(ctx)
	}
	return fmt.Errorf("No task found with name: %s", taskName)
}

// InitWork initializes the client for all of the driver's workers concurrently
func (tf *Terraform) InitWork(ctx context.Context) error {
	resultCh := make(chan error)

	for _, w := range tf.workers {
		go func(ctx context.Context, w *worker) {
			log.Printf("[TRACE] (driver.terraform) go init for work: %v", w.work)
			resultCh <- w.client.Init(ctx)
		}(ctx, w)
	}

	count := 0
	var errs []string
	for err := range resultCh {
		count++
		log.Printf("[TRACE] (driver.terraform) init receive '%s'. count %d / %d", err, count, len(tf.workers))
		if err != nil {
			errs = append(errs, err.Error())
		}
		if count == len(tf.workers) {
			break
		}
	}

	if len(errs) > 0 {
		delim := "\n * "
		return fmt.Errorf("Received %d errors from init-ing %d workers:%s%s",
			len(errs), len(tf.workers), delim, strings.Join(errs, delim))
	}
	return nil
}

// ApplyTaskWork applies changes for all of the driver's workers concurrently
func (tf *Terraform) ApplyTaskWork(taskName string, ctx context.Context) error {
	if w, ok := tf.workers[taskName]; ok {
		log.Printf("[TRACE] (driver.terraform) go apply for work: %v", w.work)
		return w.client.Apply(ctx)
	}
	return fmt.Errorf("No task found with name: %s", taskName)
}

// ApplyWork applies changes for all of the driver's workers concurrently
func (tf *Terraform) ApplyWork(ctx context.Context) error {
	resultCh := make(chan error)

	for _, w := range tf.workers {
		go func(ctx context.Context, w *worker) {
			log.Printf("[TRACE] (driver.terraform) go apply for work: %v", w.work)
			resultCh <- w.client.Apply(ctx)
		}(ctx, w)
	}

	count := 0
	var errs []string
	for err := range resultCh {
		count++
		log.Printf("[TRACE] (driver.terraform) apply receive '%s'. count %d / %d", err, count, len(tf.workers))
		if err != nil {
			errs = append(errs, err.Error())
		}
		if count == len(tf.workers) {
			break
		}
	}

	if len(errs) > 0 {
		delim := "\n * "
		return fmt.Errorf("Received %d errors from applying %d workers:%s%s",
			len(errs), len(tf.workers), delim, strings.Join(errs, delim))
	}
	return nil
}
