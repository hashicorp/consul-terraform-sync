package command

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/go-wordwrap"
)

// terminal width. use for word-wrapping
const width = uint(78)

// meta contains the meta-options and functionality for all CTS commands
type meta struct {
	UI cli.Ui

	helpOptions []string
	port        *int
}

func (m *meta) defaultFlagSet(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	m.port = flags.Int("port", config.DefaultPort, "The port to use for the Consul Terraform Sync API server")
	flags.SetOutput(ioutil.Discard)

	flags.VisitAll(func(f *flag.Flag) {
		option := fmt.Sprintf("  %s %s\n    %s\n", f.Name, f.Value, f.Usage)
		m.helpOptions = append(m.helpOptions, option)
	})
	if len(m.helpOptions) == 0 {
		m.helpOptions = append(m.helpOptions, "No options are currently available")
	}

	return flags
}

func (m *meta) setFlagsUsage(flags *flag.FlagSet, args []string, help string) {
	flags.Usage = func() {
		m.UI.Error(fmt.Sprintf("Error: unsupported arguments in flags '%s'",
			strings.Join(args, ", ")))
		m.UI.Output(fmt.Sprintf("Please see --help information below for "+
			"supported options:\n\n%s", help))
	}
}

func (m *meta) oneArgCheck(name string, args []string) bool {
	numArgs := len(args)
	if numArgs == 1 {
		return true
	}

	m.UI.Error("Error: this command requires one argument: [options] <task name>")
	if numArgs == 0 {
		m.UI.Output("No arguments were passed to the command")
	} else {
		m.UI.Output(fmt.Sprintf("%d arguments were passed to the command: '%s'",
			numArgs, strings.Join(args, ", ")))
		m.UI.Output("All flags are required to appear before positional arguments if set\n")
	}

	help := fmt.Sprintf("For additional help try 'consul-terraform-sync %s --help'",
		name)
	help = wordwrap.WrapString(help, width)

	m.UI.Output(help)
	return false
}

func (m *meta) client() *api.Client {
	return api.NewClient(&api.ClientConfig{
		Port: *m.port,
	}, nil)
}

// requestUserApproval returns an exit code and boolean describing if the user
// approved. If the user did not approve (false is returned), exit code is provided.
func (m *meta) requestUserApproval(taskName string) (int, bool) {
	m.UI.Info("Enabling the task will perform the actions described above.")
	m.UI.Output(fmt.Sprintf("Do you want to perform these actions for '%s'?", taskName))
	m.UI.Output(" - This action cannot be undone.")
	m.UI.Output(" - CTS cannot guarantee Terraform will perform these exact actions if")
	m.UI.Output("   monitored services have changed.\n")
	m.UI.Output("Only 'yes' will be accepted to approve.\n")
	v, err := m.UI.Ask(fmt.Sprintf("Enter a value:"))
	m.UI.Output("")

	if err != nil {
		m.UI.Error(fmt.Sprintf("Error asking for approval: %s", err))
		return ExitCodeError, false
	}
	if v != "yes" {
		m.UI.Output(fmt.Sprintf("Cancelled enabling task '%s'", taskName))
		return ExitCodeOK, false
	}

	return 0, true
}

// changesDetected attempts to detect if a change plan has been generated.
// Does a few string parse checks to try to prevent a false negative
func (m *meta) changesDetected(plan string) bool {
	plan = strings.TrimSpace(plan)

	// look for phrases that there are changes
	if strings.HasPrefix(plan, "An execution plan has been generated") {
		return true
	}
	if strings.HasPrefix(plan, "CTS will perform the following actions:") {
		return true
	}

	// look for phrases that there are no changes
	return !strings.Contains(plan, "No changes. Infrastructure is up-to-date.")
}
