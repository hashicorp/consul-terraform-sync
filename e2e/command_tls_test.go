//go:build e2e
// +build e2e

// Tests CTS CLI commands interacting using TLS/mTLS with a running CTS in daemon mode.
package e2e

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/command"
	"github.com/hashicorp/consul-terraform-sync/testutils"
)

const (
	invalidCACert = "../testutils/certs/consul_cert.pem"
	missingCACert = "../testutils/certs/localhost_cert2.pem"
)

const (
	testTaskFileContent = `
task {
  name           = "test-task"
  description    = "Creates a new task"
  module         = "./test_modules/local_instances_file"
  providers      = ["local"]
  condition "services" {
    names = ["api"]
  }
  enabled = true
}`
)

// TestE2E_CommandTLSErrors tests error scenarios using CLI commands with TLS. This
// starts up a local Consul server and runs CTS with TLS in dev mode.
func TestE2E_CommandTLSErrors(t *testing.T) {
	setParallelism(t)

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "command_tls_errors")

	tlsc := defaultTLSConfig()
	cts := ctsSetupTLS(t, srv, tempDir, dbTask(), tlsc)
	address := cts.FullAddress()

	// Write necessary files to temp directory that may be needed for test cases
	var taskConfig hclConfig
	taskConfig = taskConfig.appendString(testTaskFileContent)
	taskFilePath := filepath.Join(tempDir, "task.hcl")
	taskConfig.write(t, taskFilePath)

	// Get all test certs and move them to the CA path directory
	certs := []string{
		invalidCACert,
		missingCACert,
	}
	clientCAPath := copyClientCerts(t, certs, tempDir)

	commands := map[string][]string{
		"task disable": {"task", "disable"},
		"task enable":  {"task", "enable"},
		"task delete":  {"task", "delete", "-auto-approve"}, // Doesn't inspect so need to approve to get error
		"task create":  {"task", "create", fmt.Sprintf("--task-file=%s", taskFilePath)},
	}

	cases := []struct {
		name           string
		args           []string
		envVariables   []string
		outputContains string
	}{
		{
			"connect using wrong scheme",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, strings.Replace(address, "https", "http", 1)),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			[]string{},
			"sent an HTTP request to an HTTPS server",
		},
		{
			"connect using wrong scheme override right scheme from environment",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, strings.Replace(address, "https", "http", 1)),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			[]string{
				fmt.Sprintf("%s-%s", api.EnvAddress, address),
			},
			"sent an HTTP request to an HTTPS server",
		},
		{
			"connect with invalid cert",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, address),
				fmt.Sprintf("-%s=%s", command.FlagCACert, invalidCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			[]string{},
			"signed by unknown authority",
		},
		{
			"connect with ca path that does not include the server certificate ca",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, address),
				fmt.Sprintf("-%s=%s", command.FlagCAPath, clientCAPath),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			[]string{},
			"signed by unknown authority",
		},
		{
			"connect with invalid cert override env to set SSL verify to true",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, address),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			[]string{"%s-%s", api.EnvTLSSSLVerify, "false"},
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

				// Remove newline characters
				re := regexp.MustCompile(`\r?\n`)
				output = re.ReplaceAllString(output, " ")

				require.Contains(t, output, tc.outputContains)
				require.Error(t, err)
			})
		}
	}
}

