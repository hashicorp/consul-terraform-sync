package command

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/controller"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/version"
	"github.com/mitchellh/go-wordwrap"
	"github.com/posener/complete"
)

const (
	cmdRunName = "run"

	flagConfigDir             = "config-dir"
	flagConfigFiles           = "config-file"
	flagInspect               = "inspect"
	flagInspectTask           = "inspect-task"
	flagOnce                  = "once"
	flagAutocompleteInstall   = "autocomplete-install"
	flagAutocompleteUninstall = "autocomplete-uninstall"
	flagClientType            = "client-type"
)

// runCommand handles the `run` command
type runCommand struct {
	meta
	flags *flag.FlagSet

	configFiles  *config.FlagAppendSliceValue
	inspectTasks *config.FlagAppendSliceValue

	isVersion *bool

	isInspect             *bool
	isOnce                *bool
	autocompleteInstall   *bool
	autocompleteUninstall *bool

	isDefault bool

	clientType *string

	help *bool
}

func (c *runCommand) runFlags() *flag.FlagSet {
	flags := flag.NewFlagSet(cmdRunName, flag.ContinueOnError)

	var configFiles, inspectTasks config.FlagAppendSliceValue
	var isInspect, isOnce, autocompleteInstall, autocompleteUninstall bool
	var clientType string

	// Parse the flags
	flags.Var(&configFiles, flagConfigDir, "A directory to load files for configuring Sync."+
		"\n\t\tConfiguration files require an .hcl or .json "+
		"\n\t\tfile extention in order to specify their format. This option can be "+
		"\n\t\tspecified multiple times to load different directories.")
	flags.Var(&configFiles, flagConfigFiles, "A file to load for configuring Sync. "+
		"\n\t\tConfiguration file requires an .hcl or .json extension in order to "+
		"\n\t\tspecify their format. This option can be specified multiple times to "+
		"\n\t\tload different configuration files.")
	c.configFiles = &configFiles

	flags.BoolVar(&isInspect, flagInspect, false,
		"Run Sync in Inspect mode to print the proposed state changes for all tasks, "+
			"\n\t\tand then exits. No changes "+
			"\n\t\tare applied in this mode.")
	c.isInspect = &isInspect

	flags.Var(&inspectTasks, flagInspectTask, "Run Sync in Inspect mode to "+
		"\n\t\tprint the proposed state changes for the task, and then exits. No "+
		"\n\t\tchanges are applied in this mode.")
	c.inspectTasks = &inspectTasks

	flags.BoolVar(&isOnce, flagOnce, false, "Render templates and run tasks once. "+
		"\n\t\tDoes not run the process as a daemon and disables buffer periods.")
	c.isOnce = &isOnce

	// Flags for installing the shell autocomplete
	flags.BoolVar(&autocompleteInstall, flagAutocompleteInstall, false, "Install the autocomplete")
	c.autocompleteInstall = &autocompleteInstall
	flags.BoolVar(&autocompleteUninstall, flagAutocompleteUninstall, false, "Uninstall the autocomplete")
	c.autocompleteUninstall = &autocompleteUninstall

	// Development only flags. Not printed with -h, -help
	flags.StringVar(&clientType, flagClientType, "", "Use only when developing"+
		"\n\t\tconsul-terraform-sync binary. Defaults to Terraform client if empty or"+
		"\n\t\tunknown value. Values can also be 'development' or 'test'.")
	c.clientType = &clientType

	return flags
}

func newRunCommand(m meta, isDefault bool) *runCommand {
	c := &runCommand{
		meta: m,
	}
	f := c.runFlags()
	c.meta.flags = f
	c.flags = f
	c.isDefault = isDefault
	return c
}

// Name returns the subcommand
func (c runCommand) Name() string {
	return cmdRunName
}

// Help returns the command's usage, list of flags, and examples
func (c *runCommand) Help() string {
	return helpFunc(nil, "Usage CLI: consul-terraform-sync run [-help] [options]\n")
}

// HelpDefault returns the usage when this command is used as the default command,
// without explicitly selecting the `Run` command
func (c *runCommand) HelpDefault() string {

	// Create a command factor for common commands
	commands := make(map[string]string)
	for _, v := range commonCommands {
		commands[v] = fmt.Sprintf("%s\t\n", v)
	}
	return helpFunc(commands, "Usage CLI: consul-terraform-sync <command> [-help] [options]\n")
}

// Synopsis is a short one-line synopsis of the command
// For base commands don't provide a synopsis
func (c *runCommand) Synopsis() string {
	return ""
}

// AutocompleteFlags returns a mapping of supported flags and autocomplete
// options for this command. The map key for the Flags map should be the
// complete flag such as "-foo" or "--foo".
func (c *runCommand) AutocompleteFlags() complete.Flags {
	return complete.Flags{
		fmt.Sprintf("-%s", flagConfigDir): complete.PredictDirs("*"),
		fmt.Sprintf("-%s", flagConfigFiles): complete.PredictOr(
			complete.PredictFiles("*.hcl"),
			complete.PredictFiles("*.json"),
		),
		fmt.Sprintf("-%s", flagInspect):               complete.PredictNothing,
		fmt.Sprintf("-%s", flagInspectTask):           complete.PredictNothing,
		fmt.Sprintf("-%s", flagOnce):                  complete.PredictNothing,
		fmt.Sprintf("-%s", flagAutocompleteInstall):   complete.PredictNothing,
		fmt.Sprintf("-%s", flagAutocompleteUninstall): complete.PredictNothing,
		fmt.Sprintf("-%s", flagClientType):            complete.PredictNothing,
	}
}

