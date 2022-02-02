//go:build e2e
// +build e2e

// Tests CTS CLI commands interacting with a running CTS in daemon mode.
package e2e

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/command"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_MetaCommandErrors tests cases that cross subcommands coded in
// the command meta object. This starts up a local Consul server and runs
// CTS in dev mode.
func TestE2E_MetaCommandErrors(t *testing.T) {
	setParallelism(t)

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "meta_errs")
	cleanup := testutils.MakeTempDir(t, tempDir)
	// no defer to delete directory: only delete at end of test if no errors

	configPath := filepath.Join(tempDir, configFile)
	config := fakeHandlerConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock()
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath, api.CTSDevModeFlag)
	defer stop(t)
	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)

	address := cts.FullAddress()

	cases := []struct {
		name           string
		args           []string
		outputContains string
	}{
		{
			"missing required arguments",
			[]string{},
			"Error: this command requires one argument",
		},
		{
			"connect using wrong scheme",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, strings.Replace(address, "http", "https", 1)),
				dbTaskName,
			},
			"HTTP response to HTTPS client",
		},
		{
			"unsupported argument",
			[]string{"-unsupported-flag", fakeFailureTaskName},
			"Error: unsupported arguments in flags",
		},
		{
			"non-existing task",
			[]string{fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()), "non-existent-task"},
			"does not exist or has not been initialized yet",
		},
		{
			"out of order arguments",
			[]string{fakeFailureTaskName, fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress())},
			"All flags are required to appear before positional arguments",
		},
	}

	for _, lifecycle := range []string{"disable", "enable"} {
		// run through all the test cases for each task lifecycle command
		for _, tc := range cases {
			t.Run(fmt.Sprintf("%s/%s", lifecycle, tc.name), func(t *testing.T) {
				subcmd := []string{"task", lifecycle}
				subcmd = append(subcmd, tc.args...)

				output, err := runSubcommand(t, "", subcmd...)
				assert.Contains(t, output, tc.outputContains)
				assert.Error(t, err)
			})
		}
	}

	err = cleanup()
	require.NoError(t, err)
}

// TestE2E_EnableTaskCommand tests the Enable CLI and confirms the expected
// output and state given different paths. This starts up a local Consul server
// and runs CTS with a disabled task.
func TestE2E_EnableTaskCommand(t *testing.T) {
	setParallelism(t)

	cases := []struct {
		name           string
		args           []string
		input          string
		outputContains string
		expectEnabled  bool
	}{
		{
			name:           "happy path",
			args:           []string{disabledTaskName},
			input:          "yes\n",
			outputContains: "enable complete!",
			expectEnabled:  true,
		},
		{
			name:           "auto approve",
			args:           []string{"-auto-approve", disabledTaskName},
			input:          "",
			outputContains: "enable complete!",
			expectEnabled:  true,
		},
		{
			name:           "user does not approve plan",
			args:           []string{disabledTaskName},
			input:          "no\n",
			outputContains: "Cancelled enabling task",
			expectEnabled:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestConsulServer(t)
			defer srv.Stop()

			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "enable_cmd")

			cts := ctsSetup(t, srv, tempDir, disabledTaskConfig())

			subcmd := []string{"task", "enable",
				fmt.Sprintf("-%s=%d", command.FlagPort, cts.Port()),
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
			}
			subcmd = append(subcmd, tc.args...)

			output, err := runSubcommand(t, tc.input, subcmd...)
			assert.NoError(t, err)
			assert.Contains(t, output, tc.outputContains)

			// confirm that the task's final enabled state
			taskStatuses, err := cts.Status().Task(disabledTaskName, nil)
			require.NoError(t, err)
			status, ok := taskStatuses[disabledTaskName]
			require.True(t, ok)
			assert.Equal(t, tc.expectEnabled, status.Enabled)

			if !tc.expectEnabled {
				// only check for events if we expect the task to be enabled
				return
			}

			// check that there was an initial event
			eventCountBase := 1
			eventCountNow := eventCount(t, disabledTaskName, cts.Port())
			require.Equal(t, eventCountBase, eventCountNow,
				"event count did not increment once. task was not triggered as expected")

			// make a change in Consul and confirm a new event is triggered
			now := time.Now()
			service := testutil.TestService{ID: "api-1", Name: "api"}
			testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
			api.WaitForEvent(t, cts, disabledTaskName, now, defaultWaitForEvent)
			eventCountNow = eventCount(t, disabledTaskName, cts.Port())
			require.Equal(t, eventCountBase+1, eventCountNow,
				"event count did not increment once. task was not triggered as expected")
			resourcesPath := filepath.Join(tempDir, disabledTaskName, resourcesDir)
			validateServices(t, true, []string{"api", "api-1", "web"}, resourcesPath)
		})
	}
}

