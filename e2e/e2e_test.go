//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/command"
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
	// when run in the CI environment

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "basic")
	cleanup := testutils.MakeTempDir(t, tempDir)
	// no defer to delete directory: only delete at end of test if no errors

	configPath := filepath.Join(tempDir, configFile)
	config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock().
		appendDBTask().appendWebTask()
	config.write(t, configPath)

	// Start CTS and wait for once mode to complete before verifying
	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)
	err := cts.WaitForTestReadiness(defaultWaitForTestReadiness)
	require.NoError(t, err)

	dbResourcesPath := filepath.Join(tempDir, dbTaskName, resourcesDir)
	webResourcesPath := filepath.Join(tempDir, webTaskName, resourcesDir)

	files := testutils.CheckDir(t, true, dbResourcesPath)
	require.Equal(t, 2, len(files))

	files = testutils.CheckDir(t, true, webResourcesPath)
	require.Equal(t, 2, len(files))

	contents := testutils.CheckFile(t, true, dbResourcesPath, "api.txt")
	require.Equal(t, "1.2.3.4", contents)

	contents = testutils.CheckFile(t, true, webResourcesPath, "api.txt")
	require.Equal(t, "1.2.3.4", contents)

	contents = testutils.CheckFile(t, true, webResourcesPath, "web.txt")
	require.Equal(t, "5.6.7.8", contents)

	contents = testutils.CheckFile(t, true, dbResourcesPath, "db.txt")
	require.Equal(t, "10.10.10.10", contents)

	// check statefiles exist
	testutils.CheckStateFile(t, srv.HTTPAddr, dbTaskName)
	testutils.CheckStateFile(t, srv.HTTPAddr, webTaskName)

	// Make Consul catalog changes to trigger CTS tasks then verify
	now := time.Now()
	service := testutil.TestService{ID: "web-1", Name: "web", Address: "5.5.5.5"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, webTaskName, now, defaultWaitForTestReadiness)

	contents = testutils.CheckFile(t, true, webResourcesPath, "web-1.txt")
	assert.Equal(t, service.Address, contents, "web-1 should be created after registering")

	now = time.Now()
	testutils.DeregisterConsulService(t, srv, service.ID)
	api.WaitForEvent(t, cts, webTaskName, now, defaultWaitForTestReadiness)

	// web-1 should be removed after deregistering
	testutils.CheckFile(t, false, webResourcesPath, "web-1.txt")

	_ = cleanup()
}

// TestE2EArgumentParsing runs the CTS binary and checks to ensure that
// argument parsing works properly for both `=` and space-delimited argument values.
// For example, both `--a=b` and `--a b` should be acceptable ways to specify args.
func TestE2EArgumentParsing(t *testing.T) {
	setParallelism(t)
	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "argument_parsing")
	cleanupTemp := testutils.MakeTempDir(t, tempDir)
	defer cleanupTemp()

	// Create an extra, empty directory that will be searched for config files.
	// This is done because the `StartCTS` function always includes its own
	// argument of `-config-file=` and we cannot reference the same files twice.
	emptyDir := fmt.Sprintf("%s%s", tempDirPrefix, "argument_parsing_emptydir")
	cleanupEmpty := testutils.MakeTempDir(t, emptyDir)
	defer cleanupEmpty()

	configPath := filepath.Join(tempDir, configFile)
	config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock().
		appendDBTask().appendWebTask()
	config.write(t, configPath)

	// Execute CTS with these extra parameters to ensure they parse correctly. For example:
	testCases := [][]string{
		{"start", "-config-dir", emptyDir, "-config-dir=" + emptyDir},
		// TODO remove this line after the deprecated "default" implied subcommand is no longer supported.
		{"-config-dir", emptyDir, "-config-dir=" + emptyDir},
	}

	for _, args := range testCases {
		cts, stop := api.StartCTS(t, configPath, args...)
		err := cts.WaitForTestReadiness(defaultWaitForTestReadiness)
		assert.NoError(t, err, "Execution failed for arguments: %v", args)
		stop(t)
	}
}

