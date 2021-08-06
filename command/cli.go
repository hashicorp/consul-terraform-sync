package command

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/controller"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/version"
	mcli "github.com/mitchellh/cli"
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
	ExitCodeDriverError
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
	var configFiles, inspectTasks config.FlagAppendSliceValue
	var isVersion, isInspect, isOnce bool
	var clientType string
	var help, h bool

	// Parse the flags
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	f.Var(&configFiles, "config-dir", "A directory to load files for "+
		"configuring Sync. Configuration files require an .hcl or .json "+
		"file extention in order to specify their format. This option can be "+
		"specified multiple times to load different directories.")
	f.Var(&configFiles, "config-file", "A file to load for configuring Sync. "+
		"Configuration file requires an .hcl or .json extension in order to "+
		"specify their format. This option can be specified multiple times to "+
		"load different configuration files.")
	f.BoolVar(&isInspect, "inspect", false, "Run Sync in Inspect mode to "+
		"print the proposed state changes for all tasks, and then exits. No changes "+
		"are applied in this mode.")
	f.Var(&inspectTasks, "inspect-task", "Run Sync in Inspect mode to "+
		"print the proposed state changes for the task, and then exits. No "+
		"changes are applied in this mode.")
	f.BoolVar(&isOnce, "once", false, "Render templates and run tasks once. "+
		"Does not run the process as a daemon and disables buffer periods.")
	f.BoolVar(&isVersion, "version", false, "Print the version of this daemon.")

	// Setup help flags for custom output
	f.BoolVar(&help, "help", false, "Print the flag options and descriptions.")
	f.BoolVar(&h, "h", false, "Print the flag options and descriptions.")

	// Development only flags. Not printed with -h, -help
	f.StringVar(&clientType, "client-type", "", "Use only when developing"+
		"consul-terraform-sync binary. Defaults to Terraform client if empty or"+
		"unknown value. Values can also be 'development' or 'test'.")

	err := f.Parse(args[1:])
	if err != nil {
		return ExitCodeParseFlagsError
	}

	if isVersion {
		fmt.Fprintf(cli.outStream, "%s %s\n", version.Name, version.GetHumanVersion())
		fmt.Fprintf(cli.outStream, "Compatible with Terraform %s\n", version.CompatibleTerraformVersionConstraint)
		return ExitCodeOK
	}

	// Are we running the binary or a CLI command?
	// If the first unused argument isn't a flag, then assume subcommand
	unused := f.Args()
	if len(unused) > 0 && !strings.HasPrefix(unused[0], "-") {
		subcommands := &mcli.CLI{
			Name:     "consul-terraform-sync",
			Args:     args[1:],
			Commands: Commands(),
		}

		exitCode, err := subcommands.Run()
		if err != nil {
			fmt.Fprintf(cli.errStream, "Error running the CLI command '%s': %s",
				strings.Join(args, " "), err)
		}
		return exitCode
	}

	// running the binary!

	// Print out binary's help info
	if help || h {
		fmt.Fprintf(cli.outStream, "Usage of %s:\n", args[0])
		printFlags(f)
		return ExitCodeOK
	}

	// Validate required flags
	if len(configFiles) == 0 {
		fmt.Fprintf(cli.errStream, "Error: config file(s) required, use --config-dir or --config-file flag options")
		printFlags(f)
		return ExitCodeRequiredFlagsError
	}
	return cli.runBinary(configFiles, inspectTasks, isInspect, isOnce, clientType)
}

