// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"io"
	"os"

	"github.com/mitchellh/cli"
)

const (
	errCreatingRequest = "Error: unable to create request"
	errCreatingClient  = "Error: unable to create client"
)

func configureMeta(writer io.Writer, errorWriter io.Writer) meta {
	return meta{
		UI: &cli.PrefixedUi{
			InfoPrefix:   "==> ",
			OutputPrefix: "    ",
			ErrorPrefix:  "==> ",
			Ui: &cli.BasicUi{
				Writer:      writer,
				Reader:      os.Stdin,
				ErrorWriter: errorWriter,
			},
		},
		writer: writer,
	}
}

// Commands returns the mapping of CLI commands for CTS. The meta
// parameter lets you set meta options for all commands.
func Commands(writer io.Writer, errorWriter io.Writer) map[string]cli.CommandFactory {
	m := configureMeta(writer, errorWriter)

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
			return newStartCommand(m), nil
		},
	}

	return all
}
