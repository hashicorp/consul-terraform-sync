package client

import (
	"context"
	"fmt"
	"log"
)

var _ Client = (*Printer)(nil)

// Printer is a fake client that only logs out actions. Intended to mirror
// TerraformClient and to be used for development only
type Printer struct {
	logLevel   string
	workingDir string
	workspace  string
}

// PrinterConfig configures the log client
type PrinterConfig struct {
	LogLevel   string
	ExecPath   string
	WorkingDir string
	Workspace  string
}

// NewPrinter creates a new client
func NewPrinter(config *PrinterConfig) (*Printer, error) {
	return &Printer{
		logLevel:   config.LogLevel,
		workingDir: config.WorkingDir,
		workspace:  config.Workspace,
	}, nil
}

// Init logs out 'init'
func (l *Printer) Init(ctx context.Context) error {
	log.Printf("[INFO] (client.printer) initing workspace: '%s', workingdir: '%s'",
		l.workspace, l.workingDir)
	return nil
}

// Apply logs out 'apply'
func (l *Printer) Apply(ctx context.Context) error {
	log.Printf("[INFO] (client.printer) applying workspace: '%s', workingdir: '%s'",
		l.workspace, l.workingDir)
	return nil
}

// Plan logs out 'plan'
func (l *Printer) Plan(ctx context.Context) error {
	log.Printf("[INFO] (client.printer) planning workspace: '%s', workingdir: '%s'",
		l.workspace, l.workingDir)
	return nil
}

// GoString defines the printable version of this struct.
func (l *Printer) GoString() string {
	if l == nil {
		return "(*Printer)(nil)"
	}

	return fmt.Sprintf("&Printer{"+
		"LogLevel:%s, "+
		"WorkingDir:%s, "+
		"WorkSpace:%s, "+
		"}",
		l.logLevel,
		l.workingDir,
		l.workspace,
	)
}
