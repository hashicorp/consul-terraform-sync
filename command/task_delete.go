package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/mitchellh/go-wordwrap"
)

const cmdTaskDeleteName = "task delete"

// TaskDeleteCommand handles the `task delete` command
type taskDeleteCommand struct {
	meta

	flags *flag.FlagSet
}

func newTaskDeleteCommand(m meta) *taskDeleteCommand {
	flags := m.defaultFlagSet(cmdTaskDeleteName)
	return &taskDeleteCommand{
		meta:  m,
		flags: flags,
	}
}

// Name returns the subcommand
func (c taskDeleteCommand) Name() string {
	return cmdTaskDeleteName
}

// Help returns the command's usage, list of flags, and examples
func (c *taskDeleteCommand) Help() string {
	helpText := fmt.Sprintf(`
Usage: consul-terraform-sync task delete [options] <task name>

  Task Delete is used to delete existing tasks. Will not delete a task
  if the task is currently running.

Options:
%s

Example:

  $ consul-terraform-sync task delete my_task
	==> Do you want to delete 'my_task'?
		- This action cannot be undone.
	Only 'yes' will be accepted to approve.

	Enter a value: yes

	==> 'my_task' delete complete!
`, strings.Join(c.meta.helpOptions, "\n"))
	return strings.TrimSpace(helpText)
}

// Synopsis is a short one-line synopsis of the command
func (c *taskDeleteCommand) Synopsis() string {
	return "Deletes existing tasks."
}

// Run runs the command
func (c *taskDeleteCommand) Run(args []string) int {
	c.meta.setFlagsUsage(c.flags, args, c.Help())

	if err := c.flags.Parse(args); err != nil {
		return ExitCodeParseFlagsError
	}

	args = c.flags.Args()
	if ok := c.meta.oneArgCheck(c.Name(), args); !ok {
		return ExitCodeRequiredFlagsError
	}

	taskName := args[0]

	client, err := c.meta.client()
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error: unable to create client for '%s'", taskName))
		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}

	if exitCode, approved := c.meta.requestUserApprovalDelete(taskName); !approved {
		return exitCode
	}

	c.UI.Info(fmt.Sprintf("Deleting task '%s'...\n", taskName))
	err = client.Task().Delete(taskName, nil)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error: unable to delete '%s'", taskName))
		err = processEOFError(client.Scheme(), err)

		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}

	c.UI.Info(fmt.Sprintf("Deleted task '%s'", taskName))

	return ExitCodeOK
}
