// +build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCondition_CatalogServices_Registration runs the CTS binary. It is a basic
// test for a task configured with a catalog service condition. This test
// confirms that the first instance of a service registering and the last
// instance of a service deregistering triggers this task. Note, this test also
// also covers ensuring that daemon-mode can pass through once-mode with no
// initial service registrations
func TestCondition_CatalogServices_Registration(t *testing.T) {
	t.Parallel()

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	taskName := "catalog_task"
	conditionTask := fmt.Sprintf(`task {
	name = "%s"
	services = ["api"]
	source = "./test_modules/local_tags_file"
	condition "catalog-services" {
		source_includes_var = true
	}
}
`, taskName)

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "cs_condition_registration")
	cleanup := testutils.MakeTempDir(t, tempDir)

	config := baseConfig().appendConsulBlock(srv).appendTerraformBlock(tempDir).
		appendString(conditionTask)
	configPath := filepath.Join(tempDir, configFile)
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)

	err := cts.WaitForAPI(15 * time.Second)
	require.NoError(t, err)

	// Test that task is triggered on service registration and deregistration
	// 0. Confirm baseline: nothing is registered so no resource created yet
	// 1. Register 'api' service. Confirm resource created
	// 2. Deregister 'api' service. Confirm resource destroyed

	// 0. Confirm resource not created
	resourcesPath := fmt.Sprintf("%s/%s", tempDir, resourcesDir)
	testutils.CheckFile(t, false, resourcesPath, "api_tags.txt")

	// 1. Register api, resource created
	service := testutil.TestService{ID: "api-1", Name: "api"}
	testutils.RegisterConsulService(t, srv, service, testutil.HealthPassing)
	time.Sleep(7 * time.Second)
	testutils.CheckFile(t, true, resourcesPath, "api_tags.txt")

	// 2. Deregister api, resource destroyed
	testutils.DeregisterConsulService(t, srv, "api-1")
	time.Sleep(7 * time.Second)
	testutils.CheckFile(t, false, resourcesPath, "api_tags.txt")

	cleanup()
}

// TestCondition_CatalogServices_NoServicesTrigger runs the CTS binary. It
// specifically tests a task configured with a catalog service condition. This
// test confirms that changes in service instances, which do not effect the
// overall service [de]registration, do not trigger the task.
func TestCondition_CatalogServices_NoServicesTrigger(t *testing.T) {
	t.Parallel()

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	service := testutil.TestService{ID: "api-1", Name: "api"}
	testutils.RegisterConsulService(t, srv, service, testutil.HealthPassing)
	time.Sleep(7 * time.Second)

	taskName := "catalog_task"
	conditionTask := fmt.Sprintf(`task {
	name = "%s"
	services = ["api", "db"]
	source = "./test_modules/local_tags_file"
	condition "catalog-services" {
		source_includes_var = true
	}
}
`, taskName)

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "cs_condition_no_services_trigger")
	cleanup := testutils.MakeTempDir(t, tempDir)

	config := baseConfig().appendConsulBlock(srv).appendTerraformBlock(tempDir).
		appendString(conditionTask)
	configPath := filepath.Join(tempDir, configFile)
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)

	err := cts.WaitForAPI(15 * time.Second)
	require.NoError(t, err)

	// Test that task is not triggered by service-instance specific changes and
	// only by service registration changes.
	// 0. Confirm baseline: check current number of events for initially
	//    registered api service. check for instance data in tfvars
	// 1. Register api-2 instance for existing api service. Confirm task was not
	//    triggered (no new event) and therefore api-2 not in tfvars
	// 2. Register new db service. Confirm task was triggered (new event) and
	//    db and api-2 (now) rendered in tfvars

	// 0. Confirm one event. Confirm initial api service registration data
	evenCountBase := eventCount(t, taskName, cts.Port())
	require.Equal(t, 1, evenCountBase)

	workingDir := fmt.Sprintf("%s/%s", tempDir, taskName)
	content := testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "api-1")

	// 1. Register second api service instance "api-2" (no trigger)
	service = testutil.TestService{ID: "api-2", Name: "api"}
	testutils.RegisterConsulService(t, srv, service, testutil.HealthPassing)
	time.Sleep(7 * time.Second)

	eventCountNow := eventCount(t, taskName, cts.Port())
	require.Equal(t, evenCountBase, eventCountNow,
		"change in event count. task was unexpectedly triggered")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.NotContains(t, content, "api-2")

	// 2. Register db service (trigger + render template)
	service = testutil.TestService{ID: "db-1", Name: "db"}
	testutils.RegisterConsulService(t, srv, service, testutil.HealthPassing)
	time.Sleep(7 * time.Second)

	eventCountNow = eventCount(t, taskName, cts.Port())
	require.Equal(t, evenCountBase+1, eventCountNow,
		"event count did not increment once. task was not triggered as expected")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "api-2")
	assert.Contains(t, content, "db-1")

	cleanup()
}

