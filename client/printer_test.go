package client

import (
	"bytes"
	"context"
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

func DefaultTestPrinter(buf *bytes.Buffer) (*Printer, error) {
	printer, err := NewPrinter(&PrinterConfig{
		LogLevel:   "INFO",
		WorkingDir: "path/to/wd",
		Workspace:  "ws",
	})
	if err != nil {
		return nil, err
	}

	// Overwrite the printer's logger so that we can look at contents
	printer.logger.SetOutput(buf)
	return printer, nil
}

func TestPrinterSetEnv(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p, err := DefaultTestPrinter(&buf)
	assert.NoError(t, err)

	p.SetEnv(make(map[string]string))
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "client.printer")
	assert.Contains(t, buf.String(), "set")
	assert.Contains(t, buf.String(), "env")
}

func TestPrinterSetStdout(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p, err := DefaultTestPrinter(&buf)
	assert.NoError(t, err)

	p.SetStdout(&buf)
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "client.printer")
	assert.Contains(t, buf.String(), "set")
	assert.Contains(t, buf.String(), "standard out")
}

func TestPrinterInit(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p, err := DefaultTestPrinter(&buf)
	assert.NoError(t, err)

	ctx := context.Background()
	p.Init(ctx)
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "client.printer")
	assert.Contains(t, buf.String(), "init")
}

func TestPrinterApply(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p, err := DefaultTestPrinter(&buf)
	assert.NoError(t, err)

	ctx := context.Background()
	p.Apply(ctx)
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "client.printer")
	assert.Contains(t, buf.String(), "apply")
}

func TestPrinterPlan(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p, err := DefaultTestPrinter(&buf)
	assert.NoError(t, err)

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
