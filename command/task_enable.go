package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/mitchellh/go-wordwrap"
)

const cmdTaskEnableName = "task enable"

// taskEnableCommand handles the `task enable` command
type taskEnableCommand struct {
	meta
	autoApprove *bool
	flags       *flag.FlagSet
}

func newTaskEnableCommand(m meta) *taskEnableCommand {
	flags := m.defaultFlagSet(cmdTaskEnableName)
	a := flags.Bool(FlagAutoApprove, false, "Skip interactive approval of inspect plan")
	return &taskEnableCommand{
		meta:        m,
		autoApprove: a,
		flags:       flags,
	}
}

// Name returns the subcommand
func (c *taskEnableCommand) Name() string {
	return cmdTaskEnableName
}

// Help returns the command's usage, list of flags, and examples
func (c *taskEnableCommand) Help() string {
	c.meta.setHelpOptions()
	helpText := fmt.Sprintf(`
Usage: consul-terraform-sync task enable [options] <task name>

  Task Enable is used to enable existing tasks. Once enabled, a task will
  run and make changes to your network infrastructure resources. Before
  enabling, the CLI will present the operator with a inspect plan and ask for
  approval.

Options:
%s

Example:

  $ consul-terraform-sync task enable my_task
  ==> Inspecting the changes to resource if enabling task 'my_task' now...

  // ... inspection details

  ==> Enabling the task will perform the actions described above.
      Do you want to perform these actions for 'my_task'?
       - This action cannot be undone.
       - Consul Terraform Sync cannot guarantee that these exact actions will be
	     performed if monitored services have changed.

      Only 'yes' will be accepted to approve, enter 'no' or leave blank to reject.

  Enter a value: yes

  // ... output continues
`, strings.Join(c.meta.helpOptions, "\n"))
	return strings.TrimSpace(helpText)
}

// Synopsis is a short one-line synopsis of the command
func (c *taskEnableCommand) Synopsis() string {
	return "Enables existing tasks to run."
}

// Run runs the command
func (c *taskEnableCommand) Run(args []string) int {
	c.meta.setFlagsUsage(c.flags, args, c.Help())

	if err := c.flags.Parse(args); err != nil {
		return ExitCodeParseFlagsError
	}

	args = c.flags.Args()
	if ok := c.meta.oneArgCheck(c.Name(), args); !ok {
		return ExitCodeRequiredFlagsError
	}

	taskName := args[0]

	c.UI.Info(fmt.Sprintf("Inspecting changes to resource if enabling '%s'...\n",
		taskName))
	c.UI.Output("Generating plan that Consul Terraform Sync will use Terraform to execute\n")

	client, err := c.meta.client()
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error: unable to create client for '%s'", taskName))
		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}

	resp, err := client.Task().Update(taskName, api.UpdateTaskConfig{
		Enabled: config.Bool(true),
	}, &api.QueryParam{Run: driver.RunOptionInspect})
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error: unable to generate plan for '%s'", taskName))
		err = processEOFError(client.Scheme(), err)

		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}
	if resp.Inspect == nil {
		c.UI.Error(fmt.Sprintf("Error: unable to retrieve a plan for '%s'", taskName))
		return ExitCodeError
	}

	c.UI.Output(resp.Inspect.Plan)

	if !resp.Inspect.ChangesPresent {
		// enable the task but no need to run it now
		_, err = client.Task().Update(taskName, api.UpdateTaskConfig{
			Enabled: config.Bool(true)}, nil)
		if err != nil {
			c.UI.Error(fmt.Sprintf("Error: unable to enable '%s'", taskName))
			msg := wordwrap.WrapString(err.Error(), uint(78))
			c.UI.Output(msg)

			return ExitCodeError
		}

		c.UI.Info(fmt.Sprintf("'%s' enable complete!", taskName))
		return ExitCodeOK
	}

	if !*c.autoApprove {
		if exitCode, approved := c.meta.requestUserApprovalEnable(taskName); !approved {
			return exitCode
		}
	}

	c.UI.Info(fmt.Sprintf("Enabling and running '%s'...\n", taskName))
	_, err = client.Task().Update(taskName, api.UpdateTaskConfig{
		Enabled: config.Bool(true),
	}, &api.QueryParam{Run: driver.RunOptionNow})
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error: unable to enable and run '%s'", taskName))
		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}

	c.UI.Info(fmt.Sprintf("'%s' enable complete!", taskName))
	return ExitCodeOK
}
