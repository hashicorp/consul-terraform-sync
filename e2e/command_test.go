//go:build e2e
// +build e2e

// Tests CTS CLI commands interacting with a running CTS in daemon mode.
package e2e

import (
	"bytes"
	"fmt"
	"os/exec"
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

const (
	ctsClientCert = "../testutils/localhost_cert.pem"
	ctsClientKey  = "../testutils/localhost_key.pem"
	ctsCACert     = "../testutils/localhost_cert.pem"
	invalidCert   = "../testutils/cert.pem"
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

// TestE2E_TLSErrors tests the CLI to disable an enabled task. This
// starts up a local Consul server and runs CTS with TLS in dev mode.
func TestE2E_TLSErrors(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "command_tls_errors")

	tlsc := tlsConfig{
		clientCert: ctsClientCert,
		clientKey:  ctsClientKey,
		caCert:     ctsCACert,
	}
	cts := ctsSetupTLS(t, srv, tempDir, dbTask(), tlsc)
	address := cts.FullAddress()

	commands := map[string][]string{
		"task disable": {"task", "disable"},
		"task enable":  {"task", "disable"},
	}

	cases := []struct {
		name           string
		args           []string
		outputContains string
	}{
		{
			"connect using wrong scheme",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, strings.Replace(address, "https", "http", 1)),
				fmt.Sprintf("-%s=%s", command.FlagCACert, ctsCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			"EOF",
		},
		{
			"connect with invalid cert",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, address),
				fmt.Sprintf("-%s=%s", command.FlagCACert, invalidCert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			"signed by unknown authority",
		},
		{
			"no cert",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, address),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			"signed by unknown authority",
		},
	}

	for name, cmd := range commands {
		for _, tc := range cases {
			testName := fmt.Sprintf("%s_%s", tc.name, name)
			t.Run(testName, func(t *testing.T) {

				subcmd := cmd
				subcmd = append(subcmd, tc.args...)

				output, err := runSubcommand(t, "", subcmd...)
				assert.Contains(t, output, tc.outputContains)
				assert.Error(t, err)
			})
		}
	}
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
		tlsConfig      *tlsConfig
	}{
		{
			name:           "happy path",
			args:           []string{disabledTaskName},
			input:          "yes\n",
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
		{
			name: "happy path tls",
			args: []string{
				fmt.Sprintf("-%s=%s", command.FlagCACert, ctsCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				disabledTaskName,
			},
			input:          "yes\n",
			outputContains: "enable complete!",
			expectEnabled:  true,
			tlsConfig: &tlsConfig{
				clientCert: ctsClientCert,
				clientKey:  ctsClientKey,
				caCert:     ctsCACert,
			},
		},
		{
			name: "happy path tl ssl false",
			args: []string{
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "false"),
				disabledTaskName,
			},
			input:          "yes\n",
			outputContains: "enable complete!",
			expectEnabled:  true,
			tlsConfig: &tlsConfig{
				clientCert: ctsClientCert,
				clientKey:  ctsClientKey,
				caCert:     ctsCACert,
			},
		},
		{
			name: "user does not approve plan with tls",
			args: []string{
				fmt.Sprintf("-%s=%s", command.FlagCACert, ctsCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				disabledTaskName,
			},
			input:          "no\n",
			outputContains: "Cancelled enabling task",
			expectEnabled:  false,
			tlsConfig: &tlsConfig{
				clientCert: ctsClientCert,
				clientKey:  ctsClientKey,
				caCert:     ctsCACert,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestConsulServer(t)
			defer srv.Stop()

			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "enable_cmd")

			var cts *api.Client
			if tc.tlsConfig == nil {
				cts = ctsSetup(t, srv, tempDir, disabledTaskConfig(tempDir))
			} else {
				cts = ctsSetupTLS(t, srv, tempDir, disabledTaskConfig(tempDir), *tc.tlsConfig)
			}

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

// TestE2E_DisableTaskCommand tests the CLI to disable an enabled task. This
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

// TestE2E_DisableTaskCommand tests the CLI to disable an enabled task. This
// starts up a local Consul server and runs CTS with in dev mode.
func TestE2E_DisableTaskCommand_TLS(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "disable_tls_cmd")

	tlsc := tlsConfig{
		clientCert: ctsClientCert,
		clientKey:  ctsClientKey,
		caCert:     ctsCACert,
	}
	cts := ctsSetupTLS(t, srv, tempDir, dbTask(), tlsc)

	cases := []struct {
		name           string
		args           []string
		outputContains string
	}{
		{
			"happy path",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, ctsCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			"disable complete!",
		},
		{
			"ssl verify disabled",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "false"),
				dbTaskName,
			},
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
	t.Parallel()

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "reenable_trigger")
	cts := ctsSetup(t, srv, tempDir, dbTask())
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
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, dbTaskName, now, defaultWaitForEvent)

	eventCountNow := eventCount(t, dbTaskName, cts.Port())
	require.Equal(t, eventCountBase+1, eventCountNow,
		"event count did not increment once. task was not triggered as expected")
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
