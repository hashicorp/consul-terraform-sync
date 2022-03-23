package command

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/mitchellh/go-wordwrap"
	"github.com/posener/complete"
)

const cmdTaskEnableName = "task enable"

// taskEnableCommand handles the `task enable` command
type taskEnableCommand struct {
	meta
	autoApprove *bool
	flags       *flag.FlagSet

	predictorClient oapigen.ClientWithResponsesInterface
}

func newTaskEnableCommand(m meta) *taskEnableCommand {
	logging.DisableLogging()
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
Usage: consul-terraform-sync task enable [-help] [options] <task name>

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
       - Consul-Terraform-Sync cannot guarantee that these exact actions will be
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

// AutocompleteFlags returns a mapping of supported flags and autocomplete
// options for this command. The map key for the Flags map should be the
// complete flag such as "-foo" or "--foo".
func (c *taskEnableCommand) AutocompleteFlags() complete.Flags {
	return c.meta.autoCompleteFlags()
}

// AutocompleteArgs returns the argument predictor for this command.
// This commands uses a client to fetch a list of existing tasks
// to predict the correct enable argument
func (c *taskEnableCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictFunc(func(a complete.Args) []string {
		var client oapigen.ClientWithResponsesInterface
		var err error
		if c.predictorClient == nil {
			client, err = c.meta.taskLifecycleClient()
			if err != nil {
				return nil
			}
		} else {
			client = c.predictorClient
		}

		tasksResp, err := getTasks(context.Background(), client)
		if err != nil {
			return nil
		}

		taskNames := make([]string, 0)

		if tasksResp.Tasks != nil {
			for _, tasks := range *tasksResp.Tasks {
				if tasks.Enabled != nil && !*tasks.Enabled {
					taskNames = append(taskNames, tasks.Name)
				}
			}
		}
		return taskNames
	})
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
	c.UI.Output("Generating plan that Consul-Terraform-Sync will use Terraform to execute\n")

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
