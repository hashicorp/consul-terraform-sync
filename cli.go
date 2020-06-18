package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

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
	var isVersion, isInspect bool

	// Parse the flags
	f := flag.NewFlagSet("", flag.ContinueOnError)
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

	// Print version information for debugging
	log.Printf("[INFO] %s", version.GetHumanVersion())

	// If the version was requested, return an "error" containing the version
	// information. This might sound weird, but most *nix applications actually
	// print their version on stderr anyway.
	if isVersion {
		log.Printf("[DEBUG] (cli) version flag was given, exiting now")
		fmt.Fprintf(cli.errStream, "%s %s\n", version.Name, version.GetHumanVersion())
		return ExitCodeOK
	}

	if isInspect {
		log.Printf("[DEBUG] (cli) inspect flag was given, processing then exiting")
		fmt.Fprintln(cli.errStream, "TODO")
		return ExitCodeOK
	}

	return ExitCodeOK
}
