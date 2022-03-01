//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/event"
	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// tempDirPrefix is the prefix for the directory for a given e2e test
	// where files generated from e2e are stored. This directory is
	// destroyed after e2e testing if no errors.
	tempDirPrefix = "tmp_"

	// resourcesDir is the sub-directory of tempDir where the
	// Terraform resources created from running consul-terraform-sync are stored
	resourcesDir = "resources"

	// configFile is the name of the sync config file
	configFile = "config.hcl"

	// liberal default times to wait
	defaultWaitForRegistration = 8 * time.Second
	defaultWaitForEvent        = 15 * time.Second
	defaultWaitForAPI          = 30 * time.Second

	// liberal wait time to ensure event doesn't happen
	defaultWaitForNoEvent = 6 * time.Second

	// default TLS
	defaultCTSClientCert = "../testutils/certs/localhost_leaf_cert.pem"
	defaultCTSClientKey  = "../testutils/certs/localhost_leaf_key.pem"
	defaultCTSCACert     = "../testutils/certs/localhost_cert.pem"
	defaultCertDir       = "certs"

	alternateCACert = "../testutils/certs/localhost_cert3.pem"
	alternateCert   = "../testutils/certs/localhost_cert3.pem"
	alternateKey    = "../testutils/certs/localhost_key3.pem"
)

var localFlag = flag.Bool("local", false, "flag for running E2E tests locally")

type tlsConfig struct {
	clientCert     string
	clientKey      string
	caCert         string
	caPath         string
	verifyIncoming *bool
}

func defaultTLSConfig() tlsConfig {
	return tlsConfig{
		clientCert: defaultCTSClientCert,
		clientKey:  defaultCTSClientKey,
	}
}

func tlsConfigWithCAPath(tempDir string) tlsConfig {
	return tlsConfig{
		caPath:     filepath.Join(tempDir, defaultCertDir),
		clientCert: defaultCTSClientCert,
		clientKey:  defaultCTSClientKey,
	}
}

func defaultMTLSConfig() tlsConfig {
	verifyIncoming := true
	return tlsConfig{
		clientCert:     defaultCTSClientCert,
		clientKey:      defaultCTSClientKey,
		caCert:         defaultCTSCACert,
		verifyIncoming: &verifyIncoming,
	}
}

func mtlsConfigWithCAPath(tempDir string) tlsConfig {
	verifyIncoming := true
	return tlsConfig{
		clientCert:     defaultCTSClientCert,
		clientKey:      defaultCTSClientKey,
		caPath:         filepath.Join(tempDir, defaultCertDir),
		verifyIncoming: &verifyIncoming,
	}
}

func newTestConsulServer(t *testing.T) *testutil.TestServer {
	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})

	// Register services
	srv.AddAddressableService(t, "api", testutil.HealthPassing,
		"1.2.3.4", 8080, []string{"tag1"})
	srv.AddAddressableService(t, "web", testutil.HealthPassing,
		"5.6.7.8", 8000, []string{"tag2"})
	srv.AddAddressableService(t, "db", testutil.HealthPassing,
		"10.10.10.10", 8000, []string{"tag3", "tag4"})
	srv.AddAddressableService(t, "unhealthy-service", testutil.HealthCritical,
		"1.2.3.4", 8080, []string{})

	return srv
}

func runSyncStop(t *testing.T, configPath string, dur time.Duration) {
	cts, stop := api.StartCTS(t, configPath)
	err := cts.WaitForAPI(dur)
	require.NoError(t, err)
	stop(t)
}

// checkStateFileLocally checks if statefile exists
func checkStateFileLocally(t *testing.T, stateFilePath string) {
	files := testutils.CheckDir(t, true, stateFilePath)
	require.Equal(t, 1, len(files))

	stateFile := files[0]
	require.Equal(t, "terraform.tfstate", stateFile.Name())
}

// runSubcommand runs a CTS subcommand and its arguments. If user input is
// required for subcommand, pass it through 'input' parameter. Function returns
// the stdout/err output and any error when executing the subcommand.
// Note: if error returned, output will still contain any stdout/err information.
func runSubcommand(t *testing.T, input string, subcmd ...string) (string, error) {
	return runSubCommandWithEnvVars(t, input, []string{}, subcmd...)
}

func runSubCommandWithEnvVars(t *testing.T, input string, envVars []string, subcmd ...string) (string, error) {
	cmd := exec.Command("consul-terraform-sync", subcmd...)
	cmd.Env = append(cmd.Env, envVars...)

	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b

	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)

	err = cmd.Start()
	require.NoError(t, err)

	_, err = stdin.Write([]byte(input))
	require.NoError(t, err)
	err = stdin.Close()
	require.NoError(t, err)

	err = cmd.Wait()
	return b.String(), err
}

