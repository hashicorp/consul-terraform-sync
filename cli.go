package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/hashicorp/consul-nia/config"
	"github.com/hashicorp/consul-nia/logging"
	"github.com/hashicorp/consul-nia/version"
)

// Exit codes are int values that represent an exit code for a particular error.
// Sub-systems may check this unique error to determine the cause of an error
// without parsing the output or help text.
//
// Errors start at 10
const (
	ExitCodeOK int = 0

	ExitCodeError = 10 + iota
	ExitCodeInterrupt
	ExitCodeRequiredFlagsError
	ExitCodeParseFlagsError
	ExitCodeConfigError
)

// CLI is the main entry point.
type CLI struct {
	sync.Mutex

	// outSteam and errStream are the standard out and standard error streams to
	// write messages from the CLI.
	outStream, errStream io.Writer

	// signalCh is the channel where the cli receives signals.
	signalCh chan os.Signal

	// stopCh is an internal channel used to trigger a shutdown of the CLI.
	stopCh  chan struct{}
	stopped bool
}

// NewCLI creates a new CLI object with the given stdout and stderr streams.
func NewCLI(out, err io.Writer) *CLI {
	return &CLI{
		outStream: out,
		errStream: err,
		signalCh:  make(chan os.Signal, 1),
		stopCh:    make(chan struct{}),
	}
}

// Run accepts a slice of arguments and returns an int representing the exit
// status from the command.
func (cli *CLI) Run(args []string) int {
	// Handle parsing the CLI flags.
	var configFiles config.FlagAppendSliceValue
	var isVersion, isInspect bool

	// Parse the flags
	f := flag.NewFlagSet("", flag.ContinueOnError)
	f.Var(&configFiles, "config-file", "A config file to use. Can be either "+
		".hcl or .json format. Can be specified multiple times.")
	f.Var(&configFiles, "config-dir", "A directory to look for .hcl or .json "+
		"config files in. Can be specified multiple times.")
	f.BoolVar(&isVersion, "version", false, "Print the version of this daemon.")
	f.BoolVar(&isInspect, "inspect", false, "Print the current and proposed "+
		"state change, and then exits.")

	err := f.Parse(os.Args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			return ExitCodeOK
		}
		return ExitCodeParseFlagsError
	}

	// Validate required flags
	if !isVersion && len(configFiles) == 0 {
		log.Printf("[ERR] config file(s) required, use --config-file or --config-dir flag options")
		return ExitCodeRequiredFlagsError
	}

	// Build the config.
	conf, err := config.BuildConfig([]string(configFiles))
	if err != nil {
		log.Printf("[ERR] (cli) error building configuration: %s", err)
		os.Exit(ExitCodeConfigError)
	}
	conf.Finalize()

	if err := logging.Setup(&logging.Config{
		Level:          config.StringVal(conf.LogLevel),
		Syslog:         config.BoolVal(conf.Syslog.Enabled),
		SyslogFacility: config.StringVal(conf.Syslog.Facility),
		SyslogName:     config.StringVal(conf.Syslog.Name),
		Writer:         cli.errStream,
	}); err != nil {
		log.Printf("[ERR] (cli) error setting up logging: %s", err)
		return ExitCodeConfigError
	}

	// Print information on startup for debugging
	log.Printf("[INFO] %s", version.GetHumanVersion())
	log.Printf("[INFO] %s", conf.GoString())

	// If the version was requested, return an "error" containing the version
	// information. This might sound weird, but most *nix applications actually
	// print their version on stderr anyway.
	if isVersion {
		log.Printf("[DEBUG] (cli) version flag was given, exiting now")
		fmt.Fprintf(cli.errStream, "%s %s\n", version.Name, version.GetHumanVersion())
		return ExitCodeOK
	}

	if isInspect || *conf.InspectMode {
		log.Printf("[DEBUG] (cli) inspect mode enabled, processing then exiting")
		fmt.Fprintln(cli.outStream, "TODO")
		return ExitCodeOK
	}

	return ExitCodeOK
}
