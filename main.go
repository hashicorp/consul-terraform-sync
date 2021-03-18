package main

import (
	"os"

	"github.com/hashicorp/consul-terraform-sync/command"
)

func main() {
	cli := command.NewCLI(os.Stdout, os.Stderr)
	os.Exit(cli.Run(os.Args))
}
