package logging

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/go-hclog"
	gsyslog "github.com/hashicorp/go-syslog"
)

// Levels are the log levels we respond too.
var Levels = []string{"TRACE", "DEBUG", "INFO", "WARN", "ERR"}

// LogLevel is the log level of all logs in this package
// it is set by the Setup() function and is shared with all other setup variations (i.e. SetupLocal())
var LogLevel = ""

const (
	defaultLogLevel = "INFO"
)

// Logger is a type alias for hclog.Logger
type Logger hclog.Logger

// Config is the configuration for this log setup.
type Config struct {
	// Level is the log level to use.
	Level string

	// Syslog and SyslogFacility is the syslog configuration options.
	Syslog         bool
	SyslogFacility string

	// SyslogName is the progname as it will appear in syslog output (if enabled).
	SyslogName string

	// Writer is the output where logs should go. If syslog is enabled, data will
	// be written to writer in addition to syslog.
	Writer io.Writer
}

// Setup takes as an arugment a configuration and then uses it to configure the global
// logger. After setup logging.Global() can be used to access the global logger
func Setup(config *Config) error {

	// Validate the log level
	if !validateLogLevel(config.Level) {
		return fmt.Errorf("Invalid log level: %s. Valid log levels are: %v",
			config.Level,
			Levels)
	}

	// Set the global log level, this will be used by all loggers
	setLogLevel(config.Level)

	var logOutput io.Writer
	// Check if syslog is enabled
	if config.Syslog {
		syslog, err := gsyslog.NewLogger(gsyslog.LOG_NOTICE, config.SyslogFacility, config.SyslogName)
		if err != nil {
			return fmt.Errorf("error setting up syslog logger: %s", err)
		}
		logOutput = io.MultiWriter(config.Writer, syslog)
	} else {
		logOutput = config.Writer
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.LevelFromString(LogLevel),
		Output:     logOutput,
		TimeFormat: hclog.TimeFormat,
	})

	hclog.SetDefault(logger)
	return nil
}

// Global returns the hclog.Default() global logger as type Logger
func Global() Logger {
	return hclog.Default()
}

// NewNullLogger returns the hclog.NewNullLogger() null logger as type Logger
func NewNullLogger() Logger {
	return hclog.NewNullLogger()
}

func DisableLogging() {
	hclog.Default().SetLevel(hclog.Off)
}

// WithContext stores a Logger and any provided arguments into the provided context
// to access this logger, use the FromContext function
func WithContext(ctx context.Context, logger Logger, args ...interface{}) context.Context {
	return hclog.WithContext(ctx, logger, args...)
}

// FromContext returns the Logger contained with the context. If no Logger is found in the context,
// this function returns the Global Logger
func FromContext(ctx context.Context) Logger {
	return hclog.FromContext(ctx)
}

// SetupLocal returns a new log.Logger which logs to a provided io.Writer
// The logger will filter logs based on the global log level
func SetupLocal(writer io.Writer, systemName string, subsystemName string, values ...interface{}) (Logger, error) {
	setLogLevel(LogLevel)

	// Create default logger
	logger := hclog.New(&hclog.LoggerOptions{
		Name:       systemName,
		Level:      hclog.LevelFromString(LogLevel),
		Output:     writer,
		TimeFormat: hclog.TimeFormat,
	})

	// Set the subsystem name
	logger = logger.Named(subsystemName).With(values...)
	return logger, nil
}

func setLogLevel(logLevel string) {
	if len(logLevel) == 0 {
		LogLevel = defaultLogLevel
	} else {
		LogLevel = logLevel
	}
}

// validateLogLevel verifies that a new log level is valid
func validateLogLevel(minLevel string) bool {
	newLevel := strings.ToUpper(minLevel)
	for _, level := range Levels {
		if level == newLevel {
			return true
		}
	}
	return false
}
