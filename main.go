// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"os"

	"github.com/hashicorp/consul-terraform-sync/command"
)

func main() {
	cli := command.NewCLI(os.Stdout, os.Stderr)
	os.Exit(cli.Run(os.Args))
}
