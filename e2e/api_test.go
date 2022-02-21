//go:build e2e
// +build e2e

// Tests CTS API endpoints /v1/status and /v1/tasks
package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	createTestTaskTemplate = `{
	"task": {
		"description": "Writes the service name, id, and IP address to a file",
		"name": "%s",
		"providers": [
			"local"
		],
        "condition": {
            "services": {
                "names": [
                    "%s"
                ]
            }
        },
		"module": "mkam/instance-files/local"
	}
}`
)

// TestE2E_StatusEndpoints tests all CTS status endpoints and query
// parameters. This runs a Consul server and the CTS binary in daemon mode.
//	GET	/v1/status/tasks
// 	GET	/v1/status/tasks/:task_name
//	GET	/v1/status
func TestE2E_StatusEndpoints(t *testing.T) {
	setParallelism(t)

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "status_endpoints")
	cleanup := testutils.MakeTempDir(t, tempDir)
	// no defer to delete directory: only delete at end of test if no errors

	configPath := filepath.Join(tempDir, configFile)
	config := fakeHandlerConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock()
	config.write(t, configPath)

	cts, stopCTS := api.StartCTS(t, configPath, api.CTSDevModeFlag)

	// wait to run once before registering another instance to collect another event
	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)
	service := testutil.TestService{
		ID:      "api-2",
		Name:    "api",
		Address: "5.6.7.8",
		Port:    8080,
	}
	now := time.Now()
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, fakeFailureTaskName, now, defaultWaitForEvent)
	api.WaitForEvent(t, cts, fakeSuccessTaskName, now, defaultWaitForEvent)

	taskCases := []struct {
		name       string
		path       string
		statusCode int
		expected   map[string]api.TaskStatus
	}{
		{
			"all task statuses",
			"status/tasks",
			http.StatusOK,
			map[string]api.TaskStatus{
				fakeSuccessTaskName: {
					TaskName:  fakeSuccessTaskName,
					Status:    api.StatusSuccessful,
					Enabled:   true,
					Providers: []string{"fake-sync"},
					Services:  []string{},
					EventsURL: "/v1/status/tasks/fake_handler_success_task?include=events",
				},
				fakeFailureTaskName: {
					TaskName:  fakeFailureTaskName,
					Status:    api.StatusErrored,
					Enabled:   true,
					Providers: []string{"fake-sync"},
					Services:  []string{},
					EventsURL: "/v1/status/tasks/fake_handler_failure_task?include=events",
				},
				disabledTaskName: {
					TaskName:  disabledTaskName,
					Status:    api.StatusUnknown,
					Enabled:   false,
					Providers: []string{"fake-sync"},
					Services:  []string{},
					EventsURL: "",
				},
			},
		},
		{
			"existing single task status",
			"status/tasks/" + fakeSuccessTaskName,
			http.StatusOK,
			map[string]api.TaskStatus{
				fakeSuccessTaskName: {
					TaskName:  fakeSuccessTaskName,
					Status:    api.StatusSuccessful,
					Enabled:   true,
					Providers: []string{"fake-sync"},
					Services:  []string{},
					EventsURL: "/v1/status/tasks/fake_handler_success_task?include=events",
				},
			},
		},
		{
			"non-existing single task status",
			"status/tasks/" + "non-existing-task",
			http.StatusNotFound,
			map[string]api.TaskStatus{},
		},
	}

	for _, tc := range taskCases {
		t.Run(tc.name, func(t *testing.T) {
			u := fmt.Sprintf("http://localhost:%d/%s/%s", cts.Port(), "v1", tc.path)
			resp := testutils.RequestHTTP(t, http.MethodGet, u, "")
			defer resp.Body.Close()

			assert.Equal(t, tc.statusCode, resp.StatusCode)

			if tc.statusCode != http.StatusOK {
				return
			}

			var taskStatuses map[string]api.TaskStatus
			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&taskStatuses)
			require.NoError(t, err)

			// clear out some event data that we'll skip checking
			for _, stat := range taskStatuses {
				for ix, event := range stat.Events {
					event.ID = ""
					event.StartTime = time.Time{}
					event.EndTime = time.Time{}
					stat.Events[ix] = event
				}
			}

			assert.Equal(t, tc.expected, taskStatuses)
		})
	}

	eventCases := []struct {
		name               string
		path               string
		expectSuccessTask  bool
		expectFailureTask  bool
		expectDisabledTask bool
	}{
		{
			"events: all task statuses",
			"status/tasks?include=events",
			true,
			true,
			true,
		},
		{
			"events: all task statuses filtered by status",
			"status/tasks?status=errored&include=events",
			false,
			true,
			false,
		},
	}

	for _, tc := range eventCases {
		t.Run(tc.name, func(t *testing.T) {
			u := fmt.Sprintf("http://localhost:%d/%s/%s", cts.Port(), "v1", tc.path)
			resp := testutils.RequestHTTP(t, http.MethodGet, u, "")
			defer resp.Body.Close()

			require.Equal(t, resp.StatusCode, http.StatusOK)

			var taskStatuses map[string]api.TaskStatus
			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&taskStatuses)
			require.NoError(t, err)

			checkEvents(t, taskStatuses, fakeFailureTaskName, tc.expectFailureTask)
			checkEvents(t, taskStatuses, fakeSuccessTaskName, tc.expectSuccessTask)

			task, ok := taskStatuses[disabledTaskName]
			if tc.expectDisabledTask {
				assert.True(t, ok)
				assert.Nil(t, task.Events)
			} else {
				assert.False(t, ok)
			}
		})
	}

	overallCases := []struct {
		name string
		path string
	}{
		{
			"overall status",
			"status",
		},
	}

	for _, tc := range overallCases {
		t.Run(tc.name, func(t *testing.T) {
			u := fmt.Sprintf("http://localhost:%d/%s/%s", cts.Port(), "v1", tc.path)
			resp := testutils.RequestHTTP(t, http.MethodGet, u, "")
			defer resp.Body.Close()

			require.Equal(t, resp.StatusCode, http.StatusOK)

			var overallStatus api.OverallStatus
			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&overallStatus)
			require.NoError(t, err)

			// check status values
			assert.Equal(t, 1, overallStatus.TaskSummary.Status.Successful)
			assert.Equal(t, 1, overallStatus.TaskSummary.Status.Unknown)
			// failed task might be errored/critical by now depending on number of events
			assert.Equal(t, 1, overallStatus.TaskSummary.Status.Errored+
				overallStatus.TaskSummary.Status.Critical)

			// check enabled values
			assert.Equal(t, 2, overallStatus.TaskSummary.Enabled.True)
			assert.Equal(t, 1, overallStatus.TaskSummary.Enabled.False)
		})
	}

	stopCTS(t)
	_ = cleanup()
}

