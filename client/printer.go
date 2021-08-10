package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/hashicorp/consul-terraform-sync/logging"
)

var _ Client = (*Printer)(nil)

// Printer is a fake client that only logs out actions. Intended to mirror
// TerraformCLI client and to be used for development only
type Printer struct {
	logLevel   string
	workingDir string
	workspace  string
	logger     *log.Logger
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

	logger, err := logging.SetupLocal(config.Writer)
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
	p.logger.Printf("[INFO] (client.printer) setting workspace environment: "+
		"'%s', workingdir: '%s'", p.workspace, p.workingDir)
	return nil
}

// SetStdout logs out 'set standard out'
func (p *Printer) SetStdout(w io.Writer) {
	p.logger.Printf("[INFO] (client.printer) setting standard out for workspace: "+
		"'%s', workingdir: '%s'", p.workspace, p.workingDir)
}

// Init logs out 'init'
func (p *Printer) Init(ctx context.Context) error {
	p.logger.Printf("[INFO] (client.printer) initing workspace: '%s', workingdir: '%s'",
		p.workspace, p.workingDir)
	return nil
}

// Apply logs out 'apply'
func (p *Printer) Apply(ctx context.Context) error {
	p.logger.Printf("[INFO] (client.printer) applying workspace: '%s', workingdir: '%s'",
		p.workspace, p.workingDir)
	return nil
}

// Plan logs out 'plan'
func (p *Printer) Plan(ctx context.Context) (bool, error) {
	p.logger.Printf("[INFO] (client.printer) planning workspace: '%s', workingdir: '%s'",
		p.workspace, p.workingDir)
	return true, nil
}

// Validate logs out 'validate'
func (p *Printer) Validate(ctx context.Context) error {
	p.logger.Printf("[INFO] (client.printer) validating workspace: '%s', workingdir: '%s'",
		p.workspace, p.workingDir)
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
