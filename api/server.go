// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"context"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/state/event"
)

//go:generate mockery --name=Server --filename=server.go --output=../mocks/server

// Server represents the Controller methods used for the API server
type Server interface {
	Config() config.Config
	Events(ctx context.Context, taskName string) (map[string][]event.Event, error)

	Task(ctx context.Context, taskName string) (config.TaskConfig, error)
	TaskCreate(context.Context, config.TaskConfig) (config.TaskConfig, error)
	TaskCreateAndRun(context.Context, config.TaskConfig) (config.TaskConfig, error)
	TaskDelete(ctx context.Context, taskName string) error
	// TODO: update signatures to return a new run object
	TaskInspect(context.Context, config.TaskConfig) (bool, string, string, error)
	// TODO: update signature with an update config object since only a subset of
	// options can be changed and determine the location of sharable objects
	// across packages
	TaskUpdate(ctx context.Context, updateConf config.TaskConfig, runOp string) (bool, string, string, error)
	Tasks(context.Context) config.TaskConfigs
}
