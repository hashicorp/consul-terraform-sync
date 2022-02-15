package command

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/config"
	mcli "github.com/mitchellh/cli"
	"github.com/mitchellh/go-wordwrap"
)

const cmdTaskCreateName = "task create"

// TaskCreateCommand handles the `task create` command
type taskCreateCommand struct {
	meta
	autoApprove *bool
	taskFile    *string
	flags       *flag.FlagSet
}

func newTaskCreateCommand(m meta) *taskCreateCommand {
	flags := m.defaultFlagSet(cmdTaskCreateName)
	a := flags.Bool(FlagAutoApprove, false, "Skip interactive approval of inspect plan")
	f := flags.String("task-file", "", "[Required] A file containing the hcl or json definition of a task")
	return &taskCreateCommand{
		meta:        m,
		autoApprove: a,
		taskFile:    f,
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
Usage: consul-terraform-sync task create [options] -task-file=<task config>

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
       - Consul-Terraform-Sync cannot guarantee that these exact actions will be
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
	if err := c.flags.Parse(args); err != nil {
		return ExitCodeParseFlagsError
	}

	// Check that a task file was provided
	taskFile := *c.taskFile
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
		c.UI.Output(fmt.Sprintf("task file '%s' cannot contain more "+
			"than 1 task, contains %d tasks", taskFile, l))
		return ExitCodeError
	}

	if l == 0 {
		c.UI.Error(errCreatingRequest)
		c.UI.Output(fmt.Sprintf("task file '%s' does not contain a task, "+
			"must contain at least one task", taskFile))
		return ExitCodeError
	}

	taskConfig := taskConfigs[0]

	// Check if task config provided is using the deprecated fields
	if err = handleDeprecations(c.UI, taskConfig); err != nil {
		return ExitCodeError
	}

	taskReq, err := api.TaskRequestFromTaskConfig(*taskConfig)
	if err != nil {
		c.UI.Error(errCreatingRequest)
		c.UI.Output(fmt.Sprintf("task '%s' is invalid", taskFile))
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
	taskName := *taskConfig.Name
	c.UI.Info(fmt.Sprintf("Inspecting changes to resource if creating task '%s'...\n", taskName))
	c.UI.Output("Generating plan that Consul-Terraform-Sync will use Terraform to execute\n")

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

	c.UI.Info(fmt.Sprintf("Creating and running task '%s'...", taskName))
	c.UI.Output("The task creation request has been sent to the CTS server.")
	c.UI.Output("Please be patient as it may take some time to see a confirmation that this task has completed.")
	c.UI.Output("Warning: Terminating this process will not stop task creation.\n")

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

// handleDeprecations handles fields that have been deprecated as part of the config
// as fields are removed, the checks here will also be removed
func handleDeprecations(ui mcli.Ui, tc *config.TaskConfig) error {

	// Create an error flag, we want to accumulate and print out all the errors without returning
	isError := false

	// Handle deprecated task.source usage over task.module
	if tc.DeprecatedSource != nil {
		isError = true
		ui.Error(errCreatingRequest)
		ui.Output(generateSourceFieldMsg(*tc.DeprecatedSource))
	}

	// Handle deprecated source_input block over module_input
	if tc.DeprecatedSourceInputs != nil {
		isError = true
		for _, si := range *tc.DeprecatedSourceInputs {
			ui.Error(errCreatingRequest)
			switch si.(type) {
			case *config.ServicesModuleInputConfig:
				ui.Output(generateSourceInputBlockMsg("services"))
			case *config.ConsulKVModuleInputConfig:
				ui.Output(generateSourceInputBlockMsg("consul-kv"))
			}
		}
	}

	// Handle deprecated condition source_includes_var usage over use_as_module_input
	if tc.Condition != nil {
		switch v := tc.Condition.(type) {
		case *config.ServicesConditionConfig:
			if v.DeprecatedSourceIncludesVar != nil {
				isError = true
				ui.Error(errCreatingRequest)
				ui.Output(generateSourceIncludesVarMsg("services", *v.DeprecatedSourceIncludesVar))
			}
		case *config.CatalogServicesConditionConfig:
			if v.DeprecatedSourceIncludesVar != nil {
				isError = true
				ui.Error(errCreatingRequest)
				ui.Output(generateSourceIncludesVarMsg("catalog-services", *v.DeprecatedSourceIncludesVar))
			}
		case *config.ConsulKVConditionConfig:
			if v.DeprecatedSourceIncludesVar != nil {
				isError = true
				ui.Error(errCreatingRequest)
				ui.Output(generateSourceIncludesVarMsg("consul-kv", *v.DeprecatedSourceIncludesVar))
			}
		}
	}

	// Handle deprecated task.services
	if len(tc.DeprecatedServices) != 0 {
		isError = true
		ui.Error(errCreatingRequest)
		if tc.Condition != nil {
			switch tc.Condition.(type) {
			case *config.ServicesConditionConfig:
				action := `list of 'services' and 'condition "services"' block cannot both be configured. ` +
					`Consider using the 'names' field under 'condition "services"'`
				ui.Output(generateServiceFieldMsg(action))
			default:
				action := generateServiceModuleInputBlockAction(tc.DeprecatedServices)
				ui.Output(generateServiceFieldMsg(action))
			}
		}

		if tc.ModuleInputs != nil {
			for _, si := range *tc.ModuleInputs {
				switch si.(type) {
				case *config.ServicesModuleInputConfig:
					action := `list of 'services' and 'module_input "services"' block cannot both be configured. ` +
						`Consider using the 'names' field under 'module_input "services"'`
					ui.Output(generateServiceFieldMsg(action))
				}
			}
		}

		if tc.ModuleInputs == nil && tc.Condition == nil {
			action := generateServicesAction(tc.DeprecatedServices)
			ui.Output(generateServiceFieldMsg(action))
		}
	}

	if isError {
		return errors.New("invalid config")
	}
	return nil
}

func generateServicesAction(services []string) string {
	list := `"` + strings.Join(services, `","`) + `"`
	return fmt.Sprintf(servicesAction, list)
}

const servicesAction = `Please replace the 'services' field with the following 'condition "services"' block

task {
  ...
  condition "services" {
    names=[%s]
  }
  ...
}`

func generateServiceModuleInputBlockAction(services []string) string {
	list := `"` + strings.Join(services, `","`) + `"`
	return fmt.Sprintf(servicesModuleInputBlockAction, list)
}

const servicesModuleInputBlockAction = `Please replace the 'services' field with the following 'module_input' block

task {
  ...
  module_input "services" {
    names=[%s]
  }
  ...
}`

func generateServiceFieldMsg(action string) string {
	return fmt.Sprintf(servicesFieldMsg, action)
}

const servicesFieldMsg = `the 'services' field in the task block is no longer supported ` +
	`

'services' in a task configuration are to be replaced with the following:
 * condition "services": if there is _no_ preexisting condition block configured in your task
 * module_input "services": if there is a preexisting condition block configured in your task

%s

For more details and additional examples, please see:
https://consul.io/docs/nia/release-notes/0-5-0#deprecate-services-field
`

func generateSourceFieldMsg(module string) string {
	return fmt.Sprintf(sourceFieldMsg, module, module)
}

const sourceFieldMsg = `the 'source' field in the task block is no longer supported ` +
	`

Please replace 'source' with 'module' in your task configuration.

Suggested replacement:
|    task {
|  -   source =  "%s"
|  +   module =  "%s"
|      ...
|    }

For more details and examples, please see:
https://consul.io/docs/nia/release-notes/0-5-0#deprecate-source-field
`

func generateSourceInputBlockMsg(moduleInput string) string {
	return fmt.Sprintf(sourceInputBlockMsg, moduleInput, moduleInput)
}

const sourceInputBlockMsg = `the 'source_input' block in the task is no longer supported ` +
	`

Please replace 'source_input' with 'module_input' in your task configuration.

Suggested replacement:
|    task {
|  -   source_input "%s" {
|  +   module_input "%s" {
|        ...
|      }
|      ...
|    }

For more details and examples, please see:
https://consul.io/docs/nia/release-notes/0-5-0#deprecate-source_input-block
`

func generateSourceIncludesVarMsg(condition string, useAsModuleInput bool) string {
	return fmt.Sprintf(sourceIncludesVarMsg, condition, condition, useAsModuleInput, useAsModuleInput)
}

const sourceIncludesVarMsg = `the 'source_includes_var' field in the task's 'condition "%s"' block is no longer supported` +
	`

Please replace 'source_includes_var' with 'use_as_module_input' in your condition configuration.

Suggested replacement:
|    task {
|      condition "%s" {
|  -     source_includes_var = %t
|  +     use_as_module_input = %t
|      }
|      ...
|    }

For more details and examples, please see:
https://consul.io/docs/nia/release-notes/0-5-0#deprecate-source_includes_var-field
`