// TestE2E_CommandTLS tests CLI commands using TLS. This
// starts up a local Consul server and runs CTS with TLS in dev mode.
func TestE2E_CommandTLS(t *testing.T) {
	setParallelism(t)

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "command_tls")

	tlsc := defaultTLSConfig()
	cts := ctsSetupTLS(t, srv, tempDir, dbTask(), tlsc)

	// Write necessary files to temp directory that may be needed for test cases
	var taskConfig hclConfig
	taskConfig = taskConfig.appendString(testTaskFileContent)
	taskFilePath := filepath.Join(tempDir, "task.hcl")
	taskConfig.write(t, taskFilePath)

	commands := getTestCommands(taskFilePath)

	cases := []struct {
		name         string
		optionArgs   []string
		envVariables []string
	}{
		{
			name: "happy path",
			optionArgs: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
			},
		},
		{
			name: "happy path environment variables",
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, cts.FullAddress()),
				fmt.Sprintf("%s=%s", api.EnvTLSCACert, defaultCTSCACert),
				fmt.Sprintf("%s=%s", api.EnvTLSSSLVerify, "true"),
			},
		},
		{
			name: "flags override environment variables",
			optionArgs: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
			},
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, "bogus_address"),
				fmt.Sprintf("%s=%s", api.EnvTLSCACert, "bogus_cert"),
			},
		},
		{
			name: "ssl verify flag set to false",
			optionArgs: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "false"),
			},
		},
		{
			name: "ssl verify environment set to false",
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, cts.FullAddress()),
				fmt.Sprintf("%s=%s", api.EnvTLSSSLVerify, "false"),
			},
		},
	}

	for _, cmd := range commands {
		for _, tc := range cases {
			testName := fmt.Sprintf("%s_%s", tc.name, cmd.name)
			t.Run(testName, func(t *testing.T) {
				subcmd := cmd.subcmd
				subcmd = append(subcmd, tc.optionArgs...)
				subcmd = append(subcmd, cmd.arg)

				output, err := runSubCommandWithEnvVars(t, cmd.input, tc.envVariables, subcmd...)
				require.Contains(t, output, cmd.outputContains)

				if !cmd.isErrorExpected {
					require.NoError(t, err)
				}
			})
		}
	}
}

// TestE2E_CommandTLS_CAPath tests CLI commands using TLS providing a CA path instead of a CA cert file. This
// starts up a local Consul server and runs CTS with TLS in dev mode.
func TestE2E_CommandTLS_CAPath(t *testing.T) {
	setParallelism(t)

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "command_tls_capath")

	tlsc := tlsConfigWithCAPath(tempDir)
	cts := ctsSetupTLS(t, srv, tempDir, dbTask(), tlsc)

	// Write necessary files to temp directory that may be needed for test cases
	var taskConfig hclConfig
	taskConfig = taskConfig.appendString(testTaskFileContent)
	taskFilePath := filepath.Join(tempDir, "task.hcl")
	taskConfig.write(t, taskFilePath)

	commands := getTestCommands(taskFilePath)

	cases := []struct {
		name         string
		optionArgs   []string
		envVariables []string
	}{
		{
			name: "ca path flag",
			optionArgs: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCAPath, tlsc.caPath),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
			},
		},
		{
			name: "ca path environment variables",
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, cts.FullAddress()),
				fmt.Sprintf("%s=%s", api.EnvTLSCAPath, tlsc.caPath),
				fmt.Sprintf("%s=%s", api.EnvTLSSSLVerify, "true"),
			},
		},
		{
			name: "ca path flags override environment variables",
			optionArgs: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCAPath, tlsc.caPath),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
			},
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, "bogus_address"),
				fmt.Sprintf("%s=%s", api.EnvTLSCAPath, "/path/bogus_cert"),
			},
		},
		{
			name: "ca path overrides ca cert",
			optionArgs: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagCAPath, "/path/bogus_cert"),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
			},
		},
	}

	for _, cmd := range commands {
		for _, tc := range cases {
			testName := fmt.Sprintf("%s_%s", tc.name, cmd.name)
			t.Run(testName, func(t *testing.T) {
				subcmd := cmd.subcmd
				subcmd = append(subcmd, tc.optionArgs...)
				subcmd = append(subcmd, cmd.arg)

				output, err := runSubCommandWithEnvVars(t, cmd.input, tc.envVariables, subcmd...)
				require.Contains(t, output, cmd.outputContains)
				if !cmd.isErrorExpected {
					require.NoError(t, err)
				}
			})
		}
	}
}