// ctsSetup executes the following setup steps:
// 1. Creates a temporary working directory,
// 2. Creates a CTS configuration file with the provided task
// 3. Starts CTS
// 4. Waits for the CTS API to start without error, indicating that all initialization is complete
func ctsSetup(t *testing.T, srv *testutil.TestServer, tempDir string, taskConfig ...string) *api.Client {
	cleanup := testutils.MakeTempDir(t, tempDir)
	t.Cleanup(func() {
		_ = cleanup()
	})

	config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock()
	for _, c := range taskConfig {
		config = config.appendString(c)
	}
	configPath := filepath.Join(tempDir, configFile)
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath)
	t.Cleanup(func() {
		stop(t)
	})

	err := cts.WaitForAPI(defaultWaitForAPI * time.Duration(len(taskConfig)))
	require.NoError(t, err)

	return cts
}

func ctsSetupTLS(t *testing.T, srv *testutil.TestServer, tempDir string, taskConfig string, tlsConfig tlsConfig) *api.Client {
	cleanup := testutils.MakeTempDir(t, tempDir)

	// Setup CA path directory if provided
	var cleanupCAPath func() error
	if len(tlsConfig.caPath) != 0 {
		// Create CA Path directory
		cleanupCAPath = testutils.MakeTempDir(t, tlsConfig.caPath)

		// Get all test certs and move them to the CA path directory
		certs := []string{
			defaultCTSCACert,
			defaultCTSClientCert,
			alternateCACert,
		}
		testutils.CopyFiles(t, certs, tlsConfig.caPath)
		t.Cleanup(func() {
			err := cleanupCAPath()
			require.NoError(t, err)
		})
	}

	// add cleanup of tempDir after CAPath directory since CAPath directory will be inside temp directory
	t.Cleanup(func() {
		_ = cleanup()
	})

	config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock().
		appendString(taskConfig).appendTLSBlock(tlsConfig)
	configPath := filepath.Join(tempDir, configFile)
	config.write(t, configPath)

	cts, stop := api.StartCTSSecure(t, configPath, api.TLSConfig{
		CACert:     tlsConfig.caCert,
		CAPath:     tlsConfig.caPath,
		ClientCert: tlsConfig.clientCert,
		ClientKey:  tlsConfig.clientKey,
		SSLVerify:  false,
	})
	t.Cleanup(func() {
		stop(t)
	})

	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)

	return cts
}

// eventCount returns number of events that are stored for a given task by
// querying the Task Status API. Note: events have a storage limit (currently 5)
func eventCount(t *testing.T, taskName string, port int) int {
	events := events(t, taskName, port)
	return len(events)
}

// events returns the events that are stored for a given task by querying the
// Task Status API. Note: events have a storage limit (currently 5)
func events(t *testing.T, taskName string, port int) []event.Event {
	u := fmt.Sprintf("http://localhost:%d/%s/status/tasks/%s?include=events",
		port, "v1", taskName)
	resp := testutils.RequestHTTP(t, http.MethodGet, u, "")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var taskStatuses map[string]api.TaskStatus
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(&taskStatuses)
	require.NoError(t, err)

	taskStatus, ok := taskStatuses[taskName]
	require.True(t, ok, taskStatuses)
	return taskStatus.Events
}

// validateServices checks that files for each given service instance either exist or do not exist.
func validateServices(t *testing.T, expected bool, services []string, servicesPath string) {
	for _, service := range services {
		testutils.CheckFile(t, expected, servicesPath, fmt.Sprintf("%s.txt", service))
	}
}

// validateModuleFile checks whether a file dependent on a module input variable was created or not by the module.
// If the file exists, then the content of the file is checked as well. If use_as_module_input is set to false,
// then no file is expected to be created so no checks will be made. Assumes the file has a .txt extension.
//
// e.g., checking that the module created a file for a Consul KV entry where the filename is the key and the
// content is the value
func validateModuleFile(t *testing.T, useAsModuleInput, expected bool, resourcesPath, name, expectedContent string) {
	if !useAsModuleInput {
		// module will not generate files based on the module input variables,
		// nothing to validate
		return
	}
	content := testutils.CheckFile(t, expected, resourcesPath, fmt.Sprintf("%s.txt", name))
	if expected {
		assert.Equal(t, expectedContent, content)
	}
}

// validateVariable checks whether a variable in the .tfvars file contains or does not contain
// a specified value. Will fail if the variable does not exist.
func validateVariable(t *testing.T, contains bool, workingDir, name, value string) {
	content := testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)

	// Split the variable, assumes two new lines are only between variables
	vars := strings.Split(content, "\n\n")

	// Check individual variables, skipping preamble comment
	for _, v := range vars[1:] {
		v = strings.TrimLeft(v, "\n")
		if strings.HasPrefix(v, name) {
			if contains {
				assert.Contains(t, v, value)
				return
			} else {
				assert.NotContains(t, v, value)
				return
			}
		}
	}
	assert.Fail(t, fmt.Sprintf("variable '%s' not found in terraform.tfvars", name))
}

func setParallelism(t *testing.T) {
	if !*localFlag {
		t.Parallel()
	}
}