// TestCondition_CatalogServices_NoTagsTrigger runs the CTS binary. It
// specifically tests a task configured with a catalog service condition. This
// test confirms that changes in service tag data do not trigger the task.
func TestCondition_CatalogServices_NoTagsTrigger(t *testing.T) {
	t.Parallel()

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	service := testutil.TestService{ID: "api-1", Name: "api", Tags: []string{"tag_a"}}
	testutils.RegisterConsulService(t, srv, service, testutil.HealthPassing)
	time.Sleep(7 * time.Second)

	taskName := "catalog_task"
	conditionTask := fmt.Sprintf(`task {
	name = "%s"
	services = ["api", "db"]
	source = "./test_modules/local_tags_file"
	condition "catalog-services" {
		source_includes_var = true
	}
}
`, taskName)

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "cs_condition_no_tags_trigger")
	cleanup := testutils.MakeTempDir(t, tempDir)

	config := baseConfig().appendConsulBlock(srv).appendTerraformBlock(tempDir).
		appendString(conditionTask)
	configPath := filepath.Join(tempDir, configFile)
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)

	err := cts.WaitForAPI(15 * time.Second)
	require.NoError(t, err)

	// Test that task is not triggered by service tag changes and only by
	// service registration changes.
	// 0. Confirm baseline: check current number of events for initially
	//    registered api service. check for tag data in resource
	// 1. Register api-2 service instance with different tags. Confirm task was
	//    not triggered (no new event) and therefore api-2 data not in tfvars
	// 2. Register db service. Confirm task was triggered (new event) and db
	//    and api-2 data is in tfvars

	// 0. Confirm one event. Confirm tag data in resource
	evenCountBase := eventCount(t, taskName, cts.Port())
	require.Equal(t, 1, evenCountBase)

	workingDir := fmt.Sprintf("%s/%s", tempDir, taskName)
	content := testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "tag_a")

	// 1. Register another api service instance with new tags (no trigger)
	service = testutil.TestService{ID: "api-2", Name: "api", Tags: []string{"tag_b"}}
	testutils.RegisterConsulService(t, srv, service, testutil.HealthPassing)
	time.Sleep(7 * time.Second)

	eventCountNow := eventCount(t, taskName, cts.Port())
	require.Equal(t, evenCountBase, eventCountNow,
		"change in event count. task was unexpectedly triggered")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.NotContains(t, content, "tag_b")

	// 2. Register new db service (trigger + render template)
	service = testutil.TestService{ID: "db-1", Name: "db", Tags: []string{"tag_c"}}
	testutils.RegisterConsulService(t, srv, service, testutil.HealthPassing)
	time.Sleep(7 * time.Second)

	eventCountNow = eventCount(t, taskName, cts.Port())
	require.Equal(t, evenCountBase+1, eventCountNow,
		"event count did not increment once. task was not triggered as expected")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "tag_b")
	assert.Contains(t, content, "tag_c")

	cleanup()
}