// TestE2E_TaskEndpoints_UpdateEnableDisable tests the tasks endpoints. This
// runs a Consul server and the CTS binary in daemon mode.
//	PATCH	/v1/tasks/:task_name
func TestE2E_TaskEndpoints_UpdateEnableDisable(t *testing.T) {
	setParallelism(t)
	// Test enabling and disabling a task
	// 1. Start with disabled task. Confirm task is initialized, but
	//    not run (resources not created)
	// 2. API to inspect enabling task. Confirm plan looks good, resources not
	//    created, and task not actually enabled.
	// 3. API to actually enable task. Confirm resources are created
	// 4. API to disable task. Delete resources. Register new service. Confirm
	//    new service registering does not trigger creating resources

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "disabled_task")

	cts := ctsSetup(t, srv, tempDir, disabledTaskConfig())

	// Confirm that terraform files were generated for a disabled task
	taskPath := filepath.Join(tempDir, disabledTaskName)
	files := testutils.CheckDir(t, true, taskPath)
	require.Greater(t, len(files), 0)
	testutils.CheckFile(t, true, taskPath, "terraform.tfvars.tmpl")

	// Confirm that resources were not created
	resourcesPath := filepath.Join(taskPath, resourcesDir)
	testutils.CheckDir(t, false, resourcesPath)

	// Update Task API: enable task with inspect run option
	baseUrl := fmt.Sprintf("http://localhost:%d/%s/tasks/%s",
		cts.Port(), "v1", disabledTaskName)
	u := fmt.Sprintf("%s?run=inspect", baseUrl)
	resp := testutils.RequestHTTP(t, http.MethodPatch, u, `{"enabled":true}`)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var r api.UpdateTaskResponse
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(&r)
	require.NoError(t, err)

	// Confirm inspect plan response: changes present, plan not empty
	assert.NotNil(t, r.Inspect)
	assert.True(t, r.Inspect.ChangesPresent)
	assert.NotEmpty(t, r.Inspect.Plan)

	// Confirm that resources were not generated during inspect mode
	testutils.CheckDir(t, false, resourcesPath)

	// Confirm that task remained disabled
	taskStatuses, err := cts.Status().Task(disabledTaskName, nil)
	require.NoError(t, err)
	status, ok := taskStatuses[disabledTaskName]
	require.True(t, ok)
	assert.False(t, status.Enabled, "task should still be disabled")

	// Update Task API: enable task with run now option
	u = fmt.Sprintf("%s?run=now", baseUrl)
	resp1 := testutils.RequestHTTP(t, http.MethodPatch, u, `{"enabled":true}`)
	defer resp1.Body.Close()

	// Confirm that resources are generated
	testutils.CheckDir(t, true, resourcesPath)

	// Update Task API: disable task
	resp2 := testutils.RequestHTTP(t, http.MethodPatch, baseUrl, `{"enabled":false}`)
	defer resp2.Body.Close()

	// Delete Resources
	err = os.RemoveAll(resourcesPath)
	require.NoError(t, err)

	// Register a new service
	service := testutil.TestService{
		ID:      "api-2",
		Name:    "api",
		Address: "5.6.7.8",
		Port:    8080,
	}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	time.Sleep(defaultWaitForNoEvent)

	// Confirm that resources are not recreated for disabled task
	testutils.CheckDir(t, false, resourcesPath)
}

