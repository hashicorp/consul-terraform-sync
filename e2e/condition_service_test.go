//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
)

// TestCondition_Services runs the CTS binary to test a task
// configured with regexp/names in the service condition.
// This test confirms that when a service is registered that
// doesn't match the task condition's regexp/names config, no task
// is triggered.
func TestCondition_Services(t *testing.T) {
	setParallelism(t)

	regexpConfig := `task {
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

	namesConfig := `task {
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

	cases := []struct {
		name             string
		taskName         string
		taskConfig       string
		useAsModuleInput bool
	}{
		{
			"regexp & includes true",
			"services_cond_regexp_include",
			regexpConfig,
			true,
		},
		{
			"regexp & includes false",
			"services_cond_regexp_include_false",
			regexpConfig,
			false,
		},
		{
			"names & includes true",
			"services_cond_names_include",
			namesConfig,
			true,
		},
		{
			"names & includes false",
			"services_cond_names_include_false",
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

			// Test that regex filter is filtering service registration information and
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
			eventCountExpected := eventCount(t, tc.taskName, cts.Port())
			require.Equal(t, 1, eventCountExpected)

			workingDir := fmt.Sprintf("%s/%s", tempDir, tc.taskName)
			validateVariable(t, true, workingDir, "services", "{\n}")

			// 1. Register a filtered out service "db"
			service := testutil.TestService{ID: "db-1", Name: "db"}
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
		})
	}
}
