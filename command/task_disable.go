package command

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/mitchellh/go-wordwrap"
	"github.com/posener/complete"
)

const cmdTaskDisableName = "task disable"

// TaskDisableCommand handles the `task disable` command
type taskDisableCommand struct {
	meta

	flags           *flag.FlagSet
	predictorClient oapigen.ClientWithResponsesInterface
}

func newTaskDisableCommand(m meta) *taskDisableCommand {
	logging.DisableLogging()
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
Usage: consul-terraform-sync task disable [-help] [options] <task name>

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

// AutocompleteFlags returns a mapping of supported flags and autocomplete
// options for this command. The map key for the Flags map should be the
// complete flag such as "-foo" or "--foo".
func (c *taskDisableCommand) AutocompleteFlags() complete.Flags {
	return c.meta.autoCompleteFlags()
}

// AutocompleteArgs returns the argument predictor for this command.
// This commands uses a client to fetch a list of existing tasks
// to predict the correct disable argument
func (c *taskDisableCommand) AutocompleteArgs() complete.Predictor {
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
				if tasks.Enabled != nil && *tasks.Enabled {
					taskNames = append(taskNames, tasks.Name)
				}
			}
		}
		return taskNames
	})
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