// TestE2E_CommandMTLSErrors tests error scenarios using CLI commands with mTLS. This
// starts up a local Consul server and runs CTS with TLS in dev mode.
func TestE2E_CommandMTLSErrors(t *testing.T) {
	setParallelism(t)

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "command_mtls_errors")

	tlsc := mtlsConfigWithCAPath(tempDir)
	cts := ctsSetupTLS(t, srv, tempDir, dbTask(), tlsc)

	// Write necessary files to temp directory that may be needed for test cases
	var taskConfig hclConfig
	taskConfig = taskConfig.appendString(testTaskFileContent)
	taskFilePath := filepath.Join(tempDir, "task.hcl")
	taskConfig.write(t, taskFilePath)

	// Get all test certs and move them to the CA path directory
	certs := []string{
		missingCACert,
		invalidCACert,
	}

	clientCAPath := copyClientCerts(t, certs, tempDir)

	commands := map[string][]string{
		"task disable": {"task", "disable"},
		"task enable":  {"task", "enable"},
		"task delete":  {"task", "delete", "-auto-approve"},
		"task create":  {"task", "create", fmt.Sprintf("--task-file=%s", taskFilePath)},
	}

	cases := []struct {
		name           string
		args           []string
		outputContains string
	}{
		{
			"connect with invalid ca cert",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, invalidCACert),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, defaultCTSClientCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, defaultCTSClientKey),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			"signed by unknown authority",
		},
		{
			"connect with client ca path that does not include server cert ca",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCAPath, clientCAPath),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, defaultCTSClientCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, defaultCTSClientKey),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			"signed by unknown authority",
		},
		{
			"no client cert key pair provided",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			"bad certificate",
		},
		{
			"ssl verify disabled and no cert key pair provided",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "false"),
				dbTaskName,
			},
			"bad certificate",
		},
	}

	for name, cmd := range commands {
		for _, tc := range cases {
			testName := fmt.Sprintf("%s_%s", tc.name, name)
			t.Run(testName, func(t *testing.T) {
				subcmd := cmd
				subcmd = append(subcmd, tc.args...)

				output, err := runSubcommand(t, "", subcmd...)

				// Remove newline characters
				re := regexp.MustCompile(`\r?\n`)
				output = re.ReplaceAllString(output, " ")

				require.Contains(t, output, tc.outputContains)
				require.Error(t, err)
			})
		}
	}
}

// TestE2E_CommandMTLS tests CLI commands using mTLS. This
// starts up a local Consul server and runs CTS with TLS in dev mode.
func TestE2E_CommandMTLS(t *testing.T) {
	setParallelism(t)

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "command_mtls")

	tlsc := defaultMTLSConfig()
	cts := ctsSetupTLS(t, srv, tempDir, dbTask(), tlsc)

	// Write necessary files to temp directory that may be needed for test cases
	var taskConfig hclConfig
	taskConfig = taskConfig.appendString(testTaskFileContent)
	taskFilePath := filepath.Join(tempDir, "task.hcl")
	taskConfig.write(t, taskFilePath)

	commands := getTestCommands(taskFilePath)

	cases := []struct {
		name         string
		optionArgs   []string
		envVariables []string
	}{
		{
			name: "happy path",
			optionArgs: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, defaultCTSClientCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, defaultCTSClientKey),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
			},
		},
		{
			name: "happy path environment variables",
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, cts.FullAddress()),
				fmt.Sprintf("%s=%s", api.EnvTLSCACert, defaultCTSCACert),
				fmt.Sprintf("%s=%s", api.EnvTLSSSLVerify, "true"),
				fmt.Sprintf("%s=%s", api.EnvTLSClientCert, defaultCTSClientCert),
				fmt.Sprintf("%s=%s", api.EnvTLSClientKey, defaultCTSClientKey),
			},
		},
		{
			name: "flags override environment variables",
			optionArgs: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, defaultCTSClientCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, defaultCTSClientKey),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
			},
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, "bogus_address"),
				fmt.Sprintf("%s=%s", api.EnvTLSCACert, "bogus_ca_cert"),
				fmt.Sprintf("%s=%s", api.EnvTLSSSLVerify, "true"),
				fmt.Sprintf("%s=%s", api.EnvTLSClientCert, "bogus_client_cert"),
				fmt.Sprintf("%s=%s", api.EnvTLSClientKey, "bogus_client_key"),
			},
		},
		{
			name: "ssl verify disabled",
			optionArgs: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, defaultCTSClientCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, defaultCTSClientKey),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "false"),
			},
		},
	}

	for _, cmd := range commands {
		for _, tc := range cases {
			testName := fmt.Sprintf("%s_%s", tc.name, cmd.name)
			t.Run(testName, func(t *testing.T) {
				subcmd := cmd.subcmd
				subcmd = append(subcmd, tc.optionArgs...)
				subcmd = append(subcmd, cmd.arg)

				output, err := runSubCommandWithEnvVars(t, cmd.input, tc.envVariables, subcmd...)
				require.Contains(t, output, cmd.outputContains)
				if !cmd.isErrorExpected {
					require.NoError(t, err)
				}
			})
		}
	}
}

