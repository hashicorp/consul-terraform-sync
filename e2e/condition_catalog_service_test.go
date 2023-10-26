// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"path/filepath"
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

// TestCondition_CatalogServices_Registration runs the CTS binary. It is a basic
// test for a task configured with a catalog service condition. This test
// confirms that the first instance of a service registering and the last
// instance of a service deregistering triggers this task. Note, this test also
// also covers ensuring that daemon-mode can pass through once-mode with no
// initial service registrations
func TestCondition_CatalogServices_Registration(t *testing.T) {
	setParallelism(t)

	cases := []struct {
		name             string
		resource         string
		taskConf         string
		useAsModuleInput bool
	}{
		{
			"use_as_module_input_true",
			"api_tags.txt",
			`task {
	name = "catalog_task"
	module = "./test_modules/local_tags_file"
	condition "catalog-services" {
		regexp = "^api$"
		use_as_module_input = true
	}
}`,
			true,
		},
		{
			"use_as_module_input_false",
			"api-1.txt",
			`task {
	name = "catalog_task"
	module = "./test_modules/local_instances_file"
	condition "catalog-services" {
		regexp = "^api$"
		use_as_module_input = false
	}
}`,
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testCatalogServicesRegistration(t, tc.taskConf, "catalog_task",
				"cs_condition_registration_use_", tc.resource, tc.useAsModuleInput)
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
	setParallelism(t)

	cases := []struct {
		name             string
		useAsModuleInput bool
		taskConf         string
	}{
		{
			"use_as_module_input_true",
			true,
			`task {
	name = "catalog_task"
	module = "./test_modules/local_tags_file"
	condition "catalog-services" {
		regexp = "^api$|^db$"
		use_as_module_input = true
	}
	module_input "services" {
		names = ["api", "db"]
	}
}`,
		},
		{
			"use_as_module_input_false",
			false,
			`task {
	name = "catalog_task"
	module = "./test_modules/local_instances_file"
	condition "catalog-services" {
		regexp = "^api$|^db$"
		use_as_module_input = false
	}
	module_input "services" {
		names = ["api", "db"]
	}
}`,
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s/ServicesTrigger", tc.name), func(t *testing.T) {
			testCatalogServicesNoServicesTrigger(t, tc.taskConf, "catalog_task",
				"cs_condition_no_services_trigger_use_", tc.useAsModuleInput)
		})

		t.Run(fmt.Sprintf("%s/TagsTrigger", tc.name), func(t *testing.T) {
			testCatalogServicesNoTagsTrigger(t, tc.taskConf, "catalog_task",
				"cs_condition_no_tags_trigger_use_", tc.useAsModuleInput)
		})
	}
}

// TestCondition_CatalogServices_UseAsModuleInput runs the CTS binary. It
// specifically tests a task configured with a catalog service condition with the
// use_as_module_input = true. This test confirms that the catalog_service
// definition is consumed by a module as expected.
func TestCondition_CatalogServices_UseAsModuleInput(t *testing.T) {
	setParallelism(t)

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "cs_condition_use")
	conditionTask := `task {
	name = "catalog_task"
	module = "./test_modules/local_tags_file"
	condition "catalog-services" {
		regexp = "db|web"
		use_as_module_input = true
	}
}
`
	ctsSetup(t, srv, tempDir, conditionTask)

	// confirm that only two files were generated, one for db and one for web
	resourcesPath := filepath.Join(tempDir, "catalog_task", resourcesDir)
	files := testutils.CheckDir(t, true, resourcesPath)
	require.Equal(t, 2, len(files))

	validateModuleFile(t, true, true, resourcesPath, "db_tags", "tag3,tag4")
	validateModuleFile(t, true, true, resourcesPath, "web_tags", "tag2")
}

