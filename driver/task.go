package driver

import (
	"log"

	"github.com/hashicorp/consul-terraform-sync/client"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/client"
)

// Service contains service configuration information
type Service struct {
	Datacenter  string
	Description string
	Name        string
	Namespace   string
	Tag         string
}

// Task contains task configuration information
type Task struct {
	Description  string
	Name         string
	Providers    []map[string]interface{} // task.providers config info
	ProviderInfo map[string]interface{}   // driver.required_provider config info
	Services     []Service
	Source       string
	VarFiles     []string
	Version      string
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
