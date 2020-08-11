package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
)

var _ Client = (*Printer)(nil)

// Printer is a fake client that only logs out actions. Intended to mirror
// TerraformCLI client and to be used for development only
type Printer struct {
	taskName   string
	logLevel   string
	workingDir string
	workspace  string
	logger     *log.Logger
}

// PrinterConfig configures the log client
type PrinterConfig struct {
	TaskName   string
	LogLevel   string
	ExecPath   string
	WorkingDir string
	Workspace  string
}

// NewPrinter creates a new client
func NewPrinter(config *PrinterConfig) (*Printer, error) {
	if config == nil {
		return nil, errors.New("PrinterConfig cannot be nil - mirror Terraform CLI error")
	}
	return &Printer{
		taskName:   config.TaskName,
		logLevel:   config.LogLevel,
		workingDir: config.WorkingDir,
		workspace:  config.Workspace,
		// TODO: revisit to improve for long-term using a setup like
		// https://github.com/hashicorp/consul-nia/blob/master/logging/logging.go#L34
		logger: log.New(os.Stdout, "", log.Ldate|log.Lmicroseconds),
	}, nil
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
func (p *Printer) Plan(ctx context.Context) error {
	p.logger.Printf("[INFO] (client.printer) planning workspace: '%s', workingdir: '%s'",
		p.workspace, p.workingDir)
	return nil
}

// GoString defines the printable version of this struct.
func (p *Printer) GoString() string {
	if p == nil {
		return "(*Printer)(nil)"
	}

	return fmt.Sprintf("&Printer{"+
		"TaskName:%s, "+
		"LogLevel:%s, "+
		"WorkingDir:%s, "+
		"WorkSpace:%s, "+
		"}",
		p.taskName,
		p.logLevel,
		p.workingDir,
		p.workspace,
	)
}
