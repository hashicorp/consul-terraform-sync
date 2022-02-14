//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/command"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	scheduledConsulKV = `task {
	name = "%s"
	module = "./test_modules/consul_kv_file"
	condition "schedule" {
			cron = "*/10 * * * * * *"
	}
	module_input "consul-kv" {
		path = "key-path"
		datacenter = "dc1"
	}
	module_input "services" {
		names = ["web", "api"]
	}
}`

	scheduledServices = `task {
	name = "%s"
	module = "./test_modules/local_instances_file"
	condition "schedule" {
		cron = "*/10 * * * * * *"
	}
	module_input "services"{
		regexp = "^web.*|^api.*"
	}
}`
)

// TestCondition_Schedule_Basic runs CTS in daemon-mode to test a task
// configured with a schedule condition and monitoring task.services,
// task.module_input or both. This test confirms some basic schedule condition
// behavior:
// 1. Task successfully passes through once-mode and does not hang
// 2. Task runs at the scheduled interval even when no dependency changes
// 3. New dependencies do not trigger the task to run ahead of scheduled time
// 4. Task can handle multiple dependency changes
func TestCondition_Schedule_Basic(t *testing.T) {
	setParallelism(t)

	taskName := "scheduled_task"
	conditionWithServices := fmt.Sprintf(`task {
	name = "%s"
	services = ["api", "web"]
	module = "./test_modules/local_instances_file"
	condition "schedule" {
		cron = "*/10 * * * * * *"
	}
}
`, taskName)
	moduleInputServices := fmt.Sprintf(scheduledServices, taskName)
	moduleInputConsulKV := fmt.Sprintf(scheduledConsulKV, taskName)

	testcases := []struct {
		name          string
		conditionTask string
		tempDir       string
		isConsulKV    bool
	}{
		{
			name:          "with services",
			conditionTask: conditionWithServices,
			tempDir:       "schedule_basic_services",
			isConsulKV:    false,
		},
		{
			name:          "with module_input services",
			conditionTask: moduleInputServices,
			tempDir:       "schedule_basic_module_input",
			isConsulKV:    false,
		},
		{
			name:          "with module_input consul_kv",
			conditionTask: moduleInputConsulKV,
			tempDir:       "schedule_consulKV",
			isConsulKV:    true,
		},
	}

	for _, tc := range testcases {
		tc := tc // rebind tc into this lexical scope for parallel use
		t.Run(tc.name, func(t *testing.T) {
			setParallelism(t) // In the CI environment, run table tests in parallel as they can take a lot of time
			srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
				HTTPSRelPath: "../testutils",
			})
			defer srv.Stop()

			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, tc.tempDir)
			cts := ctsSetup(t, srv, tempDir, tc.conditionTask)

			// Test schedule condition overall behavior:
			// 0. Confirm baseline: check current number of events for each task.
			// 1. Make no dependency changes but confirm that task is still triggered at
			//    scheduled time.
			// 2. Register multiple services and confirm that task is only triggered at
			//    scheduled time. Check resources are created.

			port := cts.Port()
			taskSchedule := 10 * time.Second
			scheduledWait := taskSchedule + 7*time.Second // buffer for task to execute

			// 0. Confirm one event for once-mode
			eventCountBase := eventCount(t, taskName, port)
			assert.Equal(t, 1, eventCountBase)

			// 1. Wait and confirm that the task was triggered at the scheduled time
			// Special confirmation case: when task is run in once-mode, it runs on
			// demand vs. on schedule. Then the first run in daemon-mode starts on the
			// 10s mark. Therefore these two runs are not necessarily 10s apart.
			beforeEvent := time.Now()
			api.WaitForEvent(t, cts, taskName, beforeEvent, scheduledWait)

			eventCountNow := eventCount(t, taskName, port)
			assert.Equal(t, eventCountNow, eventCountBase+1)

			e := events(t, taskName, port)
			latestStartTime := e[0].StartTime.Round(time.Second)
			assert.Equal(t, 0, latestStartTime.Second()%10, fmt.Sprintf("expected "+
				"start time to be at the 10s mark but was at %s", latestStartTime))

			// 2. Register two new services. Confirm task only triggered on schedule

			// wait for scheduled task to have just ran. then register consul services and
			// a Consul KV pair
			api.WaitForEvent(t, cts, taskName, time.Now(), scheduledWait)
			registerTime := time.Now()
			services := []testutil.TestService{{ID: "api-1", Name: "api"},
				{ID: "web-1", Name: "web"}}
			testutils.AddServices(t, srv, services)
			expectedKV := "red"
			srv.SetKVString(t, "key-path", expectedKV)

			// check scheduled task did not trigger immediately and ran only on schedule
			api.WaitForEvent(t, cts, taskName, registerTime, scheduledWait)
			checkScheduledRun(t, taskName, registerTime, taskSchedule, port)

			// confirm service resources created
			resourcesPath := filepath.Join(tempDir, taskName, resourcesDir)
			validateServices(t, true, []string{"api-1", "web-1"}, resourcesPath)

			if tc.isConsulKV {
				// confirm key-value resources created, and that the values are as expected
				validateModuleFile(t, true, true, resourcesPath, "key-path", expectedKV)
			}
		})
	}
}

// TestCondition_Schedule_Dynamic runs CTS in daemon-mode to test running a
// scheduled task and a dynamic task. This test confirms that the two types
// of tasks can co-exist and operate as expected.
func TestCondition_Schedule_Dynamic(t *testing.T) {
	setParallelism(t)

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "schedule_dynamic")
	schedTaskName := "scheduled_task"
	taskSchedule := 10 * time.Second
	conditionTask := fmt.Sprintf(`task {
	name = "%s"
	module = "./test_modules/local_instances_file"
	condition "schedule" {
		cron = "*/10 * * * * * *"
	}
	module_input "services" {
		names = ["api", "web"]
	}
}
`, schedTaskName)
	tasks := fmt.Sprintf("%s\n\n%s", conditionTask, dbTask())
	cts := ctsSetup(t, srv, tempDir, tasks)

	// Configured tasks:
	// Scheduled task: monitors api and web.
	// Dynamic task: monitors api and db.

	// Test three test-cases for registering a service:
	// 1. Register a service only monitored by dynamic task (db):
	//     - Dynamic task: triggered immediately
	//     - Scheduled task: run on schedule
	// 2. Register a service only monitored by scheduled task (web):
	//     - Dynamic task: not triggered
	//     - Scheduled task: run on schedule
	// 3. Register a service monitored by both (api)
	//     - Dynamic task: triggered immediately
	//     - Scheduled task: run on schedule

	port := cts.Port()
	scheduledWait := taskSchedule + 5*time.Second // buffer for task to execute

	// 0. Confirm one event for once-mode
	schedEventCount := eventCount(t, schedTaskName, port)
	require.Equal(t, 1, schedEventCount)
	dynaEventCounter := eventCount(t, dbTaskName, port)
	require.Equal(t, 1, dynaEventCounter)

	// 1. test registering a service only monitored by dynamic task

	// wait for scheduled task to have just ran. then register db
	api.WaitForEvent(t, cts, schedTaskName, time.Now(), scheduledWait)
	registrationTime := time.Now()
	service := testutil.TestService{ID: "db-1", Name: "db"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

	// confirm dynamic task triggered
	api.WaitForEvent(t, cts, dbTaskName, registrationTime, defaultWaitForEvent)
	dynaEventCounter++
	dynaEventCountNow := eventCount(t, dbTaskName, port)
	require.Equal(t, dynaEventCounter, dynaEventCountNow,
		"dynamic task event should have been incremented")

	// check scheduled task didn't trigger immediately and ran on schedule
	api.WaitForEvent(t, cts, schedTaskName, registrationTime, scheduledWait)
	checkScheduledRun(t, schedTaskName, registrationTime, taskSchedule, port)

	// 2. test registering a service only monitored by scheduled task

	// wait for scheduled task to have just ran. then register web
	api.WaitForEvent(t, cts, schedTaskName, time.Now(), scheduledWait)
	registrationTime = time.Now()
	service = testutil.TestService{ID: "web-1", Name: "web"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

	// check scheduled task didn't trigger immediately and ran on schedule
	api.WaitForEvent(t, cts, schedTaskName, registrationTime, scheduledWait)
	checkScheduledRun(t, schedTaskName, registrationTime, taskSchedule, port)

	// confirm that dynamic task has not triggered by now
	dynaEventCountNow = eventCount(t, dbTaskName, port)
	require.Equal(t, dynaEventCounter, dynaEventCountNow,
		"dynamic event count unexpectedly changed")

	// 3. test registering a service monitored by scheduled and dynamic tasks

	// wait for scheduled task to have just ran. then register api
	api.WaitForEvent(t, cts, schedTaskName, time.Now(), scheduledWait)
	registrationTime = time.Now()
	service = testutil.TestService{ID: "api-1", Name: "api"}
	testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

	// confirm dynamic task triggered
	api.WaitForEvent(t, cts, dbTaskName, registrationTime, defaultWaitForEvent)
	dynaEventCounter++
	dynaEventCountNow = eventCount(t, dbTaskName, port)
	require.Equal(t, dynaEventCounter, dynaEventCountNow,
		"dynamic task event should have been incremented")

	// check scheduled task didn't trigger immediately and ran on schedule
	api.WaitForEvent(t, cts, schedTaskName, registrationTime, scheduledWait)
	checkScheduledRun(t, schedTaskName, registrationTime, taskSchedule, port)
}

// TestCondition_Schedule_CreateAndDeleteCLI runs CTS in daemon-mode
// with an initial scheduled task, creates a new scheduled task with
// the CLI, and deletes both tasks with the CLI. This test confirms
// that the newly created task runs on a schedule, and that the
// deleted tasks no longer run.
func TestCondition_Schedule_CreateAndDeleteCLI(t *testing.T) {
	t.Parallel()

	initTaskName := "init_scheduled_task"
	initTask := fmt.Sprintf(scheduledServices, initTaskName)
	taskName := "created_scheduled_task"
	createdTask := fmt.Sprintf(scheduledConsulKV, taskName)

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "schedule_create_delete_cli")
	cts := ctsSetup(t, srv, tempDir, initTask)

	taskSchedule := 10 * time.Second
	scheduledWait := taskSchedule + 7*time.Second // buffer for task to execute

	// Create a scheduled task with the CLI and a task file
	var taskConfig hclConfig
	taskConfig = taskConfig.appendString(createdTask)
	taskFilePath := filepath.Join(tempDir, "task.hcl")
	taskConfig.write(t, taskFilePath)

	subcmd := []string{"task", "create",
		fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
		fmt.Sprintf("-task-file=%s", taskFilePath),
	}
	out, err := runSubcommand(t, "yes\n", subcmd...)
	require.NoError(t, err, fmt.Sprintf("command '%s' failed:\n %s", subcmd, out))
	require.Contains(t, out, fmt.Sprintf("Task '%s' created", taskName))

	// 0. Confirm one event for when the task was created
	port := cts.Port()
	eventCountBase := eventCount(t, taskName, port)
	assert.Equal(t, 1, eventCountBase, "expected a scheduled task event after creation")

	// 1. Wait and confirm that the task was triggered at the scheduled time
	beforeEvent := time.Now()
	api.WaitForEvent(t, cts, taskName, beforeEvent, scheduledWait)

	eventCountNow := eventCount(t, taskName, port)
	assert.Equal(t, eventCountNow, eventCountBase+1)

	e := events(t, taskName, port)
	latestStartTime := e[0].StartTime.Round(time.Second)
	assert.Equal(t, 0, latestStartTime.Second()%10, fmt.Sprintf("expected "+
		"start time to be at the 10s mark but was at %s", latestStartTime))

	// 2. Register two new services and a Consul KV pair. Confirm task only triggered on schedule

	// wait for scheduled task to have just ran. then register consul services
	api.WaitForEvent(t, cts, taskName, time.Now(), scheduledWait)
	registerTime := time.Now()
	services := []testutil.TestService{{ID: "api-1", Name: "api"},
		{ID: "web-1", Name: "web"}}
	testutils.AddServices(t, srv, services)
	expectedKV := "red"
	srv.SetKVString(t, "key-path", expectedKV)

	// check scheduled task did not trigger immediately and ran only on schedule
	api.WaitForEvent(t, cts, taskName, registerTime, scheduledWait)
	checkScheduledRun(t, taskName, registerTime, taskSchedule, port)

	// confirm resources created
	resourcesPath := filepath.Join(tempDir, taskName, resourcesDir)
	validateServices(t, true, []string{"api-1", "web-1"}, resourcesPath)
	validateModuleFile(t, true, true, resourcesPath, "key-path", expectedKV)

	// 3. Delete both the scheduled tasks
	deleteCmd := []string{"task", "delete", "-auto-approve",
		fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
	}

	for _, n := range []string{taskName, initTaskName} {
		subcmd = append(deleteCmd, n)
		out, err = runSubcommand(t, "", subcmd...)
		require.NoError(t, err, fmt.Sprintf("command '%s' failed:\n %s", subcmd, out))
		require.Contains(t, out, fmt.Sprintf("Task '%s' has been marked for deletion", n))

		s := fmt.Sprintf("http://localhost:%d/%s/status/tasks/%s", cts.Port(), "v1", n)
		resp := testutils.RequestHTTP(t, http.MethodGet, s, "")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	}

	// check that tasks no longer run at the scheduled time
	// modify services in Consul and wait for scheduled time
	services = []testutil.TestService{{ID: "api-2", Name: "api"}}
	testutils.AddServices(t, srv, services)
	testutils.DeregisterConsulService(t, srv, "web-1")
	testutils.DeregisterConsulService(t, srv, "api-1")
	time.Sleep(scheduledWait)

	for _, n := range []string{taskName, initTaskName} {
		// confirm no events still and that resources aren't changed
		s := fmt.Sprintf("http://localhost:%d/%s/status/tasks/%s", cts.Port(), "v1", n)
		resp := testutils.RequestHTTP(t, http.MethodGet, s, "")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		resourcesPath := filepath.Join(tempDir, n, resourcesDir)
		validateServices(t, true, []string{"api-1", "web-1"}, resourcesPath)
		validateServices(t, false, []string{"api-2"}, resourcesPath)
	}
}

// TestCondition_Schedule_CreateCLICanceled tests if a task is not given
// approval to be created, it will not run on a schedule.
func TestCondition_Schedule_CreateCLICanceled(t *testing.T) {
	t.Parallel()
	taskName := "scheduled_task_create_canceled"
	createdTask := fmt.Sprintf(scheduledConsulKV, taskName)

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()
	srv.SetKVString(t, "key-path", "test")

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "schedule_create_cli_canceled")
	cts := ctsSetup(t, srv, tempDir, dbTask())

	// Initiate a scheduled task creation but cancel at approval step
	var taskConfig hclConfig
	taskConfig = taskConfig.appendString(createdTask)
	taskFilePath := filepath.Join(tempDir, "task.hcl")
	taskConfig.write(t, taskFilePath)

	subcmd := []string{"task", "create",
		fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
		fmt.Sprintf("-task-file=%s", taskFilePath),
	}
	out, err := runSubcommand(t, "no\n", subcmd...)
	require.NoError(t, err, fmt.Sprintf("command '%s' failed:\n %s", subcmd, out))
	require.Contains(t, out, "Cancelled creating task")

	// Make changes to Consul service and KV pair
	services := []testutil.TestService{{ID: "api-1", Name: "api"},
		{ID: "web-1", Name: "web"}}
	testutils.AddServices(t, srv, services)

	// Wait for scheduled time, confirm no events or resources
	taskSchedule := 10 * time.Second
	scheduledWait := taskSchedule + 7*time.Second // buffer for task to execute
	time.Sleep(scheduledWait)

	s := fmt.Sprintf("http://localhost:%d/%s/status/tasks/%s", cts.Port(), "v1", taskName)
	resp := testutils.RequestHTTP(t, http.MethodGet, s, "")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	resourcesPath := filepath.Join(tempDir, taskName, resourcesDir)
	validateServices(t, false, []string{"api-1", "web-1"}, resourcesPath)
	validateModuleFile(t, true, false, resourcesPath, "key-path", "")
}

// checkScheduledRun checks that a scheduled task's most recent task run
// occurred at approximately the expected scheduled time by checking events
func checkScheduledRun(t *testing.T, taskName string, depChangeTime time.Time,
	taskSchedule time.Duration, port int) {

	e := events(t, taskName, port)
	require.GreaterOrEqual(t, len(e), 2, "expect at least two events. cannot "+
		"use this check for first event. first event is from once-mode which "+
		"calls task on demand, not on schedule")

	// confirm that latest event is caused by latest dependency change
	latestEventStartTime := e[0].StartTime
	assert.True(t, latestEventStartTime.After(depChangeTime), fmt.Sprintf(
		"expected latest event time %s to be after dep change time %s",
		latestEventStartTime, depChangeTime))

	// confirm that prior event happened before latest dependency change
	priorEventStartTime := e[1].StartTime
	assert.True(t, priorEventStartTime.Before(depChangeTime), fmt.Sprintf(
		"expected prior event time %s to be before dep change time %s",
		priorEventStartTime, depChangeTime))

	// confirm latest event is approximately at the scheduled interval after
	// prior event
	latestEventStartTime = e[0].StartTime.Round(time.Second)
	priorEventStartTime = e[1].StartTime.Round(time.Second)
	assert.Equal(t, priorEventStartTime.Add(taskSchedule), latestEventStartTime)
}
