package driver

import "github.com/hashicorp/consul-nia/client"

type Service struct {
	Datacenter  string
	Description string
	Name        string
	Namespace   string
	Tag         string
}

type Task struct {
	Description   string
	Name          string
	Providers     []map[string]interface{}
	ProviderInfo  map[string]interface{}
	Services      []Service
	Source        string
	VariablesFile string
	Version       string
}

// worker is executes a unit of work and has a one-to-one relationship with a client
// that will be responsible for executing the work.
type worker struct {
	client client.Client
	work   *work
}

// work represents a standalone unit of work that can be executed concurrently alongside others
// or sequentially amongst others. Currently this an individual task. Instances not supported yet.
type work struct {
	task Task
	// instance
}
