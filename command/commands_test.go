// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"io/ioutil"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/stretchr/testify/assert"
)

func Test_Commands(t *testing.T) {
	cf := Commands(ioutil.Discard, ioutil.Discard)

	// map of commands to synopsis
	expectedCommands := map[string]cli.Command{
		cmdTaskCreateName:  &taskCreateCommand{},
		cmdTaskEnableName:  &taskEnableCommand{},
		cmdTaskDisableName: &taskDisableCommand{},
		cmdTaskDeleteName:  &taskDeleteCommand{},
		cmdStartName:       &startCommand{},
	}

	assert.Equal(t, len(expectedCommands), len(cf))

	for k, v := range expectedCommands {
		cmds, ok := cf[k]
		assert.True(t, ok)

		c, err := cmds()
		assert.NoError(t, err)

		// Verify that the command type constructed is the correct type
		assert.IsType(t, v, c)
	}
}
