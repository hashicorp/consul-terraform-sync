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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	t.Parallel()

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
	conditionWithSourceInput := fmt.Sprintf(`task {
	name = "%s"
	module = "./test_modules/local_instances_file"
	condition "schedule" {
		cron = "*/10 * * * * * *"
	}
    module_input "services"{
	    regexp = "^web.*|^api.*"
    }
}
`, taskName)
	sourceInputConsulKV := fmt.Sprintf(`task {
	name = "%s"
	services = ["api", "web"]
	module = "./test_modules/consul_kv_file"
	condition "schedule" {
      cron = "*/10 * * * * * *"
	}
    module_input "consul-kv" {
      path = "key-path"
      datacenter = "dc1"
    }
}
`, taskName)
	sourceInputConsulKVRecurse := fmt.Sprintf(`task {
	name = "%s"
	module = "./test_modules/consul_kv_file"
    services = ["api", "web"]
	condition "schedule" {
      cron = "*/10 * * * * * *"
	}
    module_input "consul-kv" {
      path = "key-path"
      datacenter = "dc1"
      recurse = true
    }
}
`, taskName)

	testcases := []struct {
		name          string
		conditionTask string
		tempDir       string
		isConsulKV    bool
		isRecurse     bool
	}{
		{
			name:          "with services",
			conditionTask: conditionWithServices,
			tempDir:       "schedule_basic_services",
			isConsulKV:    false,
			isRecurse:     false,
		},
		{
			name:          "with module_input services",
			conditionTask: conditionWithSourceInput,
			tempDir:       "schedule_basic_module_input",
			isConsulKV:    false,
			isRecurse:     false,
		},
		{
			name:          "with module_input consul_kv recurse false",
			conditionTask: sourceInputConsulKV,
			tempDir:       "schedule_consulKV",
			isConsulKV:    true,
			isRecurse:     false,
		},
		{
			name:          "with module_input consul_kvrecurse true",
			conditionTask: sourceInputConsulKVRecurse,
			tempDir:       "schedule_consulKV_recurse",
			isConsulKV:    true,
			isRecurse:     true,
		},
	}

	for _, tc := range testcases {
		tc := tc // rebind tc into this lexical scope for parallel use
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run table tests in parallel as they can take a lot of time
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

			// confirm service resources created
			resourcesPath := filepath.Join(tempDir, taskName, resourcesDir)
			validateServices(t, true, []string{"api-1", "web-1"}, resourcesPath)

			// Check KV events
			if tc.isConsulKV {
				// 3. Add KV. Confirm task only triggered on schedule
				// wait for next event before starting this process
				api.WaitForEvent(t, cts, taskName, time.Now(), scheduledWait)
				registerTime = time.Now()

				// add two keys and values, expected that the recursive key will only be
				// checked when recurse is enabled
				expectedKV := "red"
				expectedRecurseKV := "blue"
				srv.SetKVString(t, "key-path", expectedKV)
				srv.SetKVString(t, "key-path/recursive", expectedRecurseKV)

				// check scheduled task did not trigger immediately and ran only on schedule
				api.WaitForEvent(t, cts, taskName, registerTime, scheduledWait)
				checkScheduledRun(t, taskName, registerTime, taskSchedule, port)

				// confirm key-value resources created, and that the values are as expected
				validateModuleFile(t, true, true, resourcesPath, "key-path", expectedKV)

				if tc.isRecurse {
					validateModuleFile(t, true, true, resourcesPath, "key-path/recursive", expectedRecurseKV)
				} else {
					// if recurse is disabled, then the recursive key should not be present
					validateModuleFile(t, true, false, resourcesPath, "key-path/recursive", "")
				}
			}
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
	schedTaskName := "scheduled_task"
	taskSchedule := 10 * time.Second
	conditionTask := fmt.Sprintf(`task {
	name = "%s"
	services = ["api", "web"]
	module = "./test_modules/local_instances_file"
	condition "schedule" {
		cron = "*/10 * * * * * *"
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
