//go:build e2e
// +build e2e

// Tests CTS API endpoints /v1/status and /v1/tasks
package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_StatusEndpoints tests all of the CTS status endpoints and query
// parameters. This runs a Consul server and the CTS binary in daemon mode.
//	GET	/v1/status/tasks
// 	GET	/v1/status/tasks/:task_name
//	GET	/v1/status
func TestE2E_StatusEndpoints(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "status_endpoints")
	delete := testutils.MakeTempDir(t, tempDir)
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
				fakeSuccessTaskName: api.TaskStatus{
					TaskName:  fakeSuccessTaskName,
					Status:    api.StatusSuccessful,
					Enabled:   true,
					Providers: []string{"fake-sync"},
					Services:  []string{"api"},
					EventsURL: "/v1/status/tasks/fake_handler_success_task?include=events",
				},
				fakeFailureTaskName: api.TaskStatus{
					TaskName:  fakeFailureTaskName,
					Status:    api.StatusErrored,
					Enabled:   true,
					Providers: []string{"fake-sync"},
					Services:  []string{"api"},
					EventsURL: "/v1/status/tasks/fake_handler_failure_task?include=events",
				},
				disabledTaskName: api.TaskStatus{
					TaskName:  disabledTaskName,
					Status:    api.StatusUnknown,
					Enabled:   false,
					Providers: []string{"fake-sync"},
					Services:  []string{"api"},
					EventsURL: "",
				},
			},
		},
		{
			"existing single task status",
			"status/tasks/" + fakeSuccessTaskName,
			http.StatusOK,
			map[string]api.TaskStatus{
				fakeSuccessTaskName: api.TaskStatus{
					TaskName:  fakeSuccessTaskName,
					Status:    api.StatusSuccessful,
					Enabled:   true,
					Providers: []string{"fake-sync"},
					Services:  []string{"api"},
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
	delete()
}

// TestE2E_TaskEndpoints_UpdateEnableDisable tests the tasks endpoints. This
// runs a Consul server and the CTS binary in daemon mode.
//	PATCH	/v1/tasks/:task_name
func TestE2E_TaskEndpoints_UpdateEnableDisable(t *testing.T) {
	t.Parallel()
	// Test enabling and disabling a task
	// 1. Start with disabled task. Confirm task not initialized, resources not
	//    created
	// 2. API to inspect enabling task. Confirm plan looks good, resources not
	//    created, and task not actually enabled.
	// 3. API to actually enable task. Confirm resources are created
	// 4. API to disable task. Delete resources. Register new service. Confirm
	//    new service registering does not trigger creating resources

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "disabled_task")

	cts := ctsSetup(t, srv, tempDir, disabledTaskConfig(tempDir))

	// Confirm that terraform files were not generated for a disabled task
	files := testutils.CheckDir(t, true, fmt.Sprintf("%s/%s", tempDir, "disabled_task"))
	require.Equal(t, len(files), 0)

	// Confirm that resources were not created
	resourcesPath := filepath.Join(tempDir, disabledTaskName, resourcesDir)
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
	t.Parallel()
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
	u := fmt.Sprintf("http://localhost:%d/%s/tasks/%s",
		cts.Port(), "v1", taskName)
	resp := testutils.RequestHTTP(t, http.MethodDelete, u, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Check that the task no longer exists
	s := fmt.Sprintf("http://localhost:%d/%s/status/tasks/%s",
		cts.Port(), "v1", taskName)
	resp = testutils.RequestHTTP(t, http.MethodGet, s, "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Make a change that would have triggered the task, expect no event
	service := testutil.TestService{
		ID:      "api",
		Name:    "api",
		Address: "5.6.7.8",
		Port:    8080,
	}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
	time.Sleep(defaultWaitForNoEvent)
	resp = testutils.RequestHTTP(t, http.MethodGet, s, "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resourcesPath := filepath.Join(tempDir, taskName, resourcesDir)
	validateServices(t, false, []string{"api"}, resourcesPath)
}

// TestE2E_TaskEndpoints_Delete_Conflict tests that a running task cannot
// be deleted. This runs a Consul server and the CTS binary in daemon mode.
//	DELETE/v1/tasks/:task_name
func TestE2E_TaskEndpoints_Delete_Conflict(t *testing.T) {
	t.Parallel()
	// Test deleting a task
	// 1. Start with a task
	// 2. Trigger the task
	// 3. While task is still running, delete the task
	// 4. Check that the task and events still exist
	// 5. Delete the task after it is complete
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
	now := time.Now()
	service := testutil.TestService{
		ID:      "api",
		Name:    "api",
		Address: "5.6.7.8",
		Port:    8080,
	}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

	// Attempt to delete the task while running, expect failure
	time.Sleep(2 * time.Second) // task completion is delayed by 5s
	u := fmt.Sprintf("http://localhost:%d/%s/tasks/%s",
		cts.Port(), "v1", taskName)
	resp := testutils.RequestHTTP(t, http.MethodDelete, u, "")
	require.Equal(t, http.StatusConflict, resp.StatusCode)

	// Check that the task still exists, wait for it to complete
	api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)
	resourcesPath := filepath.Join(tempDir, taskName, resourcesDir)
	validateServices(t, true, []string{"api"}, resourcesPath)

	// Delete task now that it is completed
	resp = testutils.RequestHTTP(t, http.MethodDelete, u, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Check that the task no longer exists
	s := fmt.Sprintf("http://localhost:%d/%s/status/tasks/%s",
		cts.Port(), "v1", taskName)
	resp = testutils.RequestHTTP(t, http.MethodGet, s, "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
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
		assert.Equal(t, []string{"api"}, e.Config.Services)
		wd, err := os.Getwd()
		assert.NoError(t, err)
		source := filepath.Join(wd, "./test_modules/local_instances_file")
		assert.Equal(t, source, e.Config.Source)

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