// TestE2ERestart runs the CTS binary in daemon mode and tests restarting
// CTS results in no errors and can continue running based on the same config
// and Consul storing state.
func TestE2ERestart(t *testing.T) {
	setParallelism(t)

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "restart")
	cleanup := testutils.MakeTempDir(t, tempDir)
	// no defer to delete directory: only delete at end of test if no errors

	configPath := filepath.Join(tempDir, configFile)
	config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock().appendDBTask()
	config.write(t, configPath)

	runSyncStop(t, configPath, defaultWaitForTestReadiness)

	// rerun sync. confirm no errors e.g. recreating workspaces
	runSyncStop(t, configPath, defaultWaitForTestReadiness)

	_ = cleanup()
}

// TestE2ERestartConsul tests CTS is able to reconnect to Consul after the
// Consul agent had restarted, and CTS resumes monitoring changes to the
// Consul catalog.
func TestE2ERestartConsul(t *testing.T) {
	setParallelism(t)

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
	err := cts.WaitForTestReadiness(defaultWaitForTestReadiness)
	require.NoError(t, err)

	// stop Consul
	err = consul.Stop()
	// When Consul is killed with a SIGINT, it exists with error code 1, this error is expected
	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, 1, exitErr.ExitCode())

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

	_ = cleanup()
}

// TestE2ELocalBackend tests CTS configured with the Terraform driver using
// the local backend.
func TestE2ELocalBackend(t *testing.T) {
	setParallelism(t)

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
			cleanup := testutils.MakeTempDir(t, tempDir)
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

			_ = cleanup()
		})
	}
}

func TestE2EValidateError(t *testing.T) {
	setParallelism(t)

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "validate_errors")
	cleanup := testutils.MakeTempDir(t, tempDir)
	// no defer to delete directory: only delete at end of test if no errors

	configPath := filepath.Join(tempDir, configFile)
	taskName := "cts_error_task"
	conditionTask := fmt.Sprintf(`task {
	name = "%s"
	module = "./test_modules/incompatible_w_cts"
	condition "catalog-services" {
		regexp = "^api$|^db$"
		use_as_module_input = true
	}
}
`, taskName)

	config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock().
		appendString(conditionTask)
	config.write(t, configPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, "consul-terraform-sync", fmt.Sprintf("--config-file=%s", configPath))
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	errCh := make(chan error)
	go func() {
		if err := cmd.Run(); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		require.Error(t, err)
	case <-time.After(defaultWaitForTestReadiness):
		t.Log(buf.String())
		t.Fatal(fmt.Sprintf("TestE2EValidateError should not take more than %s",
			defaultWaitForTestReadiness))
	}

	assert.Contains(t, buf.String(), fmt.Sprintf(`module for task "%s" is missing the "services" variable`, taskName))
	require.Contains(t,
		buf.String(),
		fmt.Sprintf(`module for task "%s" is missing the "catalog_services" variable, add to module or set "use_as_module_input" to false`,
			taskName))
	_ = cleanup()
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
	setParallelism(t)

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
				module = "./test_modules/null_resource"
				condition "services" {
					names = ["api", "unhealthy-service"]
				}
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
				module = "./test_modules/null_resource"
				condition "services" {
					names = ["api", "unhealthy-service"]
					filter = "Checks.Status != \"\""
				}
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
			cleanup := testutils.MakeTempDir(t, tempDir)

			taskName := "status_filter_task"

			configPath := filepath.Join(tempDir, configFile)
			config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock().
				appendString(fmt.Sprintf(tc.config, taskName))
			config.write(t, configPath)

			api.StartCTS(t, configPath, api.CTSOnceModeFlag)

			taskDir := filepath.Join(tempDir, taskName)
			contents := testutils.CheckFile(t, true, taskDir, "terraform.tfvars")

			tc.checkTfvars(t, contents)
			_ = cleanup()
		})
	}
}