// TestE2E_DisableTaskCommand tests the CLI to disable an enabled task. This test
// starts up a local Consul server and runs CTS in dev mode.
func TestE2E_DisableTaskCommand(t *testing.T) {
	setParallelism(t)

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "disable_cmd")
	cts := ctsSetup(t, srv, tempDir, dbTask())

	cases := []struct {
		name           string
		args           []string
		outputContains string
	}{
		{
			"happy path",
			[]string{fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()), dbTaskName},
			"disable complete!",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			subcmd := []string{"task", "disable"}
			subcmd = append(subcmd, tc.args...)

			output, err := runSubcommand(t, "", subcmd...)
			assert.Contains(t, output, tc.outputContains)
			assert.NoError(t, err)
		})
	}
}

// TestE2E_ReenableTaskTriggers specifically tests the case where an enabled task
// is disabled and then re-enabled. It confirms that the task triggered as
// expected once re-enabled.
// See https://github.com/hashicorp/consul-terraform-sync/issues/320
func TestE2E_ReenableTaskTriggers(t *testing.T) {
	setParallelism(t)

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	// Setup CTS with buffer period enabled
	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "reenable_trigger")
	cleanup := testutils.MakeTempDir(t, tempDir)
	config := baseConfig(tempDir).appendConsulBlock(srv).
		appendTerraformBlock().appendDBTask()
	config = hclConfig(strings.ReplaceAll(string(config),
		"\nbuffer_period {\n\tenabled = false\n}\n", ""))
	configPath := filepath.Join(tempDir, configFile)
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath)
	t.Cleanup(func() {
		_ = cleanup()
		stop(t)
	})

	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)

	// Test that regex filter is filtering service registration information and
	// task triggers
	// 0. Setup: disable task, re-enable it
	// 1. Confirm baseline: check current number of events
	// 2. Register api service instance. Confirm that the task was triggered
	//    (one new event)

	// 0. disable then re-enable the task
	subcmd := []string{"task", "disable", fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()), dbTaskName}
	output, err := runSubcommand(t, "", subcmd...)
	assert.NoError(t, err, output)

	subcmd = []string{"task", "enable", fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()), dbTaskName}
	output, err = runSubcommand(t, "yes\n", subcmd...)
	assert.NoError(t, err, output)

	// 1. get current number of events
	eventCountBase := eventCount(t, dbTaskName, cts.Port())

	// 2. register api service. check triggers task
	now := time.Now()
	service := testutil.TestService{ID: "api-1", Name: "api"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, dbTaskName, now, defaultWaitForEvent)

	eventCountNow := eventCount(t, dbTaskName, cts.Port())
	require.Equal(t, eventCountBase+1, eventCountNow,
		"event count did not increment once. task was not triggered as expected")
}