// TestCondition_CatalogServices_Include runs the CTS binary. It specifically
// tests a task configured with a catalog service condition with the
// source_includes_var = true. This test confirms that the catalog_service
// definition can be consumed by a module as expected.
func TestCondition_CatalogServices_Include(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "cs_condition_include")
	delete := testutils.MakeTempDir(t, tempDir)

	conditionTask := `task {
	name = "catalog_task"
	services = ["api"]
	source = "./test_modules/local_tags_file"
	condition "catalog-services" {
		regexp = "db|web"
		source_includes_var = true
	}
}
`
	config := baseConfig().appendConsulBlock(srv).appendTerraformBlock(tempDir).
		appendString(conditionTask)
	configPath := filepath.Join(tempDir, configFile)
	config.write(t, configPath)

	api.StartCTS(t, configPath, api.CTSOnceModeFlag)

	// confirm that only two files were generated, one for db and one for web
	resourcesPath := fmt.Sprintf("%s/%s", tempDir, resourcesDir)
	files := testutils.CheckDir(t, true, resourcesPath)
	require.Equal(t, 2, len(files))

	contents := testutils.CheckFile(t, true, resourcesPath, "db_tags.txt")
	require.Equal(t, "tag3,tag4", string(contents))

	contents = testutils.CheckFile(t, true, resourcesPath, "web_tags.txt")
	require.Equal(t, "tag2", string(contents))

	delete()
}

// TestCondition_CatalogServices_Regexp runs the CTS binary. It specifically
// tests a task configured with a catalog service condition with the regexp
// configuration set. This test confirms that when a service is registered that
// doesn't match the task condition's regexp config:
//  1) the task is not triggered (determined by confirming from the Task Status
//     API that an event had not been added)
//  2) the service information does not exist in the tfvars file
func TestCondition_CatalogServices_Regexp(t *testing.T) {
	t.Parallel()

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "cs_condition_regexp")
	cleanup := testutils.MakeTempDir(t, tempDir)

	taskName := "catalog_task"
	conditionTask := fmt.Sprintf(`task {
	name = "%s"
	source = "./test_modules/local_tags_file"
	condition "catalog-services" {
		regexp = "api-"
		source_includes_var = true
	}
}
`, taskName)

	config := baseConfig().appendConsulBlock(srv).appendTerraformBlock(tempDir).
		appendString(conditionTask)
	configPath := filepath.Join(tempDir, configFile)
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)

	err := cts.WaitForAPI(15 * time.Second)
	require.NoError(t, err)

	// Test that regex filter is filtering service registration information and
	// task triggers
	// 0. Confirm baseline: check current number of events and that that no
	//    catalog_service variable contains no service information
	// 1. Register db service instance. Confirm that the task was not triggered
	//    (no new event) and its data is filtered out of catalog_service.
	// 2. Register api-web service instance. Confirm that task was triggered
	//    (one new event) and its data exists in catalog_service.

	// 0. Confirm only one event. Confirm empty var catalog_services
	evenCountBase := eventCount(t, taskName, cts.Port())
	require.Equal(t, 1, evenCountBase)

	workingDir := fmt.Sprintf("%s/%s", tempDir, taskName)
	content := testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "catalog_services = {\n}")

	// 1. Register a filtered out service "db"
	service := testutil.TestService{ID: "db-1", Name: "db"}
	testutils.RegisterConsulService(t, srv, service, testutil.HealthPassing)
	time.Sleep(7 * time.Second)

	eventCountNow := eventCount(t, taskName, cts.Port())
	require.Equal(t, evenCountBase, eventCountNow,
		"change in event count. task was unexpectedly triggered")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "catalog_services = {\n}")

	// 2. Register a matched service "api-web"
	service = testutil.TestService{ID: "api-web-1", Name: "api-web"}
	testutils.RegisterConsulService(t, srv, service, testutil.HealthPassing)
	time.Sleep(7 * time.Second)

	eventCountNow = eventCount(t, taskName, cts.Port())
	require.Equal(t, evenCountBase+1, eventCountNow,
		"event count did not increment once. task was not triggered as expected")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, `"api-web" = []`)

	cleanup()
}