// TestCondition_CatalogServices_Regexp runs the CTS binary. It specifically
// tests a task configured with a catalog service condition with the regexp
// configuration set. This test confirms that when a service is registered that
// doesn't match the task condition's regexp config:
//  1) the task is not triggered (determined by confirming from the Task Status
//     API that an event had not been added)
//  2) the service information does not exist in the tfvars file
func TestCondition_CatalogServices_Regexp(t *testing.T) {
	setParallelism(t)

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "cs_condition_regexp")
	taskName := "catalog_task"
	conditionTask := fmt.Sprintf(`task {
	name = "%s"
	module = "./test_modules/local_tags_file"
	condition "catalog-services" {
		regexp = "api-"
		use_as_module_input = true
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
	validateVariable(t, true, workingDir, "catalog_services", "{\n}")

	// 1. Register a filtered out service "db"
	service := testutil.TestService{ID: "db-1", Name: "db"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	time.Sleep(defaultWaitForNoEvent)

	eventCountNow := eventCount(t, taskName, cts.Port())
	require.Equal(t, eventCountBase, eventCountNow,
		"change in event count. task was unexpectedly triggered")

	validateVariable(t, true, workingDir, "catalog_services", "{\n}")
	resourcesPath := filepath.Join(workingDir, resourcesDir)
	validateModuleFile(t, true, false, resourcesPath, "db_tags", "")

	// 2. Register a matched service "api-web"
	now := time.Now()
	service = testutil.TestService{ID: "api-web-1", Name: "api-web"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)

	eventCountNow = eventCount(t, taskName, cts.Port())
	require.Equal(t, eventCountBase+1, eventCountNow,
		"event count did not increment once. task was not triggered as expected")

	validateVariable(t, true, workingDir, "catalog_services", `"api-web" = []`)
	validateModuleFile(t, true, true, resourcesPath, "api-web_tags", "")
	validateModuleFile(t, true, false, resourcesPath, "db_tags", "")
}

func TestCondition_CatalogServices_MultipleTasks(t *testing.T) {
	setParallelism(t)

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
	module = "./test_modules/local_tags_file"
	condition "catalog-services" {
		regexp = "api"
		use_as_module_input = true
	}
}
task {
	name = "%s"
	module = "./test_modules/local_tags_file"
	condition "catalog-services" {
		regexp = "^api$|^web$"
		use_as_module_input = true
	}
}
task {
	name = "%s"
	module = "./test_modules/local_tags_file"
	condition "catalog-services" {
		regexp = ".*"
		use_as_module_input = true
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

	validateModuleFile(t, true, true, allResourcesPath, "api_tags", "")
	validateModuleFile(t, true, true, apiWebResourcesPath, "api_tags", "")
	validateModuleFile(t, true, true, apiResourcesPath, "api_tags", "")

	// 2. Register web, only all_task and api_web_task create resource
	now = time.Now()
	service = testutil.TestService{ID: "web-1", Name: "web"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, allTaskName, now, defaultWaitForEvent*2)
	api.WaitForEvent(t, cts, apiWebTaskName, now, defaultWaitForEvent*2)

	validateModuleFile(t, true, true, allResourcesPath, "web_tags", "")
	validateModuleFile(t, true, true, apiWebResourcesPath, "web_tags", "")
	validateModuleFile(t, true, false, apiResourcesPath, "web_tags", "")

	// 3. Register db, only all_task create resource
	now = time.Now()
	service = testutil.TestService{ID: "db-1", Name: "db"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, allTaskName, now, defaultWaitForEvent)
	time.Sleep(defaultWaitForNoEvent) // ensure api_web_task & api_task don't trigger

	validateModuleFile(t, true, true, allResourcesPath, "db_tags", "")
	validateModuleFile(t, true, false, apiWebResourcesPath, "db_tags", "")
	validateModuleFile(t, true, false, apiResourcesPath, "db_tags", "")
}

func TestCondition_CatalogServices_InvalidQueries(t *testing.T) {
	setParallelism(t)
	config := `task {
		name = "%s"
		module = "./test_modules/null_resource"
		condition "catalog-services" {
			regexp = ".*"
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
			"namespace_with_oss_consul",
			`namespace = "foo"`,
			`Invalid query parameter: "ns"`,
		},
	}

	for _, tc := range cases {
		tc := tc // rebind tc into this lexical scope for parallel use
		taskName := "condition_catalog_services_invalid_" + tc.name
		taskConfig := fmt.Sprintf(config, taskName, tc.queryConfig)
		testInvalidTaskConfig(t, tc.name, taskName, taskConfig, tc.errMsg)
	}
}

