package command

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/mitchellh/go-wordwrap"
)

const cmdTaskCreateName = "task create"

// TaskCreateCommand handles the `task create` command
type taskCreateCommand struct {
	meta
	autoApprove *bool
	flags       *flag.FlagSet
}

func newTaskCreateCommand(m meta) *taskCreateCommand {
	flags := m.defaultFlagSet(cmdTaskCreateName)
	a := flags.Bool(FlagAutoApprove, false, "Skip interactive approval of inspect plan")
	return &taskCreateCommand{
		meta:        m,
		autoApprove: a,
		flags:       flags,
	}
}

// Name returns the subcommand
func (c taskCreateCommand) Name() string {
	return cmdTaskCreateName
}

// Help returns the command's usage, list of flags, and examples
func (c *taskCreateCommand) Help() string {
	c.meta.setHelpOptions()
	helpText := fmt.Sprintf(`
Usage: consul-terraform-sync task create [options] --task-file=<task config>

  Task Create is used to create a new task. It is not to be used for updating a task, it will not create a task if the
  task name already exists.

Options:
%s

Example:

  $ consul-terraform-sync task create --task-file="task.hcl"
  ==> Inspecting changes to resource if creating task 'my_task'...

  // ... inspection details

  ==> Creating the task will perform the actions described above.
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
func (c *taskCreateCommand) Synopsis() string {
	return "Creates a new task."
}

// Run runs the command
func (c *taskCreateCommand) Run(args []string) int {
	c.meta.setFlagsUsage(c.flags, args, c.Help())
	var taskFile string
	c.flags.StringVar(&taskFile, "task-file", "", "A file containing the hcl or json definition of a task")

	if err := c.flags.Parse(args); err != nil {
		return ExitCodeParseFlagsError
	}

	// Check that a task file was provided
	if len(taskFile) == 0 {
		c.UI.Error(errCreatingRequest)
		c.UI.Output("no task file provided")
		help := fmt.Sprintf("For additional help try 'consul-terraform-sync %s --help'",
			cmdTaskCreateName)
		help = wordwrap.WrapString(help, width)

		c.UI.Output(help)

		return ExitCodeRequiredFlagsError
	}

	// Build a CTS config and use the config.Tasks object only
	cfg, err := config.BuildConfig([]string{taskFile})
	if err != nil {
		c.UI.Error(errCreatingRequest)
		c.UI.Output("unable to read task file")
		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}
	taskConfigs := *cfg.Tasks

	// Check that we have exactly 1 task in the task config return
	l := len(taskConfigs)
	if l > 1 {
		c.UI.Error(errCreatingRequest)
		c.UI.Output(fmt.Sprintf("task file %s cannot contain more "+
			"than 1 task, contains %d tasks", taskFile, l))
		return ExitCodeError
	}

	if l == 0 {
		c.UI.Error(errCreatingRequest)
		c.UI.Output(fmt.Sprintf("task file %s does not contain a task, "+
			"must contain at least one task", taskFile))
		return ExitCodeError
	}

	taskConfig := taskConfigs[0]
	taskName := *taskConfig.Name

	taskReq, err := api.TaskRequestFromTaskConfig(*taskConfig)
	if err != nil {
		c.UI.Error(errCreatingRequest)
		c.UI.Output(fmt.Sprintf("task %s is invalid", taskFile))
		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}

	client, err := c.meta.taskLifecycleClient()
	if err != nil {
		c.UI.Error(errCreatingClient)
		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}

	// First inspect the plan
	c.UI.Info(fmt.Sprintf("Inspecting changes to resource if creating task '%s'...\n", taskName))
	c.UI.Output("Generating plan that Consul Terraform Sync will use Terraform to execute\n")

	taskResp, err := client.CreateTask(context.Background(), api.RunOptionInspect, taskReq)

	if err != nil {
		c.UI.Error(fmt.Sprintf("Error: unable to generate plan for '%s'", taskName))
		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}

	c.UI.Output(fmt.Sprintf("Request ID: %s", taskResp.RequestId))
	b, _ := json.MarshalIndent(taskReq, "    ", "  ")
	c.UI.Output("Request Payload:")
	c.UI.Output(fmt.Sprintf("%s\n", string(b)))
	c.UI.Output(fmt.Sprintf("Plan: \n%s", *taskResp.Run.Plan))
	if taskResp.Run.TfcRunUrl != nil {
		c.UI.Output(fmt.Sprintf("Terraform Cloud Run URL: %s\n", *taskResp.Run.TfcRunUrl))
	}

	if !*c.autoApprove {
		if exitCode, approved := c.meta.requestUserApprovalCreate(taskName); !approved {
			return exitCode
		}
	}

	// Plan approved, create new task and run now
	taskResp, err = client.CreateTask(context.Background(), api.RunOptionNow, taskReq)

	if err != nil {
		c.UI.Error(fmt.Sprintf("Error: unable to create '%s'", taskName))
		msg := wordwrap.WrapString(err.Error(), uint(78))
		c.UI.Output(msg)

		return ExitCodeError
	}

	c.UI.Info(fmt.Sprintf("Task '%s' created", taskResp.Task.Name))
	c.UI.Output(fmt.Sprintf("Request ID: '%s'", taskResp.RequestId))

	return ExitCodeOK
}
