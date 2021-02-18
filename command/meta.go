package command

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/go-wordwrap"
)

// terminal width. use for word-wrapping
const width = uint(78)

// meta contains the meta-options and functionality for all CTS commands
type meta struct {
	UI cli.Ui
}

// nameHelpCommand is a interface to retrieve a command's name and help text
type nameHelpCommand interface {
	Name() string
	Help() string
}

func (m *meta) defaultFlagSet(c nameHelpCommand, args []string) *flag.FlagSet {
	flags := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	flags.SetOutput(ioutil.Discard)
	flags.Usage = func() {
		m.UI.Error(fmt.Sprintf("Error: unsupported arguments in flags '%s'",
			strings.Join(args, ", ")))
		m.UI.Output(fmt.Sprintf("Please see --help information below for "+
			"supported options:\n\n%s", c.Help()))
	}

	return flags
}

func (m *meta) oneArgCheck(c nameHelpCommand, args []string) bool {
	if len(args) == 1 {
		return true
	}

	m.UI.Error("Error: this command requires one argument: <task name>")
	if len(args) == 0 {
		m.UI.Output("No arguments were passed to the command")
	} else {
		m.UI.Output(fmt.Sprintf("%d arguments were passed to the command: '%s'",
			len(args), strings.Join(args, ", ")))
	}

	help := fmt.Sprintf("For additional help try 'consul-terraform-sync %s --help'",
		c.Name())
	help = wordwrap.WrapString(help, width)

	m.UI.Output(help)
	return false
}

func (m *meta) client() *api.Client {
	return api.NewClient(&api.ClientConfig{
		// TODO: add support for configuring port when doing general options
		Port: 8558,
	}, nil)
}
