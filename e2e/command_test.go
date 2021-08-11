// +build e2e

// Tests CTS CLI commands interacting with a running CTS in daemon mode.
package e2e

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
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
	delete := testutils.MakeTempDir(t, tempDir)
	// no defer to delete directory: only delete at end of test if no errors

	configPath := filepath.Join(tempDir, configFile)
	config := fakeHandlerConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock()
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath, api.CTSDevModeFlag)
	defer stop(t)
	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)

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

	delete()
}

// TestE2E_EnableTaskCommand tests the CLI to enable a disabled task. This starts
// up a local Consul server and runs CTS in dev mode.
func TestE2E_EnableTaskCommand(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "enable_cmd")
	delete := testutils.MakeTempDir(t, tempDir)
	// no defer to delete directory: only delete at end of test if no errors

	configPath := filepath.Join(tempDir, configFile)
	config := disabledTaskConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock()
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath, api.CTSDevModeFlag)
	defer stop(t)
	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)

	cases := []struct {
		name           string
		args           []string
		input          string
		outputContains string
	}{
		{
			"happy path",
			[]string{fmt.Sprintf("-port=%d", cts.Port()), disabledTaskName},
			"yes\n",
			"enable complete!",
		},
		{
			"user does not approve plan",
			[]string{fmt.Sprintf("-port=%d", cts.Port()), disabledTaskName},
			"no\n",
			"Cancelled enabling task",
		},
		{
			"help flag",
			[]string{"-help"},
			"",
			"Usage: consul-terraform-sync task enable [options] <task name>",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			subcmd := []string{"task", "enable"}
			subcmd = append(subcmd, tc.args...)

			output, err := runSubcommand(t, tc.input, subcmd...)
			assert.NoError(t, err)
			assert.Contains(t, output, tc.outputContains)
		})
	}

	delete()
}

// TestE2E_DisableTaskCommand tests the CLI to disable an enabled task. This
// starts up a local Consul server and runs CTS in dev mode.
func TestE2E_DisableTaskCommand(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "disable_cmd")
	delete := testutils.MakeTempDir(t, tempDir)
	// no defer to delete directory: only delete at end of test if no errors

	configPath := filepath.Join(tempDir, configFile)
	config := baseConfig(tempDir).appendTerraformBlock().
		appendConsulBlock(srv).appendDBTask()
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath, api.CTSDevModeFlag)
	defer stop(t)
	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)

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

	delete()
}

// TestE2E_ReenableTaskTriggers specifically tests the case were an enabled task
// is disabled and then re-enabled. It confirms that the task triggered as
// expected once re-enabled.
// See https://github.com/hashicorp/consul-terraform-sync/issues/320
func TestE2E_ReenableTaskTriggers(t *testing.T) {
	t.Parallel()

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "reenable_trigger")
	cleanup := testutils.MakeTempDir(t, tempDir)

	configPath := filepath.Join(tempDir, configFile)
	config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock().appendDBTask()
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)
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

	now := time.Now()
	subcmd = []string{"task", "enable", fmt.Sprintf("-port=%d", cts.Port()), dbTaskName}
	output, err = runSubcommand(t, "yes\n", subcmd...)
	assert.NoError(t, err, output)
	api.WaitForEvent(t, cts, dbTaskName, now, defaultWaitForEvent)

	// 1. get current number of events
	eventCountBase := eventCount(t, dbTaskName, cts.Port())

	// 2. register api service. check triggers task
	now = time.Now()
	service := testutil.TestService{ID: "api-1", Name: "api"}
	testutils.RegisterConsulService(t, srv, service, testutil.HealthPassing,
		defaultWaitForRegistration)
	api.WaitForEvent(t, cts, dbTaskName, now, defaultWaitForEvent)

	eventCountNow := eventCount(t, dbTaskName, cts.Port())
	require.Equal(t, eventCountBase+1, eventCountNow,
		"event count did not increment once. task was not triggered as expected")

	cleanup()
}

// runSubcommand runs a CTS subcommand and its arguments. If user input is
// required for subcommand, pass it through 'input' parameter. Function returns
// the stdout/err output and any error when executing the subcommand.
// Note: if error returned, output will still contain any stdout/err information.
func runSubcommand(t *testing.T, input string, subcmd ...string) (string, error) {
	cmd := exec.Command("consul-terraform-sync", subcmd...)

	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b

	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)

	err = cmd.Start()
	require.NoError(t, err)

	_, err = stdin.Write([]byte(input))
	require.NoError(t, err)
	stdin.Close()

	err = cmd.Wait()
	return b.String(), err
}
