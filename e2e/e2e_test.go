//go:build e2e
// +build e2e

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

// TestE2EBasic runs the CTS binary in daemon mode with a configuration with 2
// tasks and a test module that writes IP addresses to disk. Tests that CTS:
// 1. executes the 2 tasks upon startup
// 2. correct module resources are created for services ("api", "web", "db")
// 3. verifies Terraform statefiles are written to Consul KV, the default
//    Terraformfor backend for CTS for each task.
// 4. Consul catalog changes trigger correct tasks
func TestE2EBasic(t *testing.T) {
	// Note: no t.Parallel() for this particular test. Choosing this test to run 'first'
	// since e2e test running simultaneously will download Terraform into shared
	// directory causes some flakiness. All other e2e tests, should have t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "basic")
	delete := testutils.MakeTempDir(t, tempDir)
	// no defer to delete directory: only delete at end of test if no errors

	configPath := filepath.Join(tempDir, configFile)
	config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock().
		appendDBTask().appendWebTask()
	config.write(t, configPath)

	// Start CTS and wait for once mode to complete before verifying
	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)
	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)

	dbResourcesPath := filepath.Join(tempDir, dbTaskName, resourcesDir)
	webResourcesPath := filepath.Join(tempDir, webTaskName, resourcesDir)

	files := testutils.CheckDir(t, true, dbResourcesPath)
	require.Equal(t, 2, len(files))

	files = testutils.CheckDir(t, true, webResourcesPath)
	require.Equal(t, 2, len(files))

	contents := testutils.CheckFile(t, true, dbResourcesPath, "api.txt")
	require.Equal(t, "1.2.3.4", string(contents))

	contents = testutils.CheckFile(t, true, webResourcesPath, "api.txt")
	require.Equal(t, "1.2.3.4", string(contents))

	contents = testutils.CheckFile(t, true, webResourcesPath, "web.txt")
	require.Equal(t, "5.6.7.8", string(contents))

	contents = testutils.CheckFile(t, true, dbResourcesPath, "db.txt")
	require.Equal(t, "10.10.10.10", string(contents))

	// check statefiles exist
	testutils.CheckStateFile(t, srv.HTTPAddr, dbTaskName)
	testutils.CheckStateFile(t, srv.HTTPAddr, webTaskName)

	// Make Consul catalog changes to trigger CTS tasks then verify
	now := time.Now()
	service := testutil.TestService{ID: "web-1", Name: "web", Address: "5.5.5.5"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, webTaskName, now, defaultWaitForAPI)

	contents = testutils.CheckFile(t, true, webResourcesPath, "web-1.txt")
	assert.Equal(t, service.Address, string(contents), "web-1 should be created after registering")

	now = time.Now()
	testutils.DeregisterConsulService(t, srv, service.ID)
	api.WaitForEvent(t, cts, webTaskName, now, defaultWaitForAPI)

	// web-1 should be removed after deregistering
	testutils.CheckFile(t, false, webResourcesPath, "web-1.txt")

	delete()
}

// TestE2ERestart runs the CTS binary in daemon mode and tests restarting
// CTS results in no errors and can continue running based on the same config
// and Consul storing state.
func TestE2ERestart(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "restart")
	delete := testutils.MakeTempDir(t, tempDir)
	// no defer to delete directory: only delete at end of test if no errors

	configPath := filepath.Join(tempDir, configFile)
	config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock().appendDBTask()
	config.write(t, configPath)

	runSyncStop(t, configPath, defaultWaitForAPI)

	// rerun sync. confirm no errors e.g. recreating workspaces
	runSyncStop(t, configPath, defaultWaitForAPI)

	delete()
}

