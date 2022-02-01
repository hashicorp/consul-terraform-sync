package client

import (
	"bytes"
	"context"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/stretchr/testify/assert"
)

func TestNewPrinter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		expectError      bool
		expectedLogLevel string
		config           *PrinterConfig
	}{
		{
			"error nil config",
			true,
			"",
			nil,
		},
		{
			"happy path debug log level",
			false,
			"DEBUG",
			&PrinterConfig{
				WorkingDir: "path/to/wd",
				Workspace:  "ws",
			},
		},
		{
			"happy path",
			false,
			"INFO",
			&PrinterConfig{
				WorkingDir: "path/to/wd",
				Workspace:  "ws",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logging.LogLevel = tc.expectedLogLevel
			actual, err := NewPrinter(tc.config)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, actual)
			assert.Equal(t, tc.expectedLogLevel, actual.logLevel)
			assert.Equal(t, tc.config.WorkingDir, actual.workingDir)
			assert.Equal(t, tc.config.Workspace, actual.workspace)
		})
	}
}

func DefaultTestPrinter(buf *bytes.Buffer) (*Printer, error) {
	printer, err := NewTestPrinter(&PrinterConfig{
		WorkingDir: "path/to/wd",
		Workspace:  "ws",
		Writer:     buf,
	})
	if err != nil {
		return nil, err
	}

	return printer, nil
}

func NewTestPrinter(config *PrinterConfig) (*Printer, error) {
	printer, err := NewPrinter(config)
	if err != nil {
		return nil, err
	}

	return printer, nil
}

func TestPrinterSetEnv(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p, err := DefaultTestPrinter(&buf)
	assert.NoError(t, err)

	err = p.SetEnv(make(map[string]string))
	assert.NoError(t, err)
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
	err = p.Init(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "client.printer")
	assert.Contains(t, buf.String(), "init")
}

func TestPrinterLogLevel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectWrite bool
		logLevel    string
		config      *PrinterConfig
	}{
		{
			"no write info log level",
			false,
			"INFO",
			&PrinterConfig{
				WorkingDir: "path/to/wd",
				Workspace:  "ws",
			},
		},
		{
			"write trace log level",
			true,
			"TRACE",
			&PrinterConfig{
				WorkingDir: "path/to/wd",
				Workspace:  "ws",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logging.LogLevel = tc.logLevel
			var buf bytes.Buffer
			tc.config.Writer = &buf
			p, err := NewTestPrinter(tc.config)
			assert.NoError(t, err)

			p.logger.Trace("Test Message")
			if tc.expectWrite {
				assert.NotEmpty(t, buf.String())
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestPrinterApply(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p, err := DefaultTestPrinter(&buf)
	assert.NoError(t, err)

	ctx := context.Background()
	err = p.Apply(ctx)
	assert.NoError(t, err)
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
	diff, err := p.Plan(ctx)
	assert.True(t, diff)
	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "client.printer")
	assert.Contains(t, buf.String(), "plan")
}

func TestPrinterValidate(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p, err := DefaultTestPrinter(&buf)
	assert.NoError(t, err)

	ctx := context.Background()
	err = p.Validate(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "client.printer")
	assert.Contains(t, buf.String(), "validating")
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
			assert.Contains(t, tc.printer.GoString(), tc.printer.workingDir)
			assert.Contains(t, tc.printer.GoString(), tc.printer.workspace)
		})
	}
}
