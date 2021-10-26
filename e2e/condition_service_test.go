//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCondition_Services_Regexp runs the CTS binary to test
// a task configured with a regexp in the service condition.
// This test confirms that when a service is registered that
// doesn't match the task condition's regexp config, no task
// is triggered.
func TestCondition_Services_Regexp(t *testing.T) {
	t.Parallel()

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "services_condition_regexp")
	taskName := "services_condition_task"
	conditionTask := fmt.Sprintf(`task {
	name = "%s"
	source = "./test_modules/local_instances_file"
	condition "services" {
		regexp = "api-"
	}
}
`, taskName)

	cts := ctsSetup(t, srv, tempDir, conditionTask)

	// Test that regex filter is filtering service registration information and
	// task triggers
	// 0. Confirm baseline: check current number of events and that that the
	//    services variable contains no service information
	// 1. Register db service instance. Confirm that the task was not triggered
	//    (no new event) and its data is filtered out of services variable.
	// 2. Register api-web service instance. Confirm that task was triggered
	//    (one new event) and its data exists in the services variable.
	// 3. Register a second node to the api-web service. Confirm that task was triggered
	//    (one new event) and its data exists in the services variable.

	// 0. Confirm only one event. Confirm empty var catalog_services
	eventCountExpected := eventCount(t, taskName, cts.Port())
	require.Equal(t, 1, eventCountExpected)

	workingDir := fmt.Sprintf("%s/%s", tempDir, taskName)
	content := testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "services = {\n}")

	// 1. Register a filtered out service "db"
	service := testutil.TestService{ID: "db-1", Name: "db"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	time.Sleep(defaultWaitForNoEvent)

	eventCountNow := eventCount(t, taskName, cts.Port())
	require.Equal(t, eventCountExpected, eventCountNow,
		"change in event count. task was unexpectedly triggered")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "services = {\n}")

	// 2. Register a matched service "api-web"
	now := time.Now()
	service = testutil.TestService{ID: "api-web-1", Name: "api-web"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)
	eventCountNow = eventCount(t, taskName, cts.Port())
	eventCountExpected++
	require.Equal(t, eventCountExpected, eventCountNow,
		"event count did not increment once. task was not triggered as expected")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, `"api-web"`)
	assert.Contains(t, content, `"api-web-1"`)

	// 3. Add a second node to the service "api-web"
	now = time.Now()
	service = testutil.TestService{ID: "api-web-2", Name: "api-web"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)
	eventCountNow = eventCount(t, taskName, cts.Port())
	eventCountExpected++
	require.Equal(t, eventCountExpected, eventCountNow,
		"event count did not increment once. task was not triggered as expected")
	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, `"api-web-2"`)
}
