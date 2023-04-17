// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCLI_Run(t *testing.T) {

	cases := []struct {
		name           string
		args           []string
		outputContains []string
	}{
		{
			name: "version with single dash",
			args: []string{"ignore", "-version"},
			outputContains: []string{
				"consul-terraform-sync",
				"Compatible with Terraform",
			},
		},
		{
			name: "version with two dashes",
			args: []string{"ignore", "--version"},
			outputContains: []string{
				"consul-terraform-sync",
				"Compatible with Terraform",
			},
		},
		{
			name: "version with v",
			args: []string{"ignore", "-v"},
			outputContains: []string{
				"consul-terraform-sync",
				"Compatible with Terraform",
			},
		},
		{
			name: "no arguments help",
			args: []string{"ignore"},
			outputContains: []string{
				"Usage CLI: consul-terraform-sync <command> [-help] [options]",
				"Commands:",
			},
		},
		{
			name: "help with single dash",
			args: []string{"ignore", "-help"},
			outputContains: []string{
				"Usage CLI: consul-terraform-sync <command> [-help] [options]",
				"Commands:",
			},
		},
		{
			name: "help with two dashes",
			args: []string{"ignore", "--help"},
			outputContains: []string{
				"Usage CLI: consul-terraform-sync <command> [-help] [options]",
				"Commands:",
			},
		},
		{
			name: "version with h",
			args: []string{"ignore", "-h"},
			outputContains: []string{
				"Usage CLI: consul-terraform-sync <command> [-help] [options]",
				"Commands:",
			},
		},
		{
			name: "command task with help",
			args: []string{"ignore", "task", "-h"},
			outputContains: []string{
				"This command is accessed by using one of the subcommands below.",
			},
		},
		{
			name: "command start with help",
			args: []string{"ignore", "start", "-h"},
			outputContains: []string{
				"Usage CLI: consul-terraform-sync start [-help] [options]",
			},
		},
		{
			name: "command with version",
			args: []string{"ignore", "start", "-version"},
			outputContains: []string{
				"flag provided but not defined: -version",
				"Usage CLI: consul-terraform-sync start [-help] [options]",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var bufOut bytes.Buffer
			var bufErr bytes.Buffer
			cli := NewCLI(&bufOut, &bufErr)
			cli.Run(tc.args)
			assert.Empty(t, bufErr)
			for _, expect := range tc.outputContains {
				assert.Contains(t, bufOut.String(), expect)
			}
		})
	}
}