// TestCondition_CatalogServices_SharedDependency_NoServices tests that a
// task with a catalog-service condition and a dependency that's the same
// as an existing task is created successfully via the API and the request
// does not hang. Tests the scenario where there are no services registered
// that satisfy the catalog-service regex.
//
// https://github.com/hashicorp/consul-terraform-sync/issues/704
func TestCondition_CatalogServices_SharedDependency_NoServices(t *testing.T) {
	setParallelism(t)

	inspectRunMode := "inspect"
	nowRunMode := "now"

	cases := []struct {
		name    string
		runMode *string // using string pointer type to test the case of no run mode being set
	}{
		{
			"create",
			nil,
		}, {
			"inspect",
			&inspectRunMode,
		},
		{
			"create_and_run",
			&nowRunMode,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			setParallelism(t)
			// Start Consul without any registered services
			srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
				HTTPSRelPath: "../testutils",
			})
			defer srv.Stop()

			// Initialize CTS with a task that has API service as a module_input
			initTask := "init_task"
			sharedService := "api"
			initConfig := fmt.Sprintf(`
task {
	name = "%s"
	module = "./test_modules/local_instances_file"
	condition "services" {
		names = ["%s"]
		use_as_module_input = true
	}
}
`, initTask, sharedService)
			tempDir := fmt.Sprintf("%scs_shared_dep_%s", tempDirPrefix, tc.name)
			cts := ctsSetup(t, srv, tempDir, initConfig)

			// Create a task that has a catalog-services condition and also has
			// the API service as a module_input
			client, err := api.NewTaskLifecycleClient(&api.ClientConfig{
				URL:       cts.FullAddress(),
				TLSConfig: api.TLSConfig{SSLVerify: false}}, nil)
			require.NoError(t, err)

			taskName := "created_task"
			createReq := api.TaskRequest{
				Task: oapigen.Task{
					Name:   taskName,
					Module: "./test_modules/local_tags_file",
					Condition: oapigen.Condition{
						CatalogServices: &oapigen.CatalogServicesCondition{
							Regexp: "web",
						},
					},
					ModuleInput: &oapigen.ModuleInput{
						Services: &oapigen.ServicesModuleInput{
							Names: &[]string{sharedService}, // shared dependency
						},
					},
				},
			}

			// Create task, check that API request is successful and does not hang
			params := oapigen.CreateTaskParams{
				Run: (*oapigen.CreateTaskParamsRun)(tc.runMode),
			}

			_, err = client.CreateTaskWithResponse(context.Background(), &params, oapigen.CreateTaskJSONRequestBody(createReq))

			require.NoError(t, err)
		})
	}
}

// TestCondition_CatalogServices_SuppressTriggers_SharedDependencies tests that a
// task created with a catalog-services condition that shares a dependencies with
// an existing task only triggers on an expected services change.
//
// https://github.com/hashicorp/consul-terraform-sync/issues/704
func TestCondition_CatalogServices_SuppressTriggers_SharedDependencies(t *testing.T) {
	setParallelism(t)

	// Set up Consul server with a service that the catalog-service condition monitors
	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()
	service := testutil.TestService{ID: "web-1", Name: "web-test"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

	// Configure and start CTS with an existing task
	taskName := "cs_cond_cli_w_shared_dep"
	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, taskName)
	initTask := "init_task"
	sharedService := "api"
	initConfig := fmt.Sprintf(`
task {
	name = "%s"
	module = "./test_modules/local_instances_file"
	condition "services" {
		names = ["%s"]
		use_as_module_input = true
	}
}
`, initTask, sharedService)
	cts := ctsSetup(t, srv, tempDir, initConfig)

	// Create a task via the CLI that has a services condition and
	// shares a dependency with the existing task (consul-kv pair)
	config := fmt.Sprintf(`task {
	name = "%s"
	module = "lornasong/cts_cs_file/local"
	condition "catalog-services" {
		regexp = "web"
	}
	module_input  "services" {
		names = ["%s"]
	}
}`, taskName, sharedService)

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
	require.Contains(t, out, fmt.Sprintf("Task '%s' created", taskName))

	// Confirm one event at creation
	count := eventCount(t, taskName, cts.Port())
	expectedCount := 1
	require.Equal(t, expectedCount, count, "unexpected number of events for created task")

	// Register an instance to the existing service with a tag change
	service = testutil.TestService{ID: "web-2", Name: "web-test", Tags: []string{"tag_a"}}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

	// Check that created task does not trigger
	time.Sleep(defaultWaitForNoEvent)
	count = eventCount(t, taskName, cts.Port())
	assert.Equal(t, expectedCount, count, "unexpected number of events for created task")

	// Register service that only the initial task will trigger on
	registrationTime := time.Now()
	service = testutil.TestService{ID: sharedService, Name: sharedService}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

	// Check that created event is not triggered
	time.Sleep(defaultWaitForNoEvent)
	count = eventCount(t, taskName, cts.Port())
	assert.Equal(t, 1, count, "unexpected number of events for created task")

	// Check that initial task does trigger
	api.WaitForEvent(t, cts, initTask, registrationTime, defaultWaitForEvent)
	initCount := eventCount(t, initTask, cts.Port())
	assert.Equal(t, 2, initCount)
	initResources := filepath.Join(tempDir, initTask, resourcesDir)
	validateServices(t, true, []string{sharedService}, initResources)

	// Register a new service that will trigger the created task
	service = testutil.TestService{ID: "web-test-new", Name: "web-test-new", Tags: []string{"tag_b"}}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

	// Check that created task does trigger
	api.WaitForEvent(t, cts, taskName, registrationTime, defaultWaitForEvent)
	count = eventCount(t, taskName, cts.Port())
	expectedCount++
	assert.Equal(t, expectedCount, count)
	resourcesPath := filepath.Join(tempDir, taskName, resourcesDir)
	validateServices(t, true, []string{"services_" + sharedService}, // module prepends 'services_' to filenames
		resourcesPath)
	validateModuleFile(t, true, true, resourcesPath, "tags_web-test-new", "tag_b")
	validateModuleFile(t, true, true, resourcesPath, "tags_web-test", "tag_a")
}