// TestE2EInspectMode tests running CTS in inspect mode and verifies that
// the plan is outputted and no changes are actually applied.
func TestE2EInspectMode(t *testing.T) {
	setParallelism(t)
	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "inspect")
	cleanup := testutils.MakeTempDir(t, tempDir)
	defer cleanup()

	config := baseConfig(tempDir).appendConsulBlock(srv).
		appendTerraformBlock().appendWebTask()

	configPath := filepath.Join(tempDir, configFile)
	config.write(t, configPath)

	cmd := exec.Command("consul-terraform-sync", fmt.Sprintf("--config-file=%s", configPath),
		api.CTSInspectFlag)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	require.NoError(t, err)

	// Confirm shutdown
	assert.Contains(t, buf.String(), "graceful shutdown")

	// Confirm plan outputted
	assert.Contains(t, buf.String(), "Plan: 2 to add, 0 to change, 0 to destroy.")

	// Confirm resources not created
	resourcePath := filepath.Join(tempDir, webTaskName, resourcesDir)
	validateServices(t, false, []string{"web", "api"}, resourcePath)
}

// TestE2E_OnceMode tests running CTS in once mode and verifies that
// the each task is run once and that CTS exits.
// Tests by using two tasks:
// - First run a scheduled task that runs every 1s
// - Then run a delayed task that takes 5s to execute
// - Confirm that scheduled task didn't run again while waiting on delayed task
func TestE2E_OnceMode(t *testing.T) {
	setParallelism(t)

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "once-mode")
	cleanup := testutils.MakeTempDir(t, tempDir)
	defer cleanup()

	// Task will run every 1 second. "a" prefix to ensure alphabetically 1st
	scheduledTaskName := "a_scheduled_task"
	scheduledTask := fmt.Sprintf(`task {
		name = "%s"
		module = "./test_modules/local_instances_file"
		condition "schedule" {
			cron = "*/1 * * * * * *"
		}
		module_input "services"{
			names = ["web", "api"]
		}
	}`, scheduledTaskName)

	// Task will take 5 seconds to execute. "b" prefix to ensure alphabetically 2nd
	delayedTaskName := "b_delayed_task"
	delayedTask := moduleTaskConfig(delayedTaskName, "./test_modules/delayed_module")

	config := baseConfig(tempDir).appendConsulBlock(srv).
		appendTerraformBlock().appendString(scheduledTask).appendString(delayedTask)

	configPath := filepath.Join(tempDir, configFile)
	config.write(t, configPath)

	cmd := exec.Command("consul-terraform-sync", fmt.Sprintf("--config-file=%s", configPath),
		api.CTSOnceModeFlag)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()

	output := buf.String()
	defer func() {
		if t.Failed() {
			t.Log("Log output:\n", output)
		}
	}()

	require.NoError(t, err, output)

	// Confirm shutdown
	assert.Contains(t, output, "graceful shutdown", output)

	// Confirm scheduled task ran before delayed task
	completedStr := "task completed: task_name=%s"
	schedIx := strings.Index(output, fmt.Sprintf(completedStr, scheduledTaskName))
	delayedIx := strings.Index(output, fmt.Sprintf(completedStr, delayedTaskName))
	assert.Less(t, schedIx, delayedIx, "delayed task ran before scheduled task")

	for _, taskName := range []string{delayedTaskName, scheduledTaskName} {
		// Confirm each task ran only once
		taskRunCount := strings.Count(output, fmt.Sprintf(completedStr, taskName))
		assert.Equal(t, 1, taskRunCount)

		// Confirm resources are created for each task
		resourcePath := filepath.Join(tempDir, taskName, resourcesDir)
		validateServices(t, true, []string{"web", "api"}, resourcePath)
	}
}