func (cli *CLI) runBinary(configFiles, inspectTasks config.FlagAppendSliceValue,
	isInspect, isOnce bool, clientType string) int {

	// Build the config.
	conf, err := config.BuildConfig([]string(configFiles))
	if err != nil {
		log.Printf("[ERR] (cli) error building configuration: %s", err)
		os.Exit(ExitCodeConfigError)
	}
	conf.Finalize()

	if err := conf.Validate(); err != nil {
		log.Printf("[ERR] (cli) error validating configuration: %s", err)
		os.Exit(ExitCodeConfigError)
	}

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
	log.Printf("[DEBUG] %s", conf.GoString())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(inspectTasks) != 0 {
		isInspect = true
		conf.Tasks, err = config.FilterTasks(conf.Tasks, inspectTasks)
		if err != nil {
			log.Printf("[ERR] (cli) error inspecting tasks: %s", err)
			return ExitCodeConfigError
		}
	}

	switch {
	case isInspect:
		log.Printf("[INFO] (cli) running controller in inspect mode")
	case isOnce:
		log.Printf("[INFO] (cli) running controller in once mode")
	default:
		log.Printf("[INFO] (cli) running controller in daemon mode")
	}

	// Set up controller
	conf.ClientType = config.String(clientType)
	var ctrl controller.Controller
	if isInspect {
		log.Printf("[DEBUG] (cli) inspect mode enabled, processing then exiting")
		log.Printf("[INFO] (cli) setting up controller: readonly")
		ctrl, err = controller.NewReadOnly(conf)
	} else {
		log.Printf("[INFO] (cli) setting up controller: readwrite")
		ctrl, err = controller.NewReadWrite(conf)
	}
	if err != nil {
		log.Printf("[ERR] (cli) error setting up controller: %s", err)
		return ExitCodeConfigError
	}
	defer ctrl.Stop()

	// Install the driver after controller has tested Consul connection
	if err := controller.InstallDriver(ctx, conf); err != nil {
		log.Printf("[ERR] (cli) error installing driver: %s", err)
		return ExitCodeDriverError
	}

	errCh := make(chan error, 1)
	exitBufLen := 2 // exit api & controller
	exitCh := make(chan struct{}, exitBufLen)

	go func() {
		log.Printf("[INFO] (cli) initializing controller")
		err := ctrl.Init(ctx)
		if err != nil {
			if err == context.Canceled {
				exitCh <- struct{}{}
				return
			}
			log.Printf("[ERR] (cli) error initializing controller: %s", err)
			errCh <- err
			return
		}

		switch c := ctrl.(type) {
		case controller.Oncer:
			if err := c.Once(ctx); err != nil {
				if err == context.Canceled {
					exitCh <- struct{}{}
				} else {
					log.Printf("[ERR] (cli) error running controller in Once mode: %s", err)
					errCh <- err
				}
				return
			}
			if isOnce {
				log.Printf("[INFO] (cli) controller in Once mode has completed")
				exitCh <- struct{}{}
				return
			}
		}

		go func() {
			if isInspect {
				return
			}
			if err = ctrl.ServeAPI(ctx); err != nil {
				if err == context.Canceled {
					exitCh <- struct{}{}
					return
				}
				errCh <- err
				return
			}
		}()

		if err := ctrl.Run(ctx); err != nil {
			if err == context.Canceled {
				exitCh <- struct{}{}
			} else {
				log.Printf("[ERR] (cli) error running controller: %s", err)
				errCh <- err
			}
			return
		}

		exitCh <- struct{}{}
	}()

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	for {
		select {
		case sig := <-interruptCh:
			// Cancel the context and wait for controller go routine to gracefully
			// shutdown
			log.Printf("[INFO] (cli) signal received to initiate graceful shutdown: %v", sig)
			cancel()
			counter := 0
			start := time.Now()
			for {
				since := time.Since(start)
				select {
				case <-exitCh:
					counter++
					if counter >= exitBufLen {
						log.Printf("[INFO] (cli) graceful shutdown")
						return ExitCodeOK
					}
				case <-time.After(10*time.Second - since):
					log.Printf("[INFO] (cli) graceful shutdown timed out, exiting")
					return ExitCodeInterrupt
				}
			}

		case <-exitCh:
			if isOnce || isInspect {
				log.Printf("[INFO] (cli) graceful shutdown")
				return ExitCodeOK
			}
			log.Printf("[WARN] (cli) unexpected shutdown")
			return ExitCodeError

		case <-errCh:
			return ExitCodeError
		}
	}
}

// printFlags prints out select flags
func printFlags(f *flag.FlagSet) {
	f.VisitAll(func(f *flag.Flag) {
		switch f.Name {
		case "h", "help":
			// don't print out help flags
			return
		case "client-type":
			// don't print out development-only flags
			return
		}
		fmt.Printf("  -%s %s\n", f.Name, f.Value)
		fmt.Printf("        %s\n", f.Usage)
	})
}
