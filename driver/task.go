package driver

import (
	"context"

	"github.com/hashicorp/consul-nia/client"
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
	Providers    []map[string]interface{}
	ProviderInfo map[string]interface{}
	Services     []Service
	Source       string
	VarFiles     []string
	Version      string
}

// worker executes a unit of work and has a one-to-one relationship with a client
// that will be responsible for executing the work. Currently worker is not safe for
// concurrent use by multiple goroutines
type worker struct {
	client client.Client
	task   Task
}

func (w *worker) init(ctx context.Context) error {
	if err := w.client.Init(ctx); err != nil {
		return err
	}
	return nil
}

func (w *worker) apply(ctx context.Context) error {
	if err := w.client.Apply(ctx); err != nil {
		return err
	}
	return nil
}