// TestE2E_ConfigStreamlining_Deprecations runs the CTS binary in once mode to test a CTS config
// with v0.5 config streamlining deprecations. This test confirms that the old
// deprecated fields still work in v0.5.0 until removal.
//
// Deprecations to remove in v0.8.0
//  - "source_input" => "module_input"
//  - "source_includes_var" => "use_as_module_input"
//
// Deprecations to remove in a future major release after v0.8.0
//  - "source" => "module"
//  - "services" => "condition "services"" or "module_input "services""
//  - "service" block => "condition "services"" or "module_input "services""
func TestE2E_ConfigStreamlining_Deprecations(t *testing.T) {
	setParallelism(t)

	srv := newTestConsulServer(t)
	defer srv.Stop()
	srv.SetKVString(t, "key", "value")
	srv.AddAddressableService(t, "web", testutil.HealthPassing, "1.2.3.4", 8080, []string{""})

	removeAfterTaskName := "remove-after-0-8"
	removeAfterConfig := fmt.Sprintf(`
	task {
		name = "%s"
		description = "source, services, service deprecation"
		source = "./test_modules/local_instances_file"
		services = ["web"]
	}
	service {
		name = "web"
		cts_user_defined_meta {
			my_meta_key = "my_meta_value"
		}
	}
	`, removeAfterTaskName)

	removeInTaskName := "remove-in-0-8"
	removeInConfig := fmt.Sprintf(`
	task {
		name = "%s"
		description = "source_includes_var & source_input deprecation"
		module = "./test_modules/consul_kv_file"

		condition "consul-kv" {
			path = "key"
			source_includes_var = true
		}

		source_input "services" {
			names = ["api"]
		}
	}
	`, removeInTaskName)

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "config_streamlining")

	cts := ctsSetup(t, srv, tempDir, removeAfterConfig+removeInConfig)

	// check resources for deprecations to be removed after 0.8
	workingDir := filepath.Join(tempDir, removeAfterTaskName)
	resourcesPath := filepath.Join(workingDir, resourcesDir)
	validateServices(t, true, []string{"web"}, resourcesPath)
	validateVariable(t, true, workingDir, "services", "meta_value")

	// check services deprecation (to be removed after 0.8) handled correctly in Get API Condition
	u := fmt.Sprintf("http://localhost:%d/%s/tasks/%s", cts.Port(), "v1", removeAfterTaskName)
	resp := testutils.RequestHTTP(t, http.MethodGet, u, "")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var r oapigen.TaskResponse
	err := json.NewDecoder(resp.Body).Decode(&r)
	require.NoError(t, err)
	require.NotNil(t, r.Task)
	require.NotNil(t, r.Task.Condition.Services)
	require.NotNil(t, r.Task.Condition.Services.Names)
	assert.Contains(t, "[web]", strings.Join(*r.Task.Condition.Services.Names, ""))

	// check resources for deprecations to be removed in 0.8
	resourcesPath = filepath.Join(tempDir, removeInTaskName, resourcesDir)
	validateModuleFile(t, true, true, resourcesPath, "key", "value")
	validateServices(t, true, []string{"api"}, resourcesPath)
}