// TestE2E_TaskEndpoints_Delete tests the delete task endpoint. This
// runs a Consul server and the CTS binary in daemon mode.
//	DELETE/v1/tasks/:task_name
func TestE2E_TaskEndpoints_Delete(t *testing.T) {
	setParallelism(t)
	// Test deleting a task
	// 1. Start with a task
	// 2. Delete the task
	// 3. Check that the task and events no longer exist
	// 4. Make a service change, check that no change is made

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()
	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "delete_task")
	taskName := "deleted_task"
	cts := ctsSetup(t, srv, tempDir,
		moduleTaskConfig(taskName, "./test_modules/local_instances_file"))

	// Delete task
	u := fmt.Sprintf("http://localhost:%d/%s/tasks/%s", cts.Port(), "v1", taskName)
	resp := testutils.RequestHTTP(t, http.MethodDelete, u, "")
	defer resp.Body.Close()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	// Check that the task no longer exists
	waitForTaskDeletion(t, cts, taskName, time.Second)

	// Make a change that would have triggered the task, expect no event
	service := testutil.TestService{
		ID:      "api",
		Name:    "api",
		Address: "5.6.7.8",
		Port:    8080,
	}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	time.Sleep(defaultWaitForNoEvent)
	s := fmt.Sprintf("http://localhost:%d/%s/status/tasks/%s",
		cts.Port(), "v1", taskName)
	resp = testutils.RequestHTTP(t, http.MethodGet, s, "")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resourcesPath := filepath.Join(tempDir, taskName, resourcesDir)
	validateServices(t, false, []string{"api"}, resourcesPath)
}

// TestE2E_TaskEndpoints_Delete_ActiveTask tests that a running task will not
// be deleted until it is complete.
//	DELETE/v1/tasks/:task_name
func TestE2E_TaskEndpoints_Delete_ActiveTask(t *testing.T) {
	setParallelism(t)
	// Test deleting a task
	// 1. Start with a task
	// 2. Trigger the task
	// 3. While task is still running, delete the task
	// 4. Check that the task and events still exist
	// 5. Wait until the task is complete
	// 6. Check that the task and events no longer exist

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "deleted_task_conflict")
	taskName := "deleted_task_conflict"
	cts := ctsSetup(t, srv, tempDir,
		moduleTaskConfig(taskName, "./test_modules/delayed_module"))

	// Trigger the task
	service := testutil.TestService{
		ID:      "api",
		Name:    "api",
		Address: "5.6.7.8",
		Port:    8080,
	}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

	// Attempt to delete the task while running, expect failure
	time.Sleep(2 * time.Second) // task completion is delayed by 5s
	u := fmt.Sprintf("http://localhost:%d/%s/tasks/%s", cts.Port(), "v1", taskName)
	resp := testutils.RequestHTTP(t, http.MethodDelete, u, "")
	defer resp.Body.Close()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	// Check that the task still exists
	count := eventCount(t, taskName, cts.Port())
	assert.Equal(t, 1, count)

	// Wait for task to be deleted
	waitForTaskDeletion(t, cts, taskName, defaultWaitForEvent+5*time.Second)

	// Check that task completed by checking created files
	resourcesPath := filepath.Join(tempDir, taskName, resourcesDir)
	validateServices(t, true, []string{"api"}, resourcesPath)
}

