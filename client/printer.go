// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/hashicorp/consul-terraform-sync/logging"
)

var _ Client = (*Printer)(nil)

const (
	printerSubsystemName = "printer"
	loggingSystemName    = "client"
)

// Printer is a fake client that only logs out actions. Intended to mirror
// TerraformCLI client and to be used for development only
type Printer struct {
	logLevel   string
	workingDir string
	workspace  string
	logger     logging.Logger
}

// PrinterConfig configures the log client
type PrinterConfig struct {
	ExecPath   string
	WorkingDir string
	Workspace  string
	Writer     io.Writer
}

// NewPrinter creates a new client
func NewPrinter(config *PrinterConfig) (*Printer, error) {
	if config == nil {
		return nil, errors.New("PrinterConfig cannot be nil - mirror Terraform CLI error")
	}

	logger, err := logging.SetupLocal(config.Writer, loggingSystemName, printerSubsystemName,
		"workspace", config.Workspace, "working_dir", config.WorkingDir)
	if err != nil {
		return &Printer{}, fmt.Errorf("error creating new logger: %s", err)
	}

	return &Printer{
		logLevel:   logging.LogLevel,
		workingDir: config.WorkingDir,
		workspace:  config.Workspace,
		logger:     logger,
	}, nil
}

// SetEnv logs out 'setenv'
func (p *Printer) SetEnv(map[string]string) error {
	p.logger.Info("setting environment for workspace")
	return nil
}

// SetStdout logs out 'set standard out'
func (p *Printer) SetStdout(io.Writer) {
	p.logger.Info("setting standard out for workspace")
}

// Init logs out 'init'
func (p *Printer) Init(context.Context) error {
	p.logger.Info("initing workspace")
	return nil
}

// Apply logs out 'apply'
func (p *Printer) Apply(context.Context) error {
	p.logger.Info("applying workspace")
	return nil
}

// Plan logs out 'plan'
func (p *Printer) Plan(context.Context) (bool, error) {
	p.logger.Info("planning workspace")
	return true, nil
}

// Validate logs out 'validate'
func (p *Printer) Validate(context.Context) error {
	p.logger.Info("validating workspace")
	return nil
}

// GoString defines the printable version of this struct.
func (p *Printer) GoString() string {
	if p == nil {
		return "(*Printer)(nil)"
	}

	return fmt.Sprintf("&Printer{"+
		"LogLevel:%s, "+
		"WorkingDir:%s, "+
		"WorkSpace:%s, "+
		"}",
		p.logLevel,
		p.workingDir,
		p.workspace,
	)
}