// TestE2E_TriggerTaskWhenActive tests making changes to monitored dependencies
// while a task is running. It verifies:
//
// 1. a change to the same task will trigger the task after the run completes
// 2. a change to a different task runs immediately
// 3. a change after the task completes will still trigger the task to run
//
// https://github.com/hashicorp/consul-terraform-sync/issues/732
func TestE2E_TriggerTaskWhenActive(t *testing.T) {
	setParallelism(t)
	// Start CTS with multiple tasks
	activeTaskName := "active_task"
	activeTaskConfig := fmt.Sprintf(`
	task {
		name = "%s"
		providers = ["local"]
		module = "./test_modules/delayed_module"
		condition "services" {
			names = ["api"]
		}
	}
	`, activeTaskName)

	inactiveTaskName := "inactive_task"
	inactiveTaskConfig := fmt.Sprintf(`
	task {
		name = "%s"
		providers = ["local"]
		module = "./test_modules/local_instances_file"
		condition "services" {
			names = ["web"]
		}
	}
	`, inactiveTaskName)

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()
	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "trigger_when_active")
	cts := ctsSetup(t, srv, tempDir, activeTaskConfig, inactiveTaskConfig)

	// Trigger the task with the long-running module so that it becomes active
	firstRegister := time.Now()
	testutils.RegisterConsulService(t, srv,
		testutil.TestService{Name: "api", ID: "api-1"}, defaultWaitForRegistration)

	// Trigger the task again while still running
	time.Sleep(1 * time.Second) // waiting for task to start, module sleeps for 5 seconds
	activeRegister := time.Now()
	testutils.RegisterConsulService(t, srv,
		testutil.TestService{Name: "api", ID: "api-2"}, defaultWaitForRegistration)

	// Trigger the other task while the other task is running
	inactiveRegister := time.Now()
	testutils.RegisterConsulService(t, srv,
		testutil.TestService{Name: "web", ID: "web-1"}, defaultWaitForRegistration)

	// Other task should complete without waiting for first task to complete
	api.WaitForEvent(t, cts, inactiveTaskName, inactiveRegister, defaultWaitForEvent)
	count := eventCount(t, inactiveTaskName, cts.Port())
	assert.Equal(t, 2, count, "unexpected number of events")
	inactivePath := filepath.Join(tempDir, inactiveTaskName, resourcesDir)
	validateServices(t, true, []string{"web-1"}, inactivePath)

	// First run of task should eventually complete
	api.WaitForEvent(t, cts, activeTaskName, firstRegister, defaultWaitForEvent)
	count = eventCount(t, activeTaskName, cts.Port())
	assert.Equal(t, 2, count, "unexpected number of events")
	resourcesPath := filepath.Join(tempDir, activeTaskName, resourcesDir)
	validateServices(t, true, []string{"api-1"}, resourcesPath)

	// Second run of task should occur after first one completes
	api.WaitForEvent(t, cts, activeTaskName, activeRegister, defaultWaitForEvent)
	count = eventCount(t, activeTaskName, cts.Port())
	assert.Equal(t, 3, count, "unexpected number of events")
	validateServices(t, true, []string{"api-1", "api-2"}, resourcesPath)

	// Trigger first task again, check for event
	now := time.Now()
	testutils.RegisterConsulService(t, srv,
		testutil.TestService{Name: "api", ID: "api-3"}, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, activeTaskName, now, defaultWaitForEvent)
	count = eventCount(t, activeTaskName, cts.Port())
	assert.Equal(t, 4, count, "unexpected number of events")
	validateServices(t, true, []string{"api-1", "api-2", "api-3"}, resourcesPath)
}