// TestE2E_CommandMTLS_CAPath tests CLI commands using mTLS providing a CAPath rather than a CA cert file. This
// starts up a local Consul server and runs CTS with TLS in dev mode.
func TestE2E_CommandMTLS_CAPath(t *testing.T) {
	setParallelism(t)

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "command_mtls_capath")

	tlsc := mtlsConfigWithCAPath(tempDir)
	cts := ctsSetupTLS(t, srv, tempDir, dbTask(), tlsc)

	// Write necessary files to temp directory that may be needed for test cases
	var taskConfig hclConfig
	taskConfig = taskConfig.appendString(testTaskFileContent)
	taskFilePath := filepath.Join(tempDir, "task.hcl")
	taskConfig.write(t, taskFilePath)

	commands := getTestCommands(taskFilePath)

	cases := []struct {
		name         string
		optionArgs   []string
		envVariables []string
	}{
		{
			name: "ca path flag",
			optionArgs: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCAPath, tlsc.caPath),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, defaultCTSClientCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, defaultCTSClientKey),
			},
		},
		{
			name: "using alternate cert",
			optionArgs: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, alternateCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, alternateKey),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
			},
		},
		{
			name: "ca path environment variables",
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, cts.FullAddress()),
				fmt.Sprintf("%s=%s", api.EnvTLSCAPath, tlsc.caPath),
				fmt.Sprintf("%s=%s", api.EnvTLSSSLVerify, "true"),
				fmt.Sprintf("%s=%s", api.EnvTLSClientCert, defaultCTSClientCert),
				fmt.Sprintf("%s=%s", api.EnvTLSClientKey, defaultCTSClientKey),
			},
		},
		{
			name: "ca override environment variables",
			optionArgs: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCAPath, tlsc.caPath),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, defaultCTSClientCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, defaultCTSClientKey),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
			},
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, "bogus_address"),
				fmt.Sprintf("%s=%s", api.EnvTLSCAPath, "path/bogus_ca_path"),
				fmt.Sprintf("%s=%s", api.EnvTLSSSLVerify, "true"),
				fmt.Sprintf("%s=%s", api.EnvTLSClientCert, "bogus_client_cert"),
				fmt.Sprintf("%s=%s", api.EnvTLSClientKey, "bogus_client_key"),
			},
		},
	}

	for _, cmd := range commands {
		for _, tc := range cases {
			testName := fmt.Sprintf("%s_%s", tc.name, cmd.name)
			t.Run(testName, func(t *testing.T) {
				subcmd := cmd.subcmd
				subcmd = append(subcmd, tc.optionArgs...)
				subcmd = append(subcmd, cmd.arg)

				output, err := runSubCommandWithEnvVars(t, cmd.input, tc.envVariables, subcmd...)
				require.Contains(t, output, cmd.outputContains)
				if !cmd.isErrorExpected {
					require.NoError(t, err)
				}
			})
		}
	}
}

func copyClientCerts(t *testing.T, certsToCopy []string, tempDir string) string {
	// Get all test certs and move them to the CA path directory
	clientCAPath := filepath.Join(tempDir, "clientCert")
	cleanup := testutils.MakeTempDir(t, clientCAPath)
	t.Cleanup(func() {
		_ = cleanup()
	})
	testutils.CopyFiles(t, certsToCopy, clientCAPath)

	return clientCAPath
}

type testCommand struct {
	name            string
	subcmd          []string
	arg             string
	isErrorExpected bool
	outputContains  string
	input           string
}

func getTestCommands(taskFilePath string) []testCommand {
	commands := []testCommand{
		{
			name:           "task disable",
			subcmd:         []string{"task", "disable"},
			outputContains: "disable complete!",
			arg:            dbTaskName,
		},
		{
			name:           "task enable",
			subcmd:         []string{"task", "enable"},
			outputContains: "enable complete!",
			arg:            dbTaskName,
		},
		{
			name:            "task delete",
			subcmd:          []string{"task", "delete", "-auto-approve"},
			outputContains:  "404 Not Found:",
			arg:             "task-no-exist",
			isErrorExpected: true, // Accept error since we can only delete once, just want to test a connection was successful
		},
		{
			name:           "task create",
			subcmd:         []string{"task", "create"},
			outputContains: "Creating the task will perform the actions described above.",
			input:          "no\n",
			arg:            fmt.Sprintf("--task-file=%s", taskFilePath),
		},
	}

	return commands
}
