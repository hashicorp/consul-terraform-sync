package driver

import (
	"log"
	"time"

	"github.com/hashicorp/consul-terraform-sync/client"
	"github.com/hashicorp/consul-terraform-sync/config"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/client"
)

const (
	// RunOptionNow runs the task immediate (now) once the task has been updated
	RunOptionNow = "now"
	// RunOptionInspect does a dry-run task update and returns dry-run info
	RunOptionInspect = "inspect"
)

// PatchTask holds the information to patch update a task. It will only include
// fields that we support updating at this time
type PatchTask struct {
	// RunOption is a set of options on how to handle the patch update
	// current options are "now" and "inspect". See constants for more details
	RunOption string

	Enabled bool
}

// Service contains service configuration information
type Service struct {
	Datacenter  string
	Description string
	Name        string
	Namespace   string
	Tag         string
}

// BufferPeriod contains the task's buffer period configuration information
// if enabled
type BufferPeriod struct {
	Min time.Duration
	Max time.Duration
}

// Task contains task configuration information
type Task struct {
	Description     string
	Name            string
	Enabled         bool
	Env             map[string]string
	Providers       TerraformProviderBlocks // task.providers config info
	ProviderInfo    map[string]interface{}  // driver.required_provider config info
	Services        []Service
	Source          string
	VarFiles        []string
	Version         string
	UserDefinedMeta map[string]map[string]string
	BufferPeriod    *BufferPeriod // nil when disabled
	Condition       config.ConditionConfig
}

// ProviderNames returns the list of providers that the task has configured
func (t *Task) ProviderNames() []string {
	names := make([]string, len(t.Providers))
	for ix, p := range t.Providers {
		names[ix] = p.Name()
	}
	return names
}

// ServiceNames returns the list of services that the task has configured
func (t *Task) ServiceNames() []string {
	names := make([]string, len(t.Services))
	for ix, s := range t.Services {
		names[ix] = s.Name
	}
	return names
}

// clientConfig configures a driver client for a task
type clientConfig struct {
	task       Task
	clientType string
	log        bool
	persistLog bool
	path       string
	workingDir string
}

// newClient initializes a specific type of client given a task
func newClient(conf *clientConfig) (client.Client, error) {
	var err error
	var c client.Client
	taskName := conf.task.Name

	switch conf.clientType {
	case developmentClient:
		log.Printf("[TRACE] (driver) creating development client for task '%s'", taskName)
		c, err = client.NewPrinter(&client.PrinterConfig{
			LogLevel:   "debug",
			ExecPath:   conf.path,
			WorkingDir: conf.workingDir,
			Workspace:  taskName,
		})
	case testClient:
		log.Printf("[TRACE] (driver) creating mock client for task '%s'", taskName)
		c = new(mocks.Client)
	default:
		log.Printf("[TRACE] (driver) creating terraform cli client for task '%s'", taskName)
		c, err = client.NewTerraformCLI(&client.TerraformCLIConfig{
			Log:        conf.log,
			PersistLog: conf.persistLog,
			ExecPath:   conf.path,
			WorkingDir: conf.workingDir,
			Workspace:  taskName,
			VarFiles:   conf.task.VarFiles,
		})
	}

	return c, err
}