// TestE2ERestartConsul tests CTS is able to reconnect to Consul after the
// Consul agent had restarted, and CTS resumes monitoring changes to the
// Consul catalog.
func TestE2ERestartConsul(t *testing.T) {
	t.Parallel()

	consul := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "restart_consul")
	cleanup := testutils.MakeTempDir(t, tempDir) // cleanup at end if no errors

	configPath := filepath.Join(tempDir, configFile)
	config := baseConfig(tempDir).appendConsulBlock(consul).
		appendTerraformBlock().appendDBTask()
	config.write(t, configPath)

	// start CTS
	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)
	// wait enough for cts to cycle through once-mode successfully
	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)

	// stop Consul
	consul.Stop()
	time.Sleep(2 * time.Second)

	// restart Consul
	consul = testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
		PortHTTPS:    consul.Config.Ports.HTTPS,
	})
	defer consul.Stop()
	time.Sleep(5 * time.Second)

	// register a new service
	now := time.Now()
	apiInstance := testutil.TestService{ID: "api_new", Name: "api"}
	testutils.RegisterConsulService(t, consul, apiInstance, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, dbTaskName, now, defaultWaitForEvent)

	// confirm that CTS reconnected with Consul and created resource for latest service
	resourcesPath := filepath.Join(tempDir, dbTaskName, resourcesDir)
	testutils.CheckFile(t, true, resourcesPath, "api_new.txt")

	cleanup()
}

// TestE2EPanosHandlerError tests that CTS stops upon an error for a task with
// invalid PANOS credentials.
func TestE2EPanosHandlerError(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "panos_handler")
	delete := testutils.MakeTempDir(t, tempDir)
	// no defer to delete directory: only delete at end of test if no errors

	requiredProviders := `required_providers {
  panos = {
    source = "paloaltonetworks/panos"
  }
}
`
	configPath := filepath.Join(tempDir, configFile)
	config := panosBadCredConfig().appendConsulBlock(srv).
		appendTerraformBlock(tempDir, requiredProviders)
	config.write(t, configPath)

	api.StartCTS(t, configPath, api.CTSOnceModeFlag)

	delete()
}

// TestE2ELocalBackend tests CTS configured with the Terraform driver using
// the local backend.
func TestE2ELocalBackend(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		tempDirPrefix    string
		backendConfig    string
		dbStateFilePath  string
		webStateFilePath string
	}{
		{
			"no parameters configured",
			"local_backend_default",
			`backend "local" {}`,
			fmt.Sprintf("tmp_local_backend_default/%s/terraform.tfstate.d/%s",
				dbTaskName, dbTaskName),
			fmt.Sprintf("tmp_local_backend_default/%s/terraform.tfstate.d/%s",
				webTaskName, webTaskName),
		},
		{
			"workspace_dir configured",
			"local_backend_ws_dir",
			`backend "local" {
				workspace_dir = "test-workspace"
			}`,
			fmt.Sprintf("tmp_local_backend_ws_dir/%s/test-workspace/%s",
				dbTaskName, dbTaskName),
			fmt.Sprintf("tmp_local_backend_ws_dir/%s/test-workspace/%s",
				webTaskName, webTaskName),
		},
		{
			"workspace_dir configured with tasks sharing a workspace dir",
			"local_backend_shared_ws_dir",
			`backend "local" {
				workspace_dir = "../shared-workspace"
			}`,
			fmt.Sprintf("tmp_local_backend_shared_ws_dir/shared-workspace/%s",
				dbTaskName),
			fmt.Sprintf("tmp_local_backend_shared_ws_dir/shared-workspace/%s",
				webTaskName),
		},
		{
			"path configured",
			"local_backend_path",
			`backend "local" {
				# Setting path is meaningless in Sync. TF only uses it for
				# default workspace; Sync only uses non-default workspaces. This
				# value is overridden by the workspace directory for non-default
				# workspaces.
				path = "this-will-be-replaced-by-default-dir/terraform.tfstate"
			}`,
			fmt.Sprintf("tmp_local_backend_path/%s/terraform.tfstate.d/%s",
				dbTaskName, dbTaskName),
			fmt.Sprintf("tmp_local_backend_path/%s/terraform.tfstate.d/%s",
				webTaskName, webTaskName),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestConsulServer(t)
			defer srv.Stop()

			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, tc.tempDirPrefix)
			delete := testutils.MakeTempDir(t, tempDir)
			// no defer to delete directory: only delete at end of test if no errors

			config := baseConfig(tempDir).appendConsulBlock(srv).
				appendTerraformBlock(tc.backendConfig).
				appendDBTask().appendWebTask()

			configPath := filepath.Join(tempDir, configFile)
			config.write(t, configPath)

			api.StartCTS(t, configPath, api.CTSOnceModeFlag)

			// check that statefile was created locally
			checkStateFileLocally(t, tc.dbStateFilePath)
			checkStateFileLocally(t, tc.webStateFilePath)

			delete()
		})
	}
}