func waitForTaskDeletion(t *testing.T, client *api.Client, name string, timeout time.Duration) {
	polling := make(chan struct{})
	stopPolling := make(chan struct{})

	go func() {
		for {
			select {
			case <-stopPolling:
				return
			default:
				_, err := client.Status().Task(name, nil)
				if err == nil {
					time.Sleep(500 * time.Millisecond)
					continue
				}

				if !strings.Contains(err.Error(), "404") {
					// expecting only a 404 not found error
					t.Fatalf("\nUnexpected error when waiting for task to %s be deleted", name)
				}

				polling <- struct{}{}
				return
			}
		}
	}()

	select {
	case <-polling:
		return
	case <-time.After(timeout):
		close(stopPolling)
		t.Fatalf("\nError: timed out after waiting for %v task %q to be deleted\n",
			timeout, name)
	}
}

// TestE2E_TaskEndpoints_Create tests the create task endpoint. This
// runs a Consul server and the CTS binary in daemon mode.
//	POST /v1/tasks
func TestE2E_TaskEndpoints_Create(t *testing.T) {
	setParallelism(t)
	// Test creating a task
	// 1. Start with a task
	// 2. Create infrastructure change that would trigger new task
	// 3. Create a new task
	// 4. Check that the new task exists
	// 5. Check that the task did not run
	// 6. Make a service change to a service tracked by the new task, verify an event exists

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()
	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "create_task")
	initialTaskName := "initial-task"
	cts := ctsSetup(t, srv, tempDir,
		moduleTaskConfig(initialTaskName, "./test_modules/local_instances_file"))

	// Make a change that would trigger the new task, if it existed
	taskName := "created-task"
	serviceName := "testService"
	service := testutil.TestService{
		ID:      serviceName,
		Name:    serviceName,
		Address: "5.6.7.8",
		Port:    8080,
	}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

	// Create task
	u := fmt.Sprintf("http://localhost:%d/v1/tasks", cts.Port())

	createTaskRequest := fmt.Sprintf(createTestTaskTemplate, taskName, serviceName)

	resp := testutils.RequestHTTP(t, http.MethodPost, u, createTaskRequest)
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "response body %s", string(bodyBytes))

	// Check that the task has been created
	s := fmt.Sprintf("http://localhost:%d/%s/status/tasks/%s", cts.Port(), "v1", taskName)
	resp = testutils.RequestHTTP(t, http.MethodGet, s, "")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check that the status of the task is unknown because task has not run
	var taskStatuses map[string]api.TaskStatus
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&taskStatuses)
	require.NoError(t, err)
	task, ok := taskStatuses[taskName]
	require.True(t, ok)
	assert.True(t, task.Enabled)
	assert.Equal(t, "unknown", task.Status)

	// Check no events stored
	e := eventCount(t, taskName, cts.Port())
	require.Equal(t, 0, e)

	// Verify that the task did not run
	resourcesPath := filepath.Join(tempDir, taskName, resourcesDir)
	validateServices(t, false, []string{service.ID}, resourcesPath)

	// Make a change that triggers the new task, verify that it runs
	service = testutil.TestService{
		ID:      serviceName + "-2",
		Name:    serviceName,
		Address: "5.6.7.9",
		Port:    8080,
	}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	api.WaitForEvent(t, cts, taskName, time.Now(), defaultWaitForEvent)

	e = eventCount(t, taskName, cts.Port())
	require.Equal(t, 1, e, "event is only tracked when the task runs")

	resourcesPath = filepath.Join(tempDir, taskName, resourcesDir)
	validateServices(t, true, []string{service.ID}, resourcesPath)
}