// AutocompleteArgs returns the argument predictorClient for this command.
// Since argument completion is not supported, this returns
// complete.PredictNothing.
func (c *runCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

// Run runs the command
func (c *runCommand) Run(args []string) int {
	c.flags.Usage = func() { c.meta.UI.Output(c.Help()) }
	if err := c.flags.Parse(args); err != nil {
		return ExitCodeParseFlagsError
	}

	// If is default, pre
	if len(*c.configFiles) == 0 && c.isDefault {
		c.UI.Output(c.HelpDefault())
		return ExitCodeRequiredFlagsError
	} else if len(*c.configFiles) == 0 {
		c.UI.Error("unable to start consul-terraform-sync")
		c.UI.Output("no config file provided")
		help := fmt.Sprintf("For additional help try 'consul-terraform-sync %s --help'",
			cmdRunName)
		help = wordwrap.WrapString(help, width)

		c.UI.Output(help)

		return ExitCodeRequiredFlagsError
	}

	// Build the config.
	conf, err := config.BuildConfig(*c.configFiles)
	logger := logging.Global().Named(logSystemName)
	if err != nil {
		logger.Error("error building configuration", "error", err)
		os.Exit(ExitCodeConfigError)
	}
	conf.Finalize()

	if err := conf.Validate(); err != nil {
		logger.Error("error validating configuration", "error", err)
		os.Exit(ExitCodeConfigError)
	}

	if err := logging.Setup(&logging.Config{
		Level:          config.StringVal(conf.LogLevel),
		Syslog:         config.BoolVal(conf.Syslog.Enabled),
		SyslogFacility: config.StringVal(conf.Syslog.Facility),
		SyslogName:     config.StringVal(conf.Syslog.Name),
		Writer:         os.Stderr,
	}); err != nil {
		logger.Error("error setting up logging", "error", err)
		return ExitCodeConfigError
	}

	// Print information on startup for debugging
	logger.Info(version.GetHumanVersion())
	logger.Debug("configuration", "config", conf.GoString())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(*c.inspectTasks) != 0 {
		*c.isInspect = true
		conf.Tasks, err = config.FilterTasks(conf.Tasks, *c.inspectTasks)
		if err != nil {
			logger.Error("error inspecting tasks", "error", err)
			return ExitCodeConfigError
		}
	}

	switch {
	case *c.isInspect:
		logger.Info("running controller in inspect mode")
	case *c.isOnce:
		logger.Info("running controller in once mode")
	default:
		logger.Info("running controller in daemon mode")
	}

	// Set up controller
	conf.ClientType = config.String(*c.clientType)
	var ctrl controller.Controller
	if *c.isInspect {
		logger.Debug("inspect mode enabled, processing then exiting")
		logger.Info("setting up controller", "type", "readonly")
		ctrl, err = controller.NewReadOnly(conf)
	} else {
		logger.Info("setting up controller", "type", "readwrite")
		ctrl, err = controller.NewReadWrite(conf)
	}
	if err != nil {
		logger.Error("error setting up controller", "error", err)
		return ExitCodeConfigError
	}
	defer ctrl.Stop()

	// Install the driver after controller has tested Consul connection
	if err := controller.InstallDriver(ctx, conf); err != nil {
		logger.Error("error installing driver", "error", err)
		return ExitCodeDriverError
	}

	errCh := make(chan error, 1)
	exitBufLen := 2 // exit api & controller
	exitCh := make(chan struct{}, exitBufLen)

	go func() {
		logger.Info("initializing controller")
		err := ctrl.Init(ctx)
		if err != nil {
			if err == context.Canceled {
				exitCh <- struct{}{}
				return
			}
			logger.Error("error initializing controller", "error", err)
			errCh <- err
			return
		}

		switch ctrl := ctrl.(type) {
		case controller.Oncer:
			if err := ctrl.Once(ctx); err != nil {
				if err == context.Canceled {
					exitCh <- struct{}{}
				} else {
					logger.Error("error running controller in Once mode", "error", err)
					errCh <- err
				}
				return
			}
			if *c.isOnce {
				logger.Info("controller in Once mode has completed")
				exitCh <- struct{}{}
				return
			}
		}

		go func() {
			if *c.isInspect {
				return
			}
			s, err := api.NewAPI(api.Config{
				Controller: ctrl.(api.Server),
				Port:       config.IntVal(conf.Port),
				TLS:        conf.TLS,
			})
			if err != nil {
				errCh <- err
				return
			}
			err = s.Serve(ctx)
			if err != nil && err != context.Canceled {
				errCh <- err
				return
			}
			exitCh <- struct{}{}
		}()

		if err := ctrl.Run(ctx); err != nil {
			if err == context.Canceled {
				exitCh <- struct{}{}
			} else {
				logger.Error("error running controller", "error", err)
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
			logger.Info("signal received to initiate graceful shutdown", "signal", sig)
			cancel()
			counter := 0
			start := time.Now()
			for {
				since := time.Since(start)
				select {
				case <-exitCh:
					counter++
					if counter >= exitBufLen {
						logger.Info("graceful shutdown")
						return ExitCodeOK
					}
				case <-time.After(10*time.Second - since):
					logger.Info("graceful shutdown timed out, exiting")
					return ExitCodeInterrupt
				}
			}

		case <-exitCh:
			if *c.isOnce || *c.isInspect {
				logger.Info("graceful shutdown")
				return ExitCodeOK
			}
			logger.Warn("unexpected shutdown")
			return ExitCodeError

		case <-errCh:
			return ExitCodeError
		}
	}
}
