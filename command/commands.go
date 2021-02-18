package command

import (
	"os"

	"github.com/mitchellh/cli"
)

// Exit codes match the main cli's exit code
const (
	ExitCodeOK int = 0

	ExitCodeError = 10 + iota
	ExitCodeInterrupt
	ExitCodeRequiredFlagsError
	ExitCodeParseFlagsError
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
			return &taskDisableCommand{
				meta: m,
			}, nil
		},
	}

	return all
}
