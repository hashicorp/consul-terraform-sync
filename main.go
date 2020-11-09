package main

import (
	"os"

	"github.com/hashicorp/consul-terraform-sync/event"
)

func main() {
	store := event.NewStore()
	cli := NewCLI(os.Stdout, os.Stderr, store)
	os.Exit(cli.Run(os.Args))
}