func TestE2EValidateError(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "validate_errors")
	delete := testutils.MakeTempDir(t, tempDir)
	// no defer to delete directory: only delete at end of test if no errors

	configPath := filepath.Join(tempDir, configFile)
	taskName := "cts_error_task"
	conditionTask := fmt.Sprintf(`task {
	name = "%s"
	module = "./test_modules/incompatible_w_cts"
	services = ["api", "db"]
	condition "catalog-services" {
		regexp = "^api$|^db$"
		use_as_module_input = true
	}
}
`, taskName)

	config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock().
		appendString(conditionTask)
	config.write(t, configPath)
	cmd := exec.Command("consul-terraform-sync", fmt.Sprintf("--config-file=%s", configPath))
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	cmd.Run()
	assert.Contains(t, buf.String(), fmt.Sprintf(`module for task "%s" is missing the "services" variable`, taskName))
	require.Contains(t,
		buf.String(),
		fmt.Sprintf(`module for task "%s" is missing the "catalog_services" variable, add to module or set "use_as_module_input" to false`,
			taskName))
	delete()
}

// TestE2E_FilterStatus checks the behavior of including/excluding non-passing
// service instances. It runs Consul registered with a critical service instance
// and CTS in once-mode and checks the terraform.tfvars contents to see whats
// included/excluded. It checks the following behavior:
// 1. By default, CTS only includes passing service instances (checked by
//    confirming in terraform.tfvars)
// 2. CTS can include non-passing service instances through additional
//    configuration
func TestE2E_FilterStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		tmpDirSuffix string
		config       string
		checkTfvars  func(*testing.T, string)
	}{
		{
			"default config excludes non-passing service instances",
			"_default",
			`task {
				name = "%s"
				source = "./test_modules/null_resource"
				services = ["api", "unhealthy-service"]
			}
			`,
			func(t *testing.T, contents string) {
				assert.NotContains(t, contents, "unhealthy-service")
				assert.NotContains(t, contents, `= "critical"`)

				// confirm that healthy service is still included
				assert.Contains(t, contents, "api")
				assert.Contains(t, contents, `= "passing"`)
			},
		},
		{
			"services filter includes non-passing service instances",
			"_w_filter",
			`task {
				name = "%s"
				source = "./test_modules/null_resource"
				services = ["api", "unhealthy-service"]
			}
			service {
				name = "unhealthy-service"
				filter = "Checks.Status != \"\""
			}
			`,
			func(t *testing.T, contents string) {
				assert.Contains(t, contents, "unhealthy-service")
				assert.Contains(t, contents, `= "critical"`)

				// confirm that healthy service is still included
				assert.Contains(t, contents, "api")
				assert.Contains(t, contents, `= "passing"`)
			},
		},
	}

	srv := newTestConsulServer(t)
	defer srv.Stop()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := fmt.Sprintf("%s%s%s", tempDirPrefix, "filter_statuses", tc.tmpDirSuffix)
			delete := testutils.MakeTempDir(t, tempDir)

			taskName := "status_filter_task"

			configPath := filepath.Join(tempDir, configFile)
			config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock().
				appendString(fmt.Sprintf(tc.config, taskName))
			config.write(t, configPath)

			api.StartCTS(t, configPath, api.CTSOnceModeFlag)

			taskDir := filepath.Join(tempDir, taskName)
			contents := testutils.CheckFile(t, true, taskDir, "terraform.tfvars")

			tc.checkTfvars(t, contents)
			delete()
		})
	}
}

// TestE2EInspectMode tests running CTS in inspect mode and verifies that
// the plan is outputted and no changes are actually applied.
func TestE2EInspectMode(t *testing.T) {
	t.Parallel()
	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "inspect")
	delete := testutils.MakeTempDir(t, tempDir)
	defer delete()

	config := baseConfig(tempDir).appendConsulBlock(srv).
		appendTerraformBlock().appendWebTask()

	configPath := filepath.Join(tempDir, configFile)
	config.write(t, configPath)

	cmd := exec.Command("consul-terraform-sync", fmt.Sprintf("--config-file=%s", configPath),
		api.CTSInspectFlag)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	cmd.Run()

	assert.Contains(t, buf.String(), "Plan: 2 to add, 0 to change, 0 to destroy.")
	resourcePath := filepath.Join(tempDir, webTaskName, resourcesDir)
	validateServices(t, false, []string{"web", "api"}, resourcePath)
}
