package logging

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	gsyslog "github.com/hashicorp/go-syslog"
	"github.com/hashicorp/logutils"
)

// Levels are the log levels we respond too.
var Levels = []logutils.LogLevel{"TRACE", "DEBUG", "INFO", "WARN", "ERR"}

// LogLevel is the log level of all logs in this package
// it is set by the Setup() function and is shared with all other setup variations (i.e. SetupLocal())
var LogLevel = ""

const (
	defaultLogLevel = "INFO"
)

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

func Setup(config *Config) error {
	// Set the global log level, this will be used by all loggers
	LogLevel = config.Level

	logFilter, err := setupFilter(config.Writer)
	if err != nil {
		return fmt.Errorf("error setting up log filter: %s", err)
	}

	var logOutput io.Writer

	// Check if syslog is enabled
	if config.Syslog {
		log.Printf("[INFO] (logging) enabling syslog on %s", config.SyslogFacility)

		l, err := gsyslog.NewLogger(gsyslog.LOG_NOTICE, config.SyslogFacility, config.SyslogName)
		if err != nil {
			return fmt.Errorf("error setting up syslog logger: %s", err)
		}
		syslog := &SyslogWrapper{l, logFilter}
		logOutput = io.MultiWriter(logFilter, syslog)
	} else {
		logOutput = io.MultiWriter(logFilter)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC)
	log.SetOutput(logOutput)

	return nil
}

// SetupLocal returns a new log.Logger which logs to a provided io.Writer
// The logger will filter logs based on the global log level
func SetupLocal(writer io.Writer) (*log.Logger, error) {
	// Create default logger
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC)
	logFilter, err := setupFilter(writer)
	if err != nil {
		return &log.Logger{}, fmt.Errorf("error setting up log filter: %s", err)
	}

	logOutput := io.Writer(logFilter)
	logger.SetOutput(logOutput)
	return logger, nil
}

// NewLogFilter returns a LevelFilter that is configured with the log levels that
// we use.
func NewLogFilter() *logutils.LevelFilter {
	return &logutils.LevelFilter{
		Levels:   Levels,
		MinLevel: "WARN",
		Writer:   ioutil.Discard,
	}
}

// ValidateLevelFilter verifies that the log levels within the filter are valid.
func ValidateLevelFilter(min logutils.LogLevel, filter *logutils.LevelFilter) bool {
	for _, level := range filter.Levels {
		if level == min {
			return true
		}
	}
	return false
}

func setupFilter(writer io.Writer) (*logutils.LevelFilter, error) {
	// Setup the default logging if nothing has been set
	if len(LogLevel) == 0 {
		LogLevel = defaultLogLevel
	}

	logFilter := NewLogFilter()
	logFilter.MinLevel = logutils.LogLevel(strings.ToUpper(LogLevel))
	logFilter.Writer = writer
	if !ValidateLevelFilter(logFilter.MinLevel, logFilter) {
		levels := make([]string, 0, len(logFilter.Levels))
		for _, level := range logFilter.Levels {
			levels = append(levels, string(level))
		}
		return &logutils.LevelFilter{}, fmt.Errorf("invalid log level %q, valid log levels are %s",
			LogLevel, strings.Join(levels, ", "))
	}

	return logFilter, nil
}