// TestE2E_TaskEndpoints_Create_Run_Now tests the create task endpoint with run now.
// This runs a Consul server and the CTS binary in daemon mode.
// POST /v1/tasks
func TestE2E_TaskEndpoints_Create_Run_Now(t *testing.T) {
	setParallelism(t)
	// Test creating a task
	// 1. Start with a task
	// 2. Create infrastructure change that would trigger new task
	// 3. Create a new task with run=now
	// 4. Check that the new task exists
	// 5. Check that new task ran based on (2) infrastructure change

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()
	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "create_task_run_now")
	initialTaskName := "initial-task"
	cts := ctsSetup(t, srv, tempDir,
		moduleTaskConfig(initialTaskName, "./test_modules/local_instances_file"))

	// Make a change that would trigger the new task, if it existed
	taskName := "created-task"
	serviceName := "testService"
	service := testutil.TestService{
		ID:      serviceName,
		Name:    serviceName,
		Address: "5.6.7.8",
		Port:    8080,
	}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

	// Create task
	u := fmt.Sprintf("http://localhost:%d/v1/tasks", cts.Port())
	u = fmt.Sprintf("%s?run=now", u)

	createTaskRequest := fmt.Sprintf(createTestTaskTemplate, taskName, serviceName)

	resp := testutils.RequestHTTP(t, http.MethodPost, u, createTaskRequest)
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "response body %s", string(bodyBytes))

	// Check that the task has been created
	s := fmt.Sprintf("http://localhost:%d/%s/status/tasks/%s", cts.Port(), "v1", taskName)
	resp = testutils.RequestHTTP(t, http.MethodGet, s, "")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify that the task did run and only a single event was stored
	e := events(t, taskName, cts.Port())
	require.Equal(t, 1, len(e))
	resourcesPath := filepath.Join(tempDir, taskName, resourcesDir)
	validateServices(t, true, []string{service.ID}, resourcesPath)
}

// TestE2E_TaskEndpoints_InvalidSchema tests the create task endpoint with an invalid schema, no task
// should be created. This runs a Consul server and the CTS binary in daemon mode.
//	POST /v1/tasks
func TestE2E_TaskEndpoints_InvalidSchema(t *testing.T) {
	setParallelism(t)
	// Test deleting a task
	// 1. Start with a task
	// 2. Attempt to create a new task with an invalid schema
	// 3. Check that the new task does not exist

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()
	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "create_task_invalid_schema")
	initialTaskName := "initial-task"
	cts := ctsSetup(t, srv, tempDir,
		moduleTaskConfig(initialTaskName, "./test_modules/local_instances_file"))

	// Create a task with invalid module field (boolean instead of string)
	u := fmt.Sprintf("http://localhost:%d/v1/tasks", cts.Port())

	taskName := "created-task"
	badRequest := fmt.Sprintf(`{
		"task": {
			"description": "Writes the service name, id, and IP address to a file",
			"enabled": true,
			"name": "%s",
			"providers": [
				"local"
			],
			"condition": {
				"services": {
					"names": [
						"api"
					]
				}
			},
			"module": true
		}
	}`, taskName)

	resp := testutils.RequestHTTP(t, http.MethodPost, u, badRequest)
	defer resp.Body.Close()
	var errorResponse oapigen.ErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&errorResponse)
	require.NoError(t, err)

	assert.Contains(t, errorResponse.Error.Message, `request body has an error: doesn't match the schema: `+
		`Error at "/task/module": Field must be set to string or not be present`)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Check that the task has not been created
	s := fmt.Sprintf("http://localhost:%d/%s/status/tasks/%s", cts.Port(), "v1", taskName)
	resp = testutils.RequestHTTP(t, http.MethodGet, s, "")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestE2E_TaskEndpoints_DryRunTaskCreate tests the dry run task API
// inspects the given task and discards it at the end of the run by:
//
// 1. Creating a dry run task
// 2. Verifying that the task and Terraform plan output is returned
// 3. Checking that there are no events for the task
// 4. Checking that no resources were created
// 5. Making a change that would trigger the task if it had been created
// 6. Verifying again that no events or resources are created
func TestE2E_TaskEndpoints_DryRunTaskCreate(t *testing.T) {
	setParallelism(t)
	// Start Consul and CTS
	srv := newTestConsulServer(t)
	defer srv.Stop()
	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "create_dry_run_task")
	initialTaskName := "initial-task"
	cts := ctsSetup(t, srv, tempDir,
		moduleTaskConfig(initialTaskName, "mkam/hello/cts"))

	// Create a dry run task
	u := fmt.Sprintf("http://localhost:%d/v1/tasks?run=inspect", cts.Port())
	taskName := "dryrun_task"
	serviceName := "api"
	req := &oapigen.TaskRequest{
		Task: oapigen.Task{
			Name: taskName,
			Condition: oapigen.Condition{
				Services: &oapigen.ServicesCondition{
					Names: &[]string{serviceName},
				},
			},

			Module: "mkam/hello/cts",
		},
	}
	resp := testutils.RequestJSON(t, http.MethodPost, u, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse response body
	defer resp.Body.Close()
	var r oapigen.TaskResponse
	err := json.NewDecoder(resp.Body).Decode(&r)
	require.NoError(t, err)
	assert.NotEmpty(t, r.RequestId, "expected request ID in response")

	// Verify run in response
	assert.NotNil(t, r.Run)
	assert.Contains(t, *r.Run.Plan, fmt.Sprintf("Hello, %s!", serviceName))
	assert.Contains(t, *r.Run.Plan, "Plan: 2 to add, 0 to change, 0 to destroy.")

	// Verify task in response
	assert.NotNil(t, r.Task)
	assert.Equal(t, req.Task.Name, r.Task.Name, "name not expected value")
	assert.Equal(t, req.Task.Module, r.Task.Module, "module not expected value")
	assert.ElementsMatch(t, *req.Task.Condition.Services.Names, *r.Task.Condition.Services.Names, "services not expected value")

	// Check that the task was not created
	s := fmt.Sprintf("http://localhost:%d/%s/status/tasks/%s", cts.Port(), "v1", taskName)
	resp = testutils.RequestHTTP(t, http.MethodGet, s, "")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Verify that the resources were not actually created
	resourcesPath := filepath.Join(tempDir, taskName, resourcesDir)
	validateServices(t, false, []string{serviceName}, resourcesPath)

	// Make a change that would trigger the task if it had been created
	service := testutil.TestService{
		ID:      serviceName + "-2",
		Name:    serviceName,
		Address: "5.6.7.9",
		Port:    8080,
	}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	time.Sleep(defaultWaitForNoEvent)

	// Verify that there still is no task
	resp = testutils.RequestHTTP(t, http.MethodGet, s, "")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	validateServices(t, false, []string{serviceName}, resourcesPath)
}

