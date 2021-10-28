//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
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

	cases := []struct {
		name        string
		tempDirName string
		resource    string
		taskConf    string
	}{
		{
			"source_includes_var=true",
			"cs_condition_registration_include",
			"api_tags.txt",
			`task {
	name = "catalog_task"
	services = ["api"]
	source = "./test_modules/local_tags_file"
	condition "catalog-services" {
		source_includes_var = true
	}
}`,
		},
		{
			"source_includes_var=false",
			"cs_condition_registration",
			"api-1.txt",
			`task {
	name = "catalog_task"
	services = ["api"]
	source = "./test_modules/local_instances_file"
	condition "catalog-services" {}
}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testCatalogServicesRegistration(t, tc.taskConf, "catalog_task",
				tc.tempDirName, tc.resource)
		})
	}
}

// TestCondition_CatalogServices_SuppressTriggers runs the CTS binary. It
// specifically tests a task configured with a catalog service condition. This
// test confirms that the types of changes below do not trigger the task:
//  - changes in service instances that do not affect overall service
//    [de]registration
//  - changes in service tag data
func TestCondition_CatalogServices_SuppressTriggers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		include  bool
		taskConf string
	}{
		{
			"source_includes_var=true",
			true,
			`task {
	name = "catalog_task"
	services = ["api", "db"]
	source = "./test_modules/local_tags_file"
	condition "catalog-services" {
		source_includes_var = true
	}
}`,
		},
		{
			"source_includes_var=false",
			false,
			`task {
	name = "catalog_task"
	services = ["api", "db"]
	source = "./test_modules/local_instances_file"
	condition "catalog-services" {}
}`,
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s/ServicesTrigger", tc.name), func(t *testing.T) {
			testCatalogServicesNoServicesTrigger(t, tc.taskConf, "catalog_task",
				fmt.Sprintf("cs_condition_no_services_trigger_include_%t", tc.include))
		})

		t.Run(fmt.Sprintf("%s/TagsTrigger", tc.name), func(t *testing.T) {
			testCatalogServicesNoTagsTrigger(t, tc.taskConf, "catalog_task",
				fmt.Sprintf("cs_condition_no_tags_trigger_include_%t", tc.include))
		})
	}
}

// TestCondition_CatalogServices_Include runs the CTS binary. It specifically
// tests a task configured with a catalog service condition with the
// source_includes_var = true. This test confirms that the catalog_service
// definition is consumed by a module as expected.
func TestCondition_CatalogServices_Include(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "cs_condition_include")
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
	ctsSetup(t, srv, tempDir, conditionTask)

	// confirm that only two files were generated, one for db and one for web
	resourcesPath := filepath.Join(tempDir, "catalog_task", resourcesDir)
	files := testutils.CheckDir(t, true, resourcesPath)
	require.Equal(t, 2, len(files))

	contents := testutils.CheckFile(t, true, resourcesPath, "db_tags.txt")
	require.Equal(t, "tag3,tag4", string(contents))

	contents = testutils.CheckFile(t, true, resourcesPath, "web_tags.txt")
	require.Equal(t, "tag2", string(contents))
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

	cts := ctsSetup(t, srv, tempDir, conditionTask)

	// Test that regex filter is filtering service registration information and
	// task triggers
	// 0. Confirm baseline: check current number of events and that that no
	//    catalog_service variable contains no service information
	// 1. Register db service instance. Confirm that the task was not triggered
	//    (no new event) and its data is filtered out of catalog_service.
	// 2. Register api-web service instance. Confirm that task was triggered
	//    (one new event) and its data exists in catalog_service.

	// 0. Confirm only one event. Confirm empty var catalog_services
	eventCountBase := eventCount(t, taskName, cts.Port())
	require.Equal(t, 1, eventCountBase)

	workingDir := fmt.Sprintf("%s/%s", tempDir, taskName)
	content := testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "catalog_services = {\n}")

	// 1. Register a filtered out service "db"
	service := testutil.TestService{ID: "db-1", Name: "db"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	time.Sleep(defaultWaitForNoEvent)

	eventCountNow := eventCount(t, taskName, cts.Port())
	require.Equal(t, eventCountBase, eventCountNow,
		"change in event count. task was unexpectedly triggered")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "catalog_services = {\n}")

	// 2. Register a matched service "api-web"
	now := time.Now()
	service = testutil.TestService{ID: "api-web-1", Name: "api-web"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)

	eventCountNow = eventCount(t, taskName, cts.Port())
	require.Equal(t, eventCountBase+1, eventCountNow,
		"event count did not increment once. task was not triggered as expected")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, `"api-web" = []`)
}

func TestCondition_CatalogServices_MultipleTasks(t *testing.T) {
	t.Parallel()

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	apiTaskName := "api_task"
	apiWebTaskName := "api_web_task"
	allTaskName := "all_task"
	tasks := fmt.Sprintf(`
task {
	name = "%s"
	source = "./test_modules/local_tags_file"
	condition "catalog-services" {
		regexp = "api"
		source_includes_var = true
	}
}
task {
	name = "%s"
	source = "./test_modules/local_tags_file"
	condition "catalog-services" {
		regexp = "^api$|^web$"
		source_includes_var = true
	}
}
task {
	name = "%s"
	source = "./test_modules/local_tags_file"
	condition "catalog-services" {
		regexp = ".*"
		source_includes_var = true
	}
}
`, apiTaskName, apiWebTaskName, allTaskName)

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "cs_condition_multi")
	cts := ctsSetup(t, srv, tempDir, tasks)

	// Test that the appropriate task is triggered given a particular service
	// registration
	// 1. Register 'api' service. Confirm all tasks are triggered
	// 2. Register 'web' service. Confirm that only api_web_task and all_task
	// 3. Register 'db' service. Confirm only all_task is registered

	apiResourcesPath := filepath.Join(tempDir, apiTaskName, resourcesDir)
	apiWebResourcesPath := filepath.Join(tempDir, apiWebTaskName, resourcesDir)
	allResourcesPath := filepath.Join(tempDir, allTaskName, resourcesDir)

	// 1. Register api, all tasks create resource
	now := time.Now()
	service := testutil.TestService{ID: "api-1", Name: "api"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	// wait longer than default since more tasks are being executed
	api.WaitForEvent(t, cts, allTaskName, now, defaultWaitForEvent*2)
	api.WaitForEvent(t, cts, apiWebTaskName, now, defaultWaitForEvent*2)
	api.WaitForEvent(t, cts, apiTaskName, now, defaultWaitForEvent*2)

	testutils.CheckFile(t, true, allResourcesPath, "api_tags.txt")
	testutils.CheckFile(t, true, apiWebResourcesPath, "api_tags.txt")
	testutils.CheckFile(t, true, apiResourcesPath, "api_tags.txt")

	// 2. Register web, only all_task and api_web_task create resource
	now = time.Now()
	service = testutil.TestService{ID: "web-1", Name: "web"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, allTaskName, now, defaultWaitForEvent*2)
	api.WaitForEvent(t, cts, apiWebTaskName, now, defaultWaitForEvent*2)

	testutils.CheckFile(t, true, allResourcesPath, "web_tags.txt")
	testutils.CheckFile(t, true, apiWebResourcesPath, "web_tags.txt")
	testutils.CheckFile(t, false, apiResourcesPath, "web_tags.txt")

	// 3. Register db, only all_task create resource
	now = time.Now()
	service = testutil.TestService{ID: "db-1", Name: "db"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, allTaskName, now, defaultWaitForEvent)
	time.Sleep(defaultWaitForNoEvent) // ensure api_web_task & api_task don't trigger

	testutils.CheckFile(t, true, allResourcesPath, "db_tags.txt")
	testutils.CheckFile(t, false, apiWebResourcesPath, "db_tags.txt")
	testutils.CheckFile(t, false, apiResourcesPath, "db_tags.txt")
}

func testCatalogServicesRegistration(t *testing.T, taskConf, taskName, tempDirName, resource string) {
	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, tempDirName)
	cts := ctsSetup(t, srv, tempDir, taskConf)

	// Test that task is triggered on service registration and deregistration
	// 0. Confirm baseline: nothing is registered so no resource created yet
	// 1. Register 'api' service. Confirm resource created
	// 2. Deregister 'api' service. Confirm resource destroyed

	// 0. Confirm resource not created
	resourcesPath := filepath.Join(tempDir, taskName, resourcesDir)
	testutils.CheckFile(t, false, resourcesPath, resource)

	// 1. Register api, resource created
	now := time.Now()
	service := testutil.TestService{ID: "api-1", Name: "api"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)

	testutils.CheckFile(t, true, resourcesPath, resource)

	// 2. Deregister api, resource destroyed
	now = time.Now()
	testutils.DeregisterConsulService(t, srv, "api-1")
	api.WaitForEvent(t, cts, taskName, now,
		defaultWaitForRegistration+defaultWaitForEvent)
	testutils.CheckFile(t, false, resourcesPath, resource)
}

func testCatalogServicesNoServicesTrigger(t *testing.T, taskConf, taskName, tempDirName string) {
	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	service := testutil.TestService{ID: "api-1", Name: "api"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, tempDirName)
	cts := ctsSetup(t, srv, tempDir, taskConf)

	// Test that task is not triggered by service-instance specific changes and
	// only by service registration changes.
	// 0. Confirm baseline: check current number of events for initially
	//    registered api service. check for instance data in tfvars
	// 1. Register api-2 instance for existing api service. Confirm task was not
	//    triggered (no new event) and therefore api-2 not in tfvars
	// 2. Register new db service. Confirm task was triggered (new event) and
	//    db and api-2 (now) rendered in tfvars

	// 0. Confirm one event. Confirm initial api service registration data
	eventCountBase := eventCount(t, taskName, cts.Port())
	require.Equal(t, 1, eventCountBase)

	workingDir := filepath.Join(tempDir, taskName)
	content := testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "api-1")

	// 1. Register second api service instance "api-2" (no trigger)
	service = testutil.TestService{ID: "api-2", Name: "api"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	time.Sleep(defaultWaitForNoEvent)

	eventCountNow := eventCount(t, taskName, cts.Port())
	require.Equal(t, eventCountBase, eventCountNow,
		"change in event count. task was unexpectedly triggered")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.NotContains(t, content, "api-2")

	// 2. Register db service (trigger + render template)
	now := time.Now()
	service = testutil.TestService{ID: "db-1", Name: "db"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)

	eventCountNow = eventCount(t, taskName, cts.Port())
	require.Equal(t, eventCountBase+1, eventCountNow,
		"event count did not increment once. task was not triggered as expected")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "api-2")
	assert.Contains(t, content, "db-1")
}

func testCatalogServicesNoTagsTrigger(t *testing.T, taskConf, taskName, tempDirName string) {
	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	service := testutil.TestService{ID: "api-1", Name: "api", Tags: []string{"tag_a"}}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, tempDirName)
	cts := ctsSetup(t, srv, tempDir, taskConf)

	// Test that task is not triggered by service tag changes and only by
	// service registration changes.
	// 0. Confirm baseline: check current number of events for initially
	//    registered api service. check for tag data in resource
	// 1. Register api-2 service instance with different tags. Confirm task was
	//    not triggered (no new event) and therefore api-2 data not in tfvars
	// 2. Register db service. Confirm task was triggered (new event) and db
	//    and api-2 data is in tfvars

	// 0. Confirm one event. Confirm tag data in resource
	eventCountBase := eventCount(t, taskName, cts.Port())
	require.Equal(t, 1, eventCountBase)

	workingDir := filepath.Join(tempDir, taskName)
	content := testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "tag_a")

	// 1. Register another api service instance with new tags (no trigger)
	service = testutil.TestService{ID: "api-2", Name: "api", Tags: []string{"tag_b"}}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	time.Sleep(defaultWaitForNoEvent)

	eventCountNow := eventCount(t, taskName, cts.Port())
	require.Equal(t, eventCountBase, eventCountNow,
		"change in event count. task was unexpectedly triggered")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.NotContains(t, content, "tag_b")

	// 2. Register new db service (trigger + render template)
	now := time.Now()
	service = testutil.TestService{ID: "db-1", Name: "db", Tags: []string{"tag_c"}}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)

	eventCountNow = eventCount(t, taskName, cts.Port())
	require.Equal(t, eventCountBase+1, eventCountNow,
		"event count did not increment once. task was not triggered as expected")

	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "tag_b")
	assert.Contains(t, content, "tag_c")
}
