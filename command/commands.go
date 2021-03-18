package command

import (
	"os"

	"github.com/mitchellh/cli"
)

// Commands returns the mapping of CLI commands for CTS. The meta
// parameter lets you set meta options for all commands.
func Commands() map[string]cli.CommandFactory {
	m := meta{
		UI: &cli.PrefixedUi{
			InfoPrefix:   "==> ",
			OutputPrefix: "    ",
			ErrorPrefix:  "==> ",
			Ui: &cli.BasicUi{
				Writer: os.Stdout,
				Reader: os.Stdin,
			},
		},
	}

	all := map[string]cli.CommandFactory{
		"task disable": func() (cli.Command, error) {
			return newTaskDisableCommand(m), nil
		},
		"task enable": func() (cli.Command, error) {
			return newTaskEnableCommand(m), nil
		},
	}

	return all
}