// TestE2E_DeleteTaskCommand tests deleting a task with the CTS CLI
func TestE2E_DeleteTaskCommand(t *testing.T) {
	setParallelism(t)
	cases := []struct {
		name           string
		taskName       string
		args           []string
		input          string
		outputContains []string
		expectErr      bool
		expectDeleted  bool
	}{
		{
			name:     "happy_path",
			taskName: dbTaskName,
			input:    "yes\n",
			outputContains: []string{
				fmt.Sprintf("Do you want to delete '%s'?", dbTaskName),
				fmt.Sprintf("Deleted task '%s'", dbTaskName)},
			expectErr:     false,
			expectDeleted: true,
		},
		{
			name:     "auto_approve",
			taskName: dbTaskName,
			input:    "",
			args:     []string{"-auto-approve"},
			outputContains: []string{
				fmt.Sprintf("Deleted task '%s'", dbTaskName)},
			expectErr:     false,
			expectDeleted: true,
		},
		{
			name:     "user_does_not_approve_deletion",
			taskName: dbTaskName,
			input:    "no\n",
			outputContains: []string{
				fmt.Sprintf("Do you want to delete '%s'?", dbTaskName),
				fmt.Sprintf("Cancelled deleting task '%s'", dbTaskName),
			},
			expectErr:     false,
			expectDeleted: false,
		},
		{
			name:     "task_does_not_exist",
			taskName: "nonexistent_task",
			input:    "yes\n",
			outputContains: []string{
				fmt.Sprintf("Error: unable to delete '%s'", "nonexistent_task"),
				"request returned 404 status code with error:",
			},
			expectErr:     true,
			expectDeleted: true, // never existed, same as deleted
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestConsulServer(t)
			defer srv.Stop()
			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "delete_cmd")
			cts := ctsSetup(t, srv, tempDir, dbTask())

			// Delete command and user approval input if required
			subcmd := []string{"task", "delete",
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
			}
			subcmd = append(subcmd, tc.args...)
			subcmd = append(subcmd, tc.taskName)
			output, err := runSubcommand(t, tc.input, subcmd...)

			// Verify result and output of command
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			for _, expect := range tc.outputContains {
				assert.Contains(t, output, expect)
			}

			// Confirm whether the task is deleted or not
			_, err = cts.Status().Task(tc.taskName, nil)
			if tc.expectDeleted {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestE2E_CreateTaskCommand tests creating a task with the CTS CLI
func TestE2E_CreateTaskCommand(t *testing.T) {
	setParallelism(t)

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "create_cmd")
	objectVarsFileName := filepath.Join(tempDir, "object.tfvars")
	filenameVarsFileName := filepath.Join(tempDir, "filename.tfvars")
	taskName := "new-task"

	cases := []struct {
		name           string
		taskName       string
		inputTask      string
		tfVarsFiles    map[string]string
		args           []string
		input          string
		outputContains []string
		expectErr      bool
		expectStatus   bool
		checkEvents    bool
	}{
		{
			name:     "happy_path",
			taskName: taskName,
			inputTask: fmt.Sprintf(`
task {
  name           = "%s"
  description    = "Creates a new task"
  module         = "./test_modules/local_instances_file"
  providers      = ["local"]
  services       = ["web"]
  enabled = true
}`, taskName),
			input: "yes\n",
			outputContains: []string{
				fmt.Sprintf("Do you want to perform these actions for '%s'?", taskName),
				fmt.Sprintf("Task '%s' created", taskName)},
			expectErr:    false,
			expectStatus: true,
			checkEvents:  true,
		},
		{
			name:     "with_tf_vars_files",
			taskName: taskName,
			tfVarsFiles: map[string]string{
				objectVarsFileName: "filename = \"e2e.txt\"",
				filenameVarsFileName: `
sample = {
  tags = ["engineer", "cts"]
  user_meta = {
    age             = "32"
    day_of_the_week = "Tuesday"
    name            = "michael"
  }
}`,
			},
			inputTask: fmt.Sprintf(`
task {
  name           = "%s"
  description    = "Creates a new task"
  module         = "./test_modules/with_tfvars_file"
  providers      = ["local"]
  services       = ["web"]
  enabled        = true
  variable_files = ["%s","%s"]
}`, taskName, filenameVarsFileName, objectVarsFileName),
			input: "yes\n",
			outputContains: []string{
				fmt.Sprintf("Do you want to perform these actions for '%s'?", taskName),
				fmt.Sprintf("Task '%s' created", taskName)},
			expectErr:    false,
			expectStatus: true,
			checkEvents:  true,
		},
		{
			name:     "auto_approve",
			taskName: taskName,
			inputTask: fmt.Sprintf(`
task {
  name           = "%s"
  description    = "Creates a new task"
  module         = "./test_modules/local_instances_file"
  providers      = ["local"]
  services       = ["web"]
  enabled = true
}`, taskName),
			outputContains: []string{
				fmt.Sprintf("Task '%s' created", taskName)},
			expectErr:    false,
			expectStatus: true,
			checkEvents:  true,
			args:         []string{"-auto-approve"},
		},
		{
			name:     "user_dose_not_approve_creation",
			taskName: taskName,
			inputTask: fmt.Sprintf(`
task {
  name           = "%s"
  description    = "Creates a new task"
  module         = "./test_modules/local_instances_file"
  providers      = ["local"]
  services       = ["web"]
  enabled = true
}`, taskName),
			input: "no\n",
			outputContains: []string{
				fmt.Sprintf("Do you want to perform these actions for '%s'?", taskName),
				fmt.Sprintf("Cancelled creating task '%s'", taskName),
			},
			expectErr:    false,
			expectStatus: false,
			checkEvents:  true,
		},
		{
			name:     "error_task_already_exists",
			taskName: dbTaskName,
			inputTask: fmt.Sprintf(`
task {
  name           = "%s"
  description    = "Creates a new task"
  module         = "./test_modules/local_instances_file"
  providers      = ["local"]
  services       = ["api"]
  enabled = true
}`, dbTaskName),
			input: "no\n",
			outputContains: []string{
				fmt.Sprintf("Error: unable to generate plan for '%s'", dbTaskName),
				fmt.Sprintf("error: task with name %s already exists", dbTaskName),
			},
			expectErr:    true,
			expectStatus: true,
			checkEvents:  false,
		},
		{
			name:     "error_more_than_one_task",
			taskName: taskName,
			inputTask: fmt.Sprintf(`
task {
  name           = "%s"
  description    = "Creates a new task"
  module         = "./test_modules/local_instances_file"
  providers      = ["local"]
  services       = ["web"]
  enabled = true
}
task {
  name           = "%s"
  description    = "Creates a new task"
  module         = "./test_modules/local_instances_file"
  providers      = ["local"]
  services       = ["web"]
  enabled = true
}
`, taskName, taskName),
			input: "yes\n",
			outputContains: []string{
				"cannot contain more than 1 task, contains 2 tasks",
			},
			expectErr:    true,
			expectStatus: false,
			checkEvents:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestConsulServer(t)
			defer srv.Stop()
			cts := ctsSetup(t, srv, tempDir, dbTask())

			// Write terraform variable files if required by test case
			var vf []string
			for k, v := range tc.tfVarsFiles {
				f, err := os.Create(k)
				require.NoError(t, err)
				_, err = f.WriteString(v)
				require.NoError(t, err)
				vf = append(vf, k)
			}

			// Write task config file
			var taskConfig hclConfig
			taskConfig = taskConfig.appendString(tc.inputTask)
			taskFilePath := filepath.Join(tempDir, "task.hcl")
			taskConfig.write(t, taskFilePath)

			// Create command and user approval input if required
			subcmd := []string{"task", "create",
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
			}
			subcmd = append(subcmd, tc.args...)
			subcmd = append(subcmd, fmt.Sprintf("--task-file=%s", taskFilePath))
			output, err := runSubcommand(t, tc.input, subcmd...)

			// Verify result and output of command
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// remove newlines from output
			re := regexp.MustCompile(`\r?\n`)
			output = re.ReplaceAllString(output, " ")

			for _, expect := range tc.outputContains {
				assert.Contains(t, output, expect)
			}

			// If required by test case, verify contents of generated variables file
			for _, v := range tc.tfVarsFiles {
				expectedFilePath := filepath.Join(tempDir, taskName, "variables.auto.tfvars")
				b, err := ioutil.ReadFile(expectedFilePath)
				require.NoError(t, err)
				assert.Contains(t, string(b), v)
			}

			// Confirm whether the task is created or not
			_, err = cts.Status().Task(tc.taskName, nil)
			if tc.expectStatus {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				return
			}

			// Check events if we expect events to be triggered
			if tc.checkEvents {
				// 1. get current number of events
				eventCountBase := eventCount(t, tc.taskName, cts.Port())

				// 2. register web service. check triggers task
				now := time.Now()
				service := testutil.TestService{ID: "web-1", Name: "web"}
				testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
				api.WaitForEvent(t, cts, tc.taskName, now, defaultWaitForEvent)

				eventCountNow := eventCount(t, tc.taskName, cts.Port())
				require.Equal(t, eventCountBase+1, eventCountNow,
					"event count did not increment once. task was not triggered as expected")
			}
		})
	}
}

// TestE2E_CreateTaskCommand_NoTaskFileProvided tests creating a task with the CTS CLI when no Task File is provided
func TestE2E_CreateTaskCommand_NoTaskFileProvided(t *testing.T) {
	srv := newTestConsulServer(t)
	defer srv.Stop()
	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "create_no_file_cmd")
	cts := ctsSetup(t, srv, tempDir, dbTask())

	// Create command and user approval input if required
	subcmd := []string{"task", "create",
		fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
	}
	output, err := runSubcommand(t, "", subcmd...)

	assert.Error(t, err)

	outputContains := []string{
		"no task file provided",
		"For additional help try 'consul-terraform-sync task create --help'",
	}

	for _, expect := range outputContains {
		assert.Contains(t, output, expect)
	}
}