// TestE2E_TaskEndpoints_Get tests the Get Task API: GET /v1/tasks/:task_name
// It runs a Consul server and the CTS binary in daemon mode.
func TestE2E_TaskEndpoints_Get(t *testing.T) {
	setParallelism(t)

	// 0. Start CTS with a task
	// 1. Try retrieving the start-up task and verify response payload
	// 2. Try retrieving a non-existent task and check status code

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()
	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "get_task_api")
	taskName := "get_task"
	module := "mkam/hello/cts"

	cts := ctsSetup(t, srv, tempDir, moduleTaskConfig(taskName, module))

	// 1. Check retrieving the existing task
	u := fmt.Sprintf("http://localhost:%d/%s/tasks/%s", cts.Port(), "v1", taskName)
	resp := testutils.RequestHTTP(t, http.MethodGet, u, "")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse response body
	var r oapigen.TaskResponse
	err := json.NewDecoder(resp.Body).Decode(&r)
	require.NoError(t, err)
	assert.NotEmpty(t, r.RequestId, "expected request ID in response")

	// Verify basic task info in response
	assert.NotNil(t, r.Task)
	assert.Equal(t, taskName, r.Task.Name, "name not expected value")
	assert.Equal(t, module, r.Task.Module, "module not expected value")
	assert.Empty(t, r.Task.TerraformVersion, "unexpected enterprise field value")

	// 2. Check retrieving a non-existing task
	u = fmt.Sprintf("http://localhost:%d/%s/tasks/%s", cts.Port(), "v1", "non-existent-task")
	resp = testutils.RequestHTTP(t, http.MethodGet, u, "")
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// checkEvents does some basic checks to loosely ensure returned events in
// responses are as expected
func checkEvents(t *testing.T, taskStatuses map[string]api.TaskStatus,
	taskName string, expect bool) {

	task, ok := taskStatuses[taskName]

	if expect {
		assert.True(t, ok)
	} else {
		assert.False(t, ok)
		return
	}

	// there should be 2-4 events
	msg := fmt.Sprintf("%s expected 2-4 events, got %d", taskName, len(task.Events))
	require.True(t, 2 <= len(task.Events) && len(task.Events) <= 4, msg)

	for ix, e := range task.Events {
		assert.Equal(t, taskName, e.TaskName)

		require.NotNil(t, e.Config)
		assert.Equal(t, []string{"fake-sync"}, e.Config.Providers)
		wd, err := os.Getwd()
		assert.NoError(t, err)
		module := filepath.Join(wd, "./test_modules/local_instances_file")
		assert.Equal(t, module, e.Config.Source)

		if taskName == fakeSuccessTaskName {
			assert.True(t, e.Success)
		}

		if taskName == fakeFailureTaskName {
			// last event should be successful, others failure
			msg := fmt.Sprintf("Event %d of %d: %v", ix+1, len(task.Events), e)
			if ix == len(task.Events)-1 {
				assert.True(t, e.Success, msg)
			} else {
				assert.False(t, e.Success, msg)
			}
		}
	}
}
