package driver

import (
	"context"
	"fmt"
	"strings"

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
	work   *work
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

// work represents a standalone unit of work that can be executed concurrently alongside others
// or sequentially amongst others. Currently this an individual task. Instances not supported yet.
type work struct {
	task Task
	desc string
	// instance
}

// String returns brief description of work
func (w *work) String() string {
	if w == nil {
		return "nil"
	}

	if len(w.desc) > 0 {
		return w.desc
	}

	providers := make([]string, len(w.task.Providers))
	for ix, p := range w.task.Providers {
		for k := range p {
			providers[ix] = k
			break // 1 map entry per provider
		}
	}

	w.desc = fmt.Sprintf("TaskName: '%s', "+
		"TaskProviders: '%s'",
		w.task.Name,
		strings.Join(providers, ", "),
	)

	return w.desc
}
