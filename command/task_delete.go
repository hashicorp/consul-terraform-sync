package command

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/mitchellh/go-wordwrap"
	"github.com/posener/complete"
)

const cmdTaskDeleteName = "task delete"

// TaskDeleteCommand handles the `task delete` command
type taskDeleteCommand struct {
	meta
	autoApprove *bool
	flags       *flag.FlagSet

	predictorClient oapigen.ClientWithResponsesInterface
}

func newTaskDeleteCommand(m meta) *taskDeleteCommand {
	logging.DisableLogging()
	flags := m.defaultFlagSet(cmdTaskDeleteName)
	a := flags.Bool(FlagAutoApprove, false, "Skip interactive approval of deleting a task")
	return &taskDeleteCommand{
		meta:        m,
		autoApprove: a,
		flags:       flags,
	}
}

// Name returns the subcommand
func (c taskDeleteCommand) Name() string {
	return cmdTaskDeleteName
}

// Help returns the command's usage, list of flags, and examples
func (c *taskDeleteCommand) Help() string {
	c.meta.setHelpOptions()
	helpText := fmt.Sprintf(`
Usage: consul-terraform-sync task delete [-help] [options] <task name>

  Task Delete is used to delete an existing task. If the task is not running,
  then it is deleted immediately. Otherwise, it will be deleted once the task
  is complete.

Options:
%s

Example:

  $ consul-terraform-sync task delete my_task
  ==> Do you want to delete 'my_task'?
       - This action cannot be undone.
       - Deleting a task will not destroy the infrastructure managed by the task.
       - If the task is not running, it will be deleted immediately.
       - If the task is running, it will be deleted once it has completed.
      Only 'yes' will be accepted to approve, enter 'no' or leave blank to reject.

  Enter a value: yes

  ==> Marking task 'my_task' for deletion...

  ==> Task 'my_task' has been marked for deletion and will be deleted when not running.
`, strings.Join(c.meta.helpOptions, "\n"))
	return strings.TrimSpace(helpText)
}

// Synopsis is a short one-line synopsis of the command
func (c *taskDeleteCommand) Synopsis() string {
	return "Deletes an existing task."
}

// AutocompleteFlags returns a mapping of supported flags and autocomplete
// options for this command. The map key for the Flags map should be the
// complete flag such as "-foo" or "--foo".
func (c *taskDeleteCommand) AutocompleteFlags() complete.Flags {
	return mergeAutocompleteFlags(c.meta.autoCompleteFlags(),
		complete.Flags{
			fmt.Sprintf("-%s", FlagAutoApprove): complete.PredictNothing,
		})
}

// AutocompleteArgs returns the argument predictor for this command.
// This commands uses a client to fetch a list of existing tasks
// to predict the correct delete argument
func (c *taskDeleteCommand) AutocompleteArgs() complete.Predictor {
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
				taskNames = append(taskNames, tasks.Name)
			}
		}
		return taskNames
	})
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

	client, err := c.meta.taskLifecycleClient()
	if err != nil {
		c.UI.Error(errCreatingClient)
		c.UI.Output(fmt.Sprintf("client could not be created for '%s'", taskName))
		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}

	if !*c.autoApprove {
		if exitCode, approved := c.meta.requestUserApprovalDelete(taskName); !approved {
			return exitCode
		}
	}

	c.UI.Info(fmt.Sprintf("Marking task '%s' for deletion...\n", taskName))
	resp, err := client.DeleteTaskByName(context.Background(), taskName)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error: unable to delete '%s'", taskName))
		err = processEOFError(client.Scheme(), err)

		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}

	c.UI.Info(fmt.Sprintf("Task '%s' has been marked for deletion "+
		"and will be deleted when not running.", taskName))

	return ExitCodeOK
}
