package controller

import (
	"context"

	"github.com/hashicorp/consul-terraform-sync/config"
)

var _ Controller = (*ReadOnly)(nil)

// ReadOnly is the controller to run in read-only mode
type ReadOnly struct {
	conf *config.Config
}

// NewReadOnly configures and initializes a new ReadOnly controller
func NewReadOnly(conf *config.Config) *ReadOnly {
	return &ReadOnly{
		conf: conf,
	}
}

// Init initializes the controller before it can be run
func (ro *ReadOnly) Init(ctx context.Context) error {
	// TODO
	return nil
}

// Run runs the controller in read-only mode by checking Consul catalog once for
// latest and using the driver to plan network infrastructure changes
func (ro *ReadOnly) Run(ctx context.Context) error {
	// TODO
	return nil
}