// TestE2E_CreateDeleteCreateTrigger tests that after creating a task, then deleting
// the task, and then finally re-creating the task, the task will trigger
func TestE2E_CreateDeleteCreateTrigger(t *testing.T) {
	setParallelism(t)

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "create_delete_create_cmd")

	srv := newTestConsulServer(t)
	defer srv.Stop()
	cts := ctsSetup(t, srv, tempDir, dbTask())

	// Write task config file
	var taskConfig hclConfig
	taskName := "new-task"
	inputTask := fmt.Sprintf(`
task {
  name           = "%s"
  description    = "Creates a new task"
  module         = "./test_modules/local_instances_file"
  providers      = ["local"]
  services       = ["web"]
  enabled = true
}`, taskName)
	taskConfig = taskConfig.appendString(inputTask)
	taskFilePath := filepath.Join(tempDir, "task.hcl")
	taskConfig.write(t, taskFilePath)

	// Create task
	subcmdCreate := []string{"task", "create",
		fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
		fmt.Sprintf("--task-file=%s", taskFilePath),
	}

	input := "yes\n"
	_, err := runSubcommand(t, input, subcmdCreate...)
	assert.NoError(t, err)

	// Confirm task was created
	_, err = cts.Status().Task(taskName, nil)
	assert.NoError(t, err)

	// Delete task
	subcmdDelete := []string{"task", "delete",
		fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
		taskName,
	}
	_, err = runSubcommand(t, input, subcmdDelete...)
	require.NoError(t, err)

	// Confirm task was deleted
	_, err = cts.Status().Task(taskName, nil)
	assert.Error(t, err)

	// Re-create task
	_, err = runSubcommand(t, input, subcmdCreate...)
	assert.NoError(t, err)

	// Confirm task was re-created
	_, err = cts.Status().Task(taskName, nil)
	require.NoError(t, err)

	// Verify events trigger
	// 1. get current number of events
	eventCountBase := eventCount(t, taskName, cts.Port())

	// 2. register web service. check triggers task
	now := time.Now()
	service := testutil.TestService{ID: "web-1", Name: "web"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)

	eventCountNow := eventCount(t, taskName, cts.Port())
	require.Equal(t, eventCountBase+1, eventCountNow,
		"event count did not increment once. task was not triggered as expected")
}

// TestE2E_DeleteTaskCommand_Help tests that the usage is outputted
// for the task help commands. Does not require a running CTS binary.
func TestE2E_TaskCommand_Help(t *testing.T) {
	setParallelism(t)
	cases := []struct {
		command        string
		outputContains []string
	}{
		{
			command: "enable",
			outputContains: []string{
				"Usage: consul-terraform-sync task enable [options] <task name>",
				"auto-approve false",
			},
		},
		{
			command: "disable",
			outputContains: []string{
				"Usage: consul-terraform-sync task disable [options] <task name>",
			},
		},
		{
			command: "delete",
			outputContains: []string{
				"Usage: consul-terraform-sync task delete [options] <task name>",
				"auto-approve false",
			},
		},
		{
			command: "create",
			outputContains: []string{
				"Usage: consul-terraform-sync task create [options] --task-file=<task config>",
				"auto-approve false",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			subcmd := []string{"task", tc.command, "-help"}
			output, err := runSubcommand(t, "", subcmd...)
			assert.NoError(t, err)

			for _, expect := range tc.outputContains {
				assert.Contains(t, output, expect)
			}
		})
	}
}
