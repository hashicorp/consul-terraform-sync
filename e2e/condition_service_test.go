//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/command"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
)

const (
	regexpConfig = `task {
		name = "%s"
		module = "./test_modules/local_instances_file"
		condition "services" {
			regexp = "api-"
			filter = "Service.Tags not contains \"tag_a\""
			cts_user_defined_meta {
				my_meta_key = "my_meta_value"
			}
			use_as_module_input = %t
		}
	}`
	namesConfig = `task {
		name = "%s"
		module = "./test_modules/local_instances_file"
		condition "services" {
			names  = ["api-web"]
			filter = "Service.Tags not contains \"tag_a\""
			cts_user_defined_meta {
				my_meta_key = "my_meta_value"
			}
			use_as_module_input = %t
		}
	}`
)

type servicesConditionCase struct {
	name             string
	taskName         string
	taskConfig       string
	useAsModuleInput bool
}

type servicesConditionTest struct {
	servicesConditionCase
	srv     *testutil.TestServer
	cts     *api.Client
	tempDir string
}

// TestCondition_Services runs the CTS binary to test a task
// configured with regexp/names in the service condition.
// This test confirms that when a service is registered that
// doesn't match the task condition's regexp/names config, no task
// is triggered.
func TestCondition_Services(t *testing.T) {
	setParallelism(t)

	cases := []servicesConditionCase{
		{
			"regexp - use true",
			"services_cond_regexp_use",
			regexpConfig,
			true,
		},
		{
			"regexp - use false",
			"services_cond_regexp_use_false",
			regexpConfig,
			false,
		},
		{
			"names - use true",
			"services_cond_names_use",
			namesConfig,
			true,
		},
		{
			"names - use false",
			"services_cond_names_use_false",
			namesConfig,
			false,
		},
	}

	for _, tc := range cases {
		tc := tc // rebind tc into this lexical scope for parallel use
		t.Run(tc.name, func(t *testing.T) {
			setParallelism(t) // In the CI environment, run table tests in parallel as they can take a lot of time
			srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
				HTTPSRelPath: "../testutils",
			})
			defer srv.Stop()

			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, tc.taskName)

			config := fmt.Sprintf(tc.taskConfig, tc.taskName, tc.useAsModuleInput)
			cts := ctsSetup(t, srv, tempDir, config)

			testServicesCondition(t, servicesConditionTest{
				tc, srv, cts, tempDir,
			})
		})

		t.Run(tc.name+"_CreateCLI", func(t *testing.T) {
			setParallelism(t)
			srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
				HTTPSRelPath: "../testutils",
			})
			defer srv.Stop()

			tempDir := fmt.Sprintf("%s%s_cli", tempDirPrefix, tc.taskName)

			cts := ctsSetup(t, srv, tempDir, disabledTaskConfig())
			config := fmt.Sprintf(tc.taskConfig, tc.taskName, tc.useAsModuleInput)

			// Create task via the CLI
			var taskConfig hclConfig
			taskConfig = taskConfig.appendString(config)
			taskFilePath := filepath.Join(tempDir, "task.hcl")
			taskConfig.write(t, taskFilePath)

			subcmd := []string{"task", "create",
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-task-file=%s", taskFilePath),
			}
			out, err := runSubcommand(t, "yes\n", subcmd...)
			require.NoError(t, err, fmt.Sprintf("command '%s' failed:\n %s", subcmd, out))
			require.Contains(t, out, fmt.Sprintf("Task '%s' created", tc.taskName))

			testServicesCondition(t, servicesConditionTest{
				tc, srv, cts, tempDir,
			})
		})
	}
}

