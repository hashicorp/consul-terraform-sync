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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCondition_Schedule_Basic runs CTS in daemon-mode to test a task
// configured with a schedule condition and monitoring either task.services or task.source_input. This test
// confirms some basic schedule condition behavior:
// 1. Task successfully passes through once-mode and does not hang
// 2. Task runs at the scheduled interval even when no dependency changes
// 3. New dependencies do not trigger the task to run ahead of scheduled time
// 4. Task can handle multiple dependency changes
func TestCondition_Schedule_Basic(t *testing.T) {
	t.Parallel()

	taskName := "scheduled_task"
	conditionWithServices := fmt.Sprintf(`task {
	name = "%s"
	services = ["api", "web"]
	source = "./test_modules/local_instances_file"
	condition "schedule" {
		cron = "*/10 * * * * * *"
	}
}
`, taskName)
	conditionWithSourceInput := fmt.Sprintf(`task {
	name = "%s"
	source = "./test_modules/local_instances_file"
	condition "schedule" {
		cron = "*/10 * * * * * *"
	}
    source_input "services"{
	    regexp = "^web.*|^api.*"
    }
}
`, taskName)

	testcases := []struct {
		name          string
		conditionTask string
		tempDir       string
	}{
		{
			"with services",
			conditionWithServices,
			"schedule_basic_services",
		},
		{
			"with source_input",
			conditionWithSourceInput,
			"schedule_basic_source_input",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
				HTTPSRelPath: "../testutils",
			})
			defer srv.Stop()

			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, tc.tempDir)
			cleanup := testutils.MakeTempDir(t, tempDir)

			taskSchedule := 10 * time.Second

			config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock().
				appendString(tc.conditionTask)
			configPath := filepath.Join(tempDir, configFile)
			config.write(t, configPath)

			cts, stop := api.StartCTS(t, configPath)
			defer stop(t)
			err := cts.WaitForAPI(defaultWaitForAPI)
			require.NoError(t, err)

			// Test schedule condition overall behavior:
			// 0. Confirm baseline: check current number of events for each task.
			// 1. Make no dependency changes but confirm that task is still triggered at
			//    scheduled time.
			// 2. Register multiple services and confirm that task is only triggered at
			//    scheduled time. Check resources are created.

			port := cts.Port()
			scheduledWait := taskSchedule + 5*time.Second // buffer for task to execute

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

			// wait for scheduled task to have just ran. then register consul services
			api.WaitForEvent(t, cts, taskName, time.Now(), scheduledWait)
			registerTime := time.Now()
			services := []testutil.TestService{{ID: "api-1", Name: "api"},
				{ID: "web-1", Name: "web"}}
			testutils.AddServices(t, srv, services)

			// check scheduled task did not trigger immediately and ran only on schedule
			api.WaitForEvent(t, cts, taskName, registerTime, scheduledWait)
			checkScheduledRun(t, taskName, registerTime, taskSchedule, port)

			// confirm resources created
			resourcesPath := filepath.Join(tempDir, taskName, resourcesDir)
			testutils.CheckFile(t, true, resourcesPath, "api-1.txt")
			testutils.CheckFile(t, true, resourcesPath, "web-1.txt")

			cleanup()
		})
	}
}

// TestCondition_Schedule_Dynamic runs CTS in daemon-mode to test running a
// scheduled task and a dynamic task. This test confirms that the two types
// of tasks can co-exist and operate as expected.
func TestCondition_Schedule_Dynamic(t *testing.T) {
	t.Parallel()

	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "schedule_dynamic")
	cleanup := testutils.MakeTempDir(t, tempDir)

	schedTaskName := "scheduled_task"
	taskSchedule := 10 * time.Second
	conditionTask := fmt.Sprintf(`task {
	name = "%s"
	services = ["api", "web"]
	source = "./test_modules/local_instances_file"
	condition "schedule" {
		cron = "*/10 * * * * * *"
	}
}
`, schedTaskName)

	config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock().
		appendDBTask().appendString(conditionTask)
	configPath := filepath.Join(tempDir, configFile)
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)
	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)

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

	cleanup()
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
