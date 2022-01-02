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
		cmdTaskDisableName: func() (cli.Command, error) {
			return newTaskDisableCommand(m), nil
		},
		cmdTaskEnableName: func() (cli.Command, error) {
			return newTaskEnableCommand(m), nil
		},
		cmdTaskDeleteName: func() (cli.Command, error) {
			return newTaskDeleteCommand(m), nil
		},
	}

	return all
}