// TestE2E_TriggerTaskWhenActiveMultipleTimes tests that after triggering
// an active task multiple times, the next run will render with the latest
// information from Consul. It also checks that there will only be one event
// even though there were multiple changes, as the template has already
// rendered with the latest information so there are no subsequent changes.
//
// Note: This does not use buffer period.
//
// https://github.com/hashicorp/consul-terraform-sync/issues/732
func TestE2E_TriggerTaskWhenActiveMultipleTimes(t *testing.T) {
	setParallelism(t)

	activeTaskName := "active_task"
	activeTaskConfig := fmt.Sprintf(`
	task {
		name = "%s"
		providers = ["local"]
		module = "./test_modules/delayed_module"
		condition "services" {
			names = ["api"]
		}
	}
	`, activeTaskName)

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()
	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "trigger_when_active_multiple")
	cts := ctsSetup(t, srv, tempDir, activeTaskConfig)

	// Trigger the task with the long-running module so that it becomes active
	firstRegister := time.Now()
	testutils.RegisterConsulService(t, srv,
		testutil.TestService{Name: "api", ID: "api-1"}, defaultWaitForRegistration)

	// Register a new instance while the task is still running
	time.Sleep(2 * time.Second) // waiting for task to start, module sleeps for 5 seconds
	activeRegister := time.Now()
	testutils.RegisterConsulService(t, srv,
		testutil.TestService{Name: "api", ID: "api-2"}, defaultWaitForRegistration)

	// Deregister the newly added instance
	testutils.DeregisterConsulService(t, srv, "api-2")

	// Register a different instance
	testutils.RegisterConsulService(t, srv,
		testutil.TestService{Name: "api", ID: "api-3"}, defaultWaitForRegistration)

	// First run of task should eventually complete
	api.WaitForEvent(t, cts, activeTaskName, firstRegister, defaultWaitForEvent)
	count := eventCount(t, activeTaskName, cts.Port())
	expectedCount := 2
	assert.Equal(t, expectedCount, count, "unexpected number of events")
	resourcesPath := filepath.Join(tempDir, activeTaskName, resourcesDir)
	validateServices(t, true, []string{"api-1"}, resourcesPath)
	validateServices(t, false, []string{"api-2", "api-3"}, resourcesPath)

	// Second run of task completes with latest services info
	api.WaitForEvent(t, cts, activeTaskName, activeRegister, defaultWaitForEvent)
	count = eventCount(t, activeTaskName, cts.Port())
	expectedCount++
	assert.Equal(t, expectedCount, count, "unexpected number of events")
	validateServices(t, true, []string{"api-1", "api-3"}, resourcesPath)
	validateServices(t, false, []string{"api-2"}, resourcesPath)

	// Subsequent triggers should not create an event
	time.Sleep(defaultWaitForNoEvent + (5 * time.Second)) // accounting for 5s delay in module
	count = eventCount(t, activeTaskName, cts.Port())
	assert.Equal(t, expectedCount, count, "unexpected number of events")
}

// testInvalidTaskConfig tests that task creation fails with the given task configuration.
// Creation is tested both at CTS startup and with the CLI create command.
func testInvalidTaskConfig(t *testing.T, testName, taskName, taskConfig, errMsg string) {
	// Create tasks at start up
	t.Run(testName, func(t *testing.T) {
		setParallelism(t)
		srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
			HTTPSRelPath: "../testutils",
		})
		defer srv.Stop()

		tempDir := fmt.Sprintf("%s%s", tempDirPrefix, taskName)
		cleanup := testutils.MakeTempDir(t, tempDir)
		t.Cleanup(func() {
			cleanup()
		})

		config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock().
			appendString(taskConfig)
		configPath := filepath.Join(tempDir, configFile)
		config.write(t, configPath)

		out, err := runSubcommand(t, "",
			fmt.Sprintf("-config-file=%s", configPath), "--once")

		require.Error(t, err)
		assert.Contains(t, out, errMsg)
	})

	// Create tasks via the CLI
	t.Run(testName+"_create_cli", func(t *testing.T) {
		setParallelism(t)
		srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
			HTTPSRelPath: "../testutils",
		})
		defer srv.Stop()

		tempDir := fmt.Sprintf("%s%s_cli", tempDirPrefix, taskName)
		cts := ctsSetup(t, srv, tempDir, dbTask())

		var c hclConfig
		c = c.appendString(taskConfig)
		taskFilePath := filepath.Join(tempDir, "task.hcl")
		c.write(t, taskFilePath)

		subcmd := []string{"task", "create",
			fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
			fmt.Sprintf("-task-file=%s", taskFilePath),
		}
		out, err := runSubcommand(t, "yes\n", subcmd...)
		require.Error(t, err)
		out = strings.ReplaceAll(out, "\n", " ") // word wrapping can cause new lines in error message
		assert.Contains(t, out, errMsg)

		// check that CTS binary is still running
		_, err = cts.Status().Overall()
		assert.NoError(t, err)

		// check that existing tasks are still monitored
		registerTime := time.Now()
		services := []testutil.TestService{{ID: "api-1", Name: "api"}}
		testutils.AddServices(t, srv, services)
		api.WaitForEvent(t, cts, dbTaskName, registerTime, defaultWaitForEvent)
	})
}