// TestCondition_CatalogServices_NodeMeta runs the CTS binary. It specifically
// tests a task configured with a catalog service condition with the node-meta
// configuration set. This test confirms that when a service is registered with
// a Consul server that does not have the configured node-meta:
//  1) the task is not triggered (determined by confirming from the Task Status
//     API that an event had not been added)
//  2) the service information does not exist in the tfvars file
func TestCondition_CatalogServices_NodeMeta(t *testing.T) {
	t.Parallel()

	// start a regular server
	srv1 := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv1.Stop()

	// start another server with custom node-meta
	srv2, err := testutil.NewTestServerConfigT(t,
		func(c *testutil.TestServerConfig) {
			c.Bootstrap = false
			c.LogLevel = "warn"
			c.Stdout = ioutil.Discard
			c.Stderr = ioutil.Discard
			c.NodeMeta = map[string]string{"k": "v"}
		})
	require.NoError(t, err, "failed to start consul server 2")
	defer srv2.Stop()

	// join the two servers
	srv1.JoinLAN(t, srv2.LANAddr)

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "cs_condition_node_meta")
	cleanup := testutils.MakeTempDir(t, tempDir)

	taskName := "catalog_task"
	conditionTask := fmt.Sprintf(`task {
	name = "%s"
	source = "./test_modules/local_tags_file"
	condition "catalog-services" {
		regexp = "api"
		node_meta {
			k = "v"
		}
		source_includes_var = true
	}
}
`, taskName)

	config := baseConfig().appendConsulBlock(srv1).appendTerraformBlock(tempDir).
		appendString(conditionTask)
	configPath := filepath.Join(tempDir, configFile)
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)

	err = cts.WaitForAPI(15 * time.Second)
	require.NoError(t, err)

	// Test that node-meta filter is filtering service registration information
	// and task triggers
	// 0. Confirm baseline: check current number of events and that that no
	//    catalog_service variable contains no service information
	// 1. Register service with server with no node-meta. Confirm that the task
	//    was not triggered (no new event) and data is filtered out of
	//    catalog_service.
	// 2. Register service with server with configured node-meta. Confirm that
	//    task was triggered (one new event) and its data exists in catalog_service.

	// 0. Confirm only one event. Confirm empty var catalog_services
	evenCountBase := eventCount(t, taskName, cts.Port())
	require.Equal(t, 1, evenCountBase)

	workingDir := fmt.Sprintf("%s/%s", tempDir, taskName)
	content := testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "catalog_services = {\n}")

	// 1. Register a filtered out service with server 1 (no node-meta)
	srv1.AddAddressableService(t, "api", testutil.HealthPassing, "1.2.3.4", 8080,
		[]string{"tag_a"})
	time.Sleep(7 * time.Second)

	eventCountNow := eventCount(t, taskName, cts.Port())
	require.Equal(t, evenCountBase, eventCountNow,
		"change in event count. task was unexpectedly triggered")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "catalog_services = {\n}")

	// 2. Register a matched service with server 2 (with configured node-meta)
	srv2.AddAddressableService(t, "api", testutil.HealthPassing, "1.2.3.4", 8080,
		[]string{"tag_b"})
	time.Sleep(7 * time.Second)

	eventCountNow = eventCount(t, taskName, cts.Port())
	require.Equal(t, evenCountBase+1, eventCountNow,
		"event count did not increment once. task was not triggered as expected")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, `"api" = ["tag_b"]`)

	cleanup()
}

// eventCount returns number of events that are stored for a given task by
// querying the Task Status API. Note: events have a storage limit (currently 5)
func eventCount(t *testing.T, taskName string, port int) int {
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
	require.True(t, ok)
	return len(taskStatus.Events)
}
