package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/mitchellh/go-wordwrap"
)

const cmdTaskDisableName = "task disable"

// TaskDisableCommand handles the `task disable` command
type taskDisableCommand struct {
	meta

	flags *flag.FlagSet
}

func newTaskDisableCommand(m meta) *taskDisableCommand {
	flags := m.defaultFlagSet(cmdTaskDisableName)
	return &taskDisableCommand{
		meta:  m,
		flags: flags,
	}
}

// Name returns the subcommand
func (c taskDisableCommand) Name() string {
	return cmdTaskDisableName
}

// Help returns the command's usage, list of flags, and examples
func (c *taskDisableCommand) Help() string {
	c.meta.setHelpOptions()
	helpText := fmt.Sprintf(`
Usage: consul-terraform-sync task disable [options] <task name>

  Task Disable is used to disable existing tasks. Once disabled, a task will no
  longer run and make changes to your network infrastructure resources.

Options:
%s

Example:

  $ consul-terraform-sync task disable my_task
    ==> Waiting to disable 'Test_2'...

    ==> 'Test_2' disable complete!
`, strings.Join(c.meta.helpOptions, "\n"))
	return strings.TrimSpace(helpText)
}

// Synopsis is a short one-line synopsis of the command
func (c *taskDisableCommand) Synopsis() string {
	return "Disables existing tasks from running."
}

// Run runs the command
func (c *taskDisableCommand) Run(args []string) int {
	c.meta.setFlagsUsage(c.flags, args, c.Help())

	if err := c.flags.Parse(args); err != nil {
		return ExitCodeParseFlagsError
	}

	args = c.flags.Args()
	if ok := c.meta.oneArgCheck(c.Name(), args); !ok {
		return ExitCodeRequiredFlagsError
	}

	taskName := args[0]

	c.UI.Info(fmt.Sprintf("Waiting to disable '%s'...", taskName))
	c.UI.Output("")

	client, err := c.meta.client()
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error: unable to create client for '%s'", taskName))
		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}

	_, err = client.Task().Update(taskName, api.UpdateTaskConfig{
		Enabled: config.Bool(false),
	}, nil)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error: unable to disable '%s'", taskName))
		err = processEOFError(client.Scheme(), err)

		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}

	c.UI.Info(fmt.Sprintf("'%s' disable complete!", taskName))

	return ExitCodeOK
}
