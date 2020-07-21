package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/hashicorp/consul-nia/config"
	"github.com/hashicorp/consul-nia/driver"
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
	var configFiles config.FlagAppendSliceValue
	var isVersion, isInspect bool

	// Parse the flags
	f := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
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
		log.Printf("[ERR] config file(s) required, use --config-dir or --config-file flag options")
		f.PrintDefaults()
		return ExitCodeRequiredFlagsError
	}

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

	if isVersion {
		log.Printf("[DEBUG] (cli) version flag was given, exiting now")
		fmt.Fprintf(cli.errStream, "%s %s\n", version.Name, version.GetHumanVersion())
		return ExitCodeOK
	}

	log.Printf("[INFO] (cli) setting up Terraform driver")
	driver := newTerraformDriver(conf)
	if err := driver.Init(); err != nil {
		log.Printf("[ERR] (cli) error initializing Terraform driver: %s", err)
		return ExitCodeDriverError
	}
	log.Printf("[INFO] (cli) Terraform driver initialized")

	// initialize tasks. this is hardcoded in main function for demo purposes
	// TODO: separate by provider instances using workspaces.
	// Future: improve by combining tasks into workflows.
	tasks := newDriverTasks(conf)
	for _, task := range tasks {
		err := driver.InitTask(task, true)
		if err != nil {
			// TODO: flush out error better
			log.Printf("[ERR] %s", err)
		}
	}

	if isInspect || *conf.InspectMode {
		log.Printf("[DEBUG] (cli) inspect mode enabled, processing then exiting")
		fmt.Fprintln(cli.outStream, "TODO")
		return ExitCodeOK
	}

	return ExitCodeOK
}

func newTerraformDriver(conf *config.Config) *driver.Terraform {
	tfConf := *conf.Driver.Terraform
	return driver.NewTerraform(&driver.TerraformConfig{
		LogLevel:   *tfConf.LogLevel,
		Path:       *tfConf.Path,
		DataDir:    *tfConf.DataDir,
		WorkingDir: *tfConf.WorkingDir,
		SkipVerify: *tfConf.SkipVerify,
		Backend:    tfConf.Backend,
	})
}

func newDriverTasks(conf *config.Config) []driver.Task {
	tasks := make([]driver.Task, len(*conf.Tasks))
	for i, t := range *conf.Tasks {

		services := make([]driver.Service, len(t.Services))
		for si, service := range t.Services {
			services[si] = getService(conf.Services, service)
		}

		providers := make([]map[string]interface{}, len(t.Providers))
		for pi, provider := range t.Providers {
			providers[pi] = getProvider(conf.Providers, provider)
		}

		tasks[i] = driver.Task{
			Description: *t.Description,
			Name:        *t.Name,
			Providers:   providers,
			Services:    services,
			Source:      *t.Source,
			Version:     *t.Version,
		}
	}

	return tasks
}

// getService is a helper to find and convert a user-defined service
// configuration by ID to a driver service type. If a service is not
// explicitly configured, it assumes the service is a logical service name
// in the default namespace.
func getService(services *config.ServiceConfigs, id string) driver.Service {
	for _, s := range *services {
		if *s.ID == id {
			return driver.Service{
				Description: *s.Description,
				Name:        *s.Name,
				Namespace:   *s.Namespace,
			}
		}
	}

	return driver.Service{Name: id}
}

func getProvider(providers *config.ProviderConfigs, name string) map[string]interface{} {
	for _, p := range *providers {
		if _, ok := (*p)[name]; ok {
			return *p
		}
	}

	return nil
}