func testServicesCondition(t *testing.T, tc servicesConditionTest) {
	// Test that services condition is filtering service registration information and
	// task triggers
	// 0. Confirm baseline: check current number of events and that that the
	//    services variable contains no service information
	// 1. Register db service instance. Confirm that the task was not triggered
	//    (no new event) and its data is filtered out of services variable.
	// 2. Register api-web service instance. Confirm that task was triggered
	//    (one new event). Confirm data exists in the services variable and
	//    confirm metadata in tfvars (use_as_module_input=true only).
	// 3. Register a second node to the api-web service. Confirm that
	//    task was triggered (one new event). Confirm data exists in
	//    the services variable (use_as_module_input=true only).
	// 4. Register a third node to the api-web service with tag_a. Confirm that
	//    task was not triggered and the data is not in the services variable.

	// 0. Confirm only one event. Confirm empty var catalog_services
	cts := tc.cts
	eventCountExpected := eventCount(t, tc.taskName, cts.Port())
	require.Equal(t, 1, eventCountExpected)

	workingDir := fmt.Sprintf("%s/%s", tc.tempDir, tc.taskName)
	validateVariable(t, true, workingDir, "services", "{\n}")

	// 1. Register a filtered out service "db"
	service := testutil.TestService{ID: "db-1", Name: "db"}
	srv := tc.srv
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	time.Sleep(defaultWaitForNoEvent)

	eventCountNow := eventCount(t, tc.taskName, cts.Port())
	require.Equal(t, eventCountExpected, eventCountNow,
		"change in event count. task was unexpectedly triggered")

	validateVariable(t, true, workingDir, "services", "{\n}")

	// 2. Register a matched service "api-web"
	now := time.Now()
	service = testutil.TestService{ID: "api-web-1", Name: "api-web"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, tc.taskName, now, defaultWaitForEvent)
	eventCountNow = eventCount(t, tc.taskName, cts.Port())
	eventCountExpected++
	require.Equal(t, eventCountExpected, eventCountNow,
		"event count did not increment once. task was not triggered as expected")
	resourcesPath := filepath.Join(workingDir, resourcesDir)
	validateServices(t, tc.useAsModuleInput, []string{"api-web-1"}, resourcesPath)
	validateVariable(t, tc.useAsModuleInput, workingDir, "services", "my_meta_value")

	// 3. Add a second node to the service "api-web"
	now = time.Now()
	service = testutil.TestService{ID: "api-web-2", Name: "api-web"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, tc.taskName, now, defaultWaitForEvent)
	eventCountNow = eventCount(t, tc.taskName, cts.Port())
	eventCountExpected++
	require.Equal(t, eventCountExpected, eventCountNow,
		"event count did not increment once. task was not triggered as expected")
	validateServices(t, tc.useAsModuleInput, []string{"api-web-1", "api-web-2"}, resourcesPath)

	// 4. Register a matched service "api-web" with tag_a
	now = time.Now()
	service = testutil.TestService{ID: "api-web-3", Name: "api-web", Tags: []string{"tag_a"}}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	time.Sleep(defaultWaitForNoEvent)
	eventCountNow = eventCount(t, tc.taskName, cts.Port())
	require.Equal(t, eventCountExpected, eventCountNow,
		"change in event count. task was unexpectedly triggered")
	validateServices(t, false, []string{"api-web-3"}, resourcesPath)
}

func TestCondition_Services_InvalidQueries(t *testing.T) {
	setParallelism(t)
	config := `task {
		name = "%s"
		module = "./test_modules/null_resource"
		condition "services" {
			names  = ["api, web"]
			%s
		}
	}`

	cases := []struct {
		name        string
		queryConfig string
		errMsg      string
	}{
		{
			"datacenter",
			`datacenter = "foo"`,
			"No path to datacenter",
		},
		{
			"filter",
			`filter = "\"foo\" in Service.Bar"`,
			`Selector "Service.Bar" is not valid`,
		},
		{
			"namespace_with_oss_consul",
			`namespace = "foo"`,
			`Invalid query parameter: "ns"`,
		},
	}

	for _, tc := range cases {
		tc := tc // rebind tc into this lexical scope for parallel use
		taskName := "condition_services_invalid_" + tc.name
		taskConfig := fmt.Sprintf(config, taskName, tc.queryConfig)
		testInvalidTaskConfig(t, tc.name, taskName, taskConfig, tc.errMsg)
	}
}
