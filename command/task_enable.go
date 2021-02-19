package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/mitchellh/go-wordwrap"
)

// taskEnableCommand handles the `task enable` command
type taskEnableCommand struct {
	meta
}

// Name returns the subcommand
func (c *taskEnableCommand) Name() string {
	return "task enable"
}

// Help returns the command's usage, list of flags, and examples
func (c *taskEnableCommand) Help() string {
	helpText := `
Usage: consul-terraform-sync task enable [options] <task name>

  Task Enable is used to enable existing tasks. Once enabled, a task will
  run and make changes to your network infrastructure resources. Before
  enabling, the CLI will present the operator with a inspect plan and ask for
  approval.

Options:
  No options are currently available

Example:

  $ consul-terraform-sync task enable my_task
  ==> Inspecting the changes to resource if enabling task 'my_task' now...

  // ... inspection details

  ==> Enabling the task will perform the actions described above.
      Do you want to perform these actions for 'my_task'?
       - This action cannot be undone.
       - CTS cannot guarantee that these exact actions will be performed if
         monitored services have changed.

      Only 'yes' will be accepted to approve.

Enter a value: yes

==> Enabling and running 'my_task'...

==> 'my_task' enable complete!
`
	return strings.TrimSpace(helpText)
}

// Synopsis is a short one-line synopsis of the command
func (c *taskEnableCommand) Synopsis() string {
	return "Enables existing tasks to run."
}

// Run runs the command
func (c *taskEnableCommand) Run(args []string) int {
	flags := c.meta.defaultFlagSet(c, args)

	if err := flags.Parse(args); err != nil {
		return ExitCodeParseFlagsError
	}

	args = flags.Args()
	if ok := c.meta.oneArgCheck(c, args); !ok {
		return ExitCodeRequiredFlagsError
	}

	taskName := args[0]

	c.UI.Info(fmt.Sprintf("Inspecting changes to resource if enabling '%s'...\n",
		taskName))
	c.UI.Output("Generating plan that CTS will use Terraform to execute\n")

	client := c.meta.client()
	plan, err := client.Task().Update(taskName, api.UpdateTaskConfig{
		Enabled: config.Bool(true),
	}, &api.QueryParam{Run: driver.RunOptionInspect})
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error: unable to generate plan for '%s'", taskName))
		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}

	c.UI.Output(plan)

	if detected := c.meta.changesDetected(plan); !detected {
		c.UI.Info(fmt.Sprintf("'%s' enable complete!", taskName))
		return ExitCodeOK
	}

	if exitCode, approved := c.meta.requestUserApproval(taskName); !approved {
		return exitCode
	}

	c.UI.Info(fmt.Sprintf("Enabling and running '%s'...\n", taskName))
	_, err = client.Task().Update(taskName, api.UpdateTaskConfig{
		Enabled: config.Bool(true),
	}, &api.QueryParam{Run: driver.RunOptionNow})
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error: unable to enable '%s'", taskName))
		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}

	c.UI.Info(fmt.Sprintf("'%s' enable complete!", taskName))
	return ExitCodeOK
}
