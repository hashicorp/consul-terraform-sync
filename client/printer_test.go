package client

import (
	"bytes"
	"context"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPrinter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		config      *PrinterConfig
	}{
		{
			"error nil config",
			true,
			nil,
		},
		{
			"happy path",
			false,
			&PrinterConfig{
				LogLevel:   "INFO",
				WorkingDir: "path/to/wd",
				Workspace:  "ws",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := NewPrinter(tc.config)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, actual)
			assert.Equal(t, tc.config.LogLevel, actual.logLevel)
			assert.Equal(t, tc.config.WorkingDir, actual.workingDir)
			assert.Equal(t, tc.config.Workspace, actual.workspace)
		})
	}
}

func DefaultTestPrinter() (*Printer, error) {
	return NewPrinter(&PrinterConfig{
		LogLevel:   "INFO",
		WorkingDir: "path/to/wd",
		Workspace:  "ws",
	})
}

func TestPrinterInit(t *testing.T) {
	// no t.Parallel() because changing log output. occasionally fails when parallel.

	p, err := DefaultTestPrinter()
	assert.NoError(t, err)

	// output log to a buffer
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	ctx := context.Background()
	p.Init(ctx)
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "client.printer")
	assert.Contains(t, buf.String(), "init")
}

func TestPrinterApply(t *testing.T) {
	// no t.Parallel() because changing log output. occasionally fails when parallel.

	p, err := DefaultTestPrinter()
	assert.NoError(t, err)

	// output log to a buffer
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	ctx := context.Background()
	p.Apply(ctx)
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "client.printer")
	assert.Contains(t, buf.String(), "apply")
}

func TestPrinterPlan(t *testing.T) {
	// no t.Parallel() because changing log output. occasionally fails when parallel.

	p, err := DefaultTestPrinter()
	assert.NoError(t, err)

	// output log to a buffer
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	ctx := context.Background()
	p.Plan(ctx)
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "client.printer")
	assert.Contains(t, buf.String(), "plan")
}

func TestPrinterGoString(t *testing.T) {
	cases := []struct {
		name    string
		printer *Printer
	}{
		{
			"nil printer",
			nil,
		},
		{
			"happy path",
			&Printer{
				logLevel:   "INFO",
				workingDir: "path/to/wd",
				workspace:  "ws",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.printer == nil {
				assert.Contains(t, tc.printer.GoString(), "nil")
				return
			}

			assert.Contains(t, tc.printer.GoString(), "&Printer")
			assert.Contains(t, tc.printer.GoString(), tc.printer.logLevel)
			assert.Contains(t, tc.printer.GoString(), tc.printer.workingDir)
			assert.Contains(t, tc.printer.GoString(), tc.printer.workspace)
		})
	}
}
