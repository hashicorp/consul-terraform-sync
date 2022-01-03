//go:build e2e
// +build e2e

// Tests CTS CLI commands interacting with a running CTS in daemon mode.
package e2e

import (
	"fmt"
	"path/filepath"
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
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "meta_errs")
	deleteTemp := testutils.MakeTempDir(t, tempDir)
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
			[]string{fmt.Sprintf("-port=%d", cts.Port()), "non-existent-task"},
			"does not exist or has not been initialized yet",
		},
		{
			"out of order arguments",
			[]string{fakeFailureTaskName, fmt.Sprintf("-port %d", cts.Port())},
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

	deleteTemp()
}

// TestE2E_EnableTaskCommand tests the Enable CLI and confirms the expected
// output and state given different paths. This starts up a local Consul server
// and runs CTS with a disabled task.
func TestE2E_EnableTaskCommand(t *testing.T) {
	t.Parallel()

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
		{
			name:           "help flag",
			args:           []string{"-help"},
			input:          "",
			outputContains: "Usage: consul-terraform-sync task enable [options] <task name>",
			expectEnabled:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestConsulServer(t)
			defer srv.Stop()

			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "enable_cmd")

			cts := ctsSetup(t, srv, tempDir, disabledTaskConfig(tempDir))

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
		})
	}
}

// TestE2E_DisableTaskCommand tests the CLI to disable an enabled task. This test
// starts up a local Consul server and runs CTS in dev mode.
func TestE2E_DisableTaskCommand(t *testing.T) {
	t.Parallel()

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
			[]string{fmt.Sprintf("-port=%d", cts.Port()), dbTaskName},
			"disable complete!",
		},
		{
			"help flag",
			[]string{"-help"},
			"consul-terraform-sync task disable [options] <task name>",
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
	t.Parallel()

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
		cleanup()
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
	subcmd := []string{"task", "disable", fmt.Sprintf("-port=%d", cts.Port()), dbTaskName}
	output, err := runSubcommand(t, "", subcmd...)
	assert.NoError(t, err, output)

	subcmd = []string{"task", "enable", fmt.Sprintf("-port=%d", cts.Port()), dbTaskName}
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
	t.Parallel()
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
			args: []string{"-auto-approve"},
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
				fmt.Sprintf("request returned 404 status code with error:"),
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

// TestE2E_DeleteTaskCommand_Help tests that the usage is outputted
// for the task delete help command. Does not require a running
// CTS binary.
func TestE2E_DeleteTaskCommand_Help(t *testing.T) {
	t.Parallel()
	subcmd := []string{"task", "delete", "-help"}
	output, err := runSubcommand(t, "", subcmd...)
	assert.NoError(t, err)
	assert.Contains(t, output,
		"Usage: consul-terraform-sync task delete [options] <task name>")
}
