package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/mitchellh/go-wordwrap"
)

// TaskDisableCommand handles the `task disable` command
type taskDisableCommand struct {
	meta
}

// Name returns the subcommand
func (c *taskDisableCommand) Name() string {
	return "task disable"
}

// Help returns the command's usage, list of flags, and examples
func (c *taskDisableCommand) Help() string {
	helpText := `
Usage: consul-terraform-sync task disable [options] <task name>

  Task Disable is used to disable existing tasks. Once disabled, a task will no
  longer run and make changes to your network infrastructure resources.

Options:
  No options are currently available

Example:

  $ consul-terraform-sync task disable my_task
    ==> Waiting to disable 'Test_2'...

    ==> 'Test_2' disable complete!
`
	return strings.TrimSpace(helpText)
}

// Synopsis is a short one-line synopsis of the command
func (c *taskDisableCommand) Synopsis() string {
	return "Disables existing tasks from running."
}

// Run runs the command
func (c *taskDisableCommand) Run(args []string) int {
	flags := c.meta.defaultFlagSet(c, args)

	if err := flags.Parse(args); err != nil {
		return ExitCodeParseFlagsError
	}

	args = flags.Args()
	if ok := c.meta.oneArgCheck(c, args); !ok {
		return ExitCodeRequiredFlagsError
	}

	taskName := args[0]

	c.UI.Info(fmt.Sprintf("Waiting to disable '%s'...", taskName))
	c.UI.Output("")

	client := c.meta.client()
	_, err := client.Task().Update(taskName, api.UpdateTaskConfig{
		Enabled: config.Bool(false),
	}, nil)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error: unable to disable '%s'", taskName))

		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}

	c.UI.Info(fmt.Sprintf("'%s' disable complete!", taskName))

	return ExitCodeOK
}