func testCatalogServicesRegistration(t *testing.T, taskConf, taskName,
	tempDirName, resource string, useAsModuleInput bool) {

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s%t", tempDirPrefix, tempDirName, useAsModuleInput)
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
	validateModuleFile(t, useAsModuleInput, true, resourcesPath, "api_tags", "")

	// 2. Deregister api, resource destroyed
	now = time.Now()
	testutils.DeregisterConsulService(t, srv, "api-1")
	api.WaitForEvent(t, cts, taskName, now,
		defaultWaitForRegistration+defaultWaitForEvent)
	validateModuleFile(t, useAsModuleInput, false, resourcesPath, "api_tags", "")
}

func testCatalogServicesNoServicesTrigger(t *testing.T, taskConf, taskName,
	tempDirName string, useAsModuleInput bool) {

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	service := testutil.TestService{ID: "api-1", Name: "api"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

	tempDir := fmt.Sprintf("%s%s%t", tempDirPrefix, tempDirName, useAsModuleInput)
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
	validateVariable(t, true, workingDir, "services", "api-1")
	resourcesPath := filepath.Join(workingDir, resourcesDir)
	validateModuleFile(t, useAsModuleInput, true, resourcesPath, "api_tags", "")

	// 1. Register second api service instance "api-2" (no trigger)
	service = testutil.TestService{ID: "api-2", Name: "api"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	time.Sleep(defaultWaitForNoEvent)

	eventCountNow := eventCount(t, taskName, cts.Port())
	require.Equal(t, eventCountBase, eventCountNow,
		"change in event count. task was unexpectedly triggered")

	validateVariable(t, false, workingDir, "services", "api-2")
	validateModuleFile(t, useAsModuleInput, true, resourcesPath, "api_tags", "")

	// 2. Register db service (trigger + render template)
	now := time.Now()
	service = testutil.TestService{ID: "db-1", Name: "db"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)

	eventCountNow = eventCount(t, taskName, cts.Port())
	require.Equal(t, eventCountBase+1, eventCountNow,
		"event count did not increment once. task was not triggered as expected")

	validateVariable(t, true, workingDir, "services", "api-2")
	validateVariable(t, true, workingDir, "services", "db-1")
	validateModuleFile(t, useAsModuleInput, true, resourcesPath, "api_tags", "")
	validateModuleFile(t, useAsModuleInput, true, resourcesPath, "db_tags", "")
}

func testCatalogServicesNoTagsTrigger(t *testing.T, taskConf, taskName,
	tempDirName string, useAsModuleInput bool) {

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	service := testutil.TestService{ID: "api-1", Name: "api", Tags: []string{"tag_a"}}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

	tempDir := fmt.Sprintf("%s%s%t", tempDirPrefix, tempDirName, useAsModuleInput)
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
	validateVariable(t, true, workingDir, "services", "tag_a")

	resourcesPath := filepath.Join(tempDir, "catalog_task", resourcesDir)
	validateModuleFile(t, useAsModuleInput, true, resourcesPath, "api_tags", "tag_a")

	// 1. Register another api service instance with new tags (no trigger)
	service = testutil.TestService{ID: "api-2", Name: "api", Tags: []string{"tag_b"}}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	time.Sleep(defaultWaitForNoEvent)

	eventCountNow := eventCount(t, taskName, cts.Port())
	require.Equal(t, eventCountBase, eventCountNow,
		"change in event count. task was unexpectedly triggered")

	validateVariable(t, false, workingDir, "services", "tag_b")
	validateModuleFile(t, useAsModuleInput, true, resourcesPath, "api_tags", "tag_a")

	// 2. Register new db service (trigger + render template)
	now := time.Now()
	service = testutil.TestService{ID: "db-1", Name: "db", Tags: []string{"tag_c"}}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)

	eventCountNow = eventCount(t, taskName, cts.Port())
	require.Equal(t, eventCountBase+1, eventCountNow,
		"event count did not increment once. task was not triggered as expected")

	validateVariable(t, true, workingDir, "services", "tag_b")
	validateVariable(t, true, workingDir, "services", "tag_c")
	validateModuleFile(t, useAsModuleInput, true, resourcesPath, "api_tags", "tag_a,tag_b")
	validateModuleFile(t, useAsModuleInput, true, resourcesPath, "db_tags", "tag_c")
}
