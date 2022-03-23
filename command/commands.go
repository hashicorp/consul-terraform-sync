package command

import (
	"os"

	"github.com/mitchellh/cli"
)

const (
	errCreatingRequest = "Error: unable to create request"
	errCreatingClient  = "Error: unable to create client"
)

func configureMeta() meta {
	return meta{
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
}

// Commands returns the mapping of CLI commands for CTS. The meta
// parameter lets you set meta options for all commands.
func Commands() map[string]cli.CommandFactory {
	// Disable logging, we want to control what is output
	m := configureMeta()

	// The command factory will use the run command as the default
	// an empty string key ("") is interpreted as the default command
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
		cmdTaskCreateName: func() (cli.Command, error) {
			return newTaskCreateCommand(m), nil
		},
		cmdStartName: func() (cli.Command, error) {
			return newStartCommand(m, false), nil
		},
		"": func() (cli.Command, error) {
			return newStartCommand(m, true), nil
		},
	}

	return all
}
