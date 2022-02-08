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
	"github.com/stretchr/testify/require"
)

type kvTaskOpts struct {
	path             string
	recurse          bool
	useAsModuleInput bool
}

// newKVTaskConfig returns a task configuration with a consul-kv condition
// given the provided options.
func newKVTaskConfig(taskName string, opts kvTaskOpts) string {
	var module string
	if opts.useAsModuleInput {
		module = "lornasong/cts_kv_file/local"
	} else {
		module = "./test_modules/local_instances_file"
	}

	conditionTask := fmt.Sprintf(`task {
		name = "%s"
		services = ["web", "api"]
		module = "%s"
		condition "consul-kv" {
			path = "%s"
			use_as_module_input = %t
			recurse = %t
		}
	}
	`, taskName, module, opts.path, opts.useAsModuleInput, opts.recurse)
	return conditionTask
}

// TestConditionConsulKV_NewKey runs the CTS binary using a task with a consul-kv
// condition block, where the KV pair for the configured path will not exist initially
// in Consul. The test will add a key with the configured path, add a key with
// an unrelated path, and add a key prefixed by the configured path. The expected
// behavior of a prefixed path key will depend on whether recurse is set or not.
func TestConditionConsulKV_NewKey(t *testing.T) {
	setParallelism(t)

	testcases := []struct {
		name             string
		recurse          bool
		useAsModuleInput bool
	}{
		{
			"single_key",
			false,
			false,
		},
		{
			"recurse",
			true,
			false,
		},
		{
			"single_key_use_as_module_input_true",
			false,
			true,
		},
		{
			"recurse_use_as_module_input_true",
			true,
			true,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			setParallelism(t)
			// Set up Consul server
			srv := newTestConsulServer(t)
			t.Cleanup(func() {
				_ = srv.Stop()
			})

			// Configure and start CTS
			taskName := "consul_kv_condition_new_" + tc.name
			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, taskName)
			path := "test-key/path"
			config := newKVTaskConfig(taskName, kvTaskOpts{
				path:             path,
				recurse:          tc.recurse,
				useAsModuleInput: tc.useAsModuleInput,
			})
			cts := ctsSetup(t, srv, tempDir, config)

			// Confirm only one event
			eventCountBase := eventCount(t, taskName, cts.Port())
			require.Equal(t, 1, eventCountBase)
			workingDir := fmt.Sprintf("%s/%s", tempDir, taskName)
			resourcesPath := filepath.Join(workingDir, resourcesDir)
			if tc.useAsModuleInput {
				// Confirm empty var consul_kv
				validateVariable(t, true, workingDir, "consul_kv", "{\n}")
			}

			// Add a key that is monitored by task, check for event
			now := time.Now()
			v := "test-value"
			srv.SetKVString(t, path, v)
			api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)
			eventCountNow := eventCount(t, taskName, cts.Port())
			eventCountBase++
			require.Equal(t, eventCountBase, eventCountNow,
				"event count did not increment once. task was not triggered as expected")
			validateModuleFile(t, tc.useAsModuleInput, true, resourcesPath, path, v)

			// Add a key that is not monitored, check for no event
			ignored := "not/related/path"
			srv.SetKVString(t, ignored, "test")
			time.Sleep(defaultWaitForNoEvent)
			eventCountNow = eventCount(t, taskName, cts.Port())
			require.Equal(t, eventCountBase, eventCountNow,
				"change in event count. task was unexpectedly triggered")
			validateModuleFile(t, tc.useAsModuleInput, false, resourcesPath, ignored, "")

			// Add a key prefixed by the existing path
			prefixed := fmt.Sprintf("%s/prefixed", path)
			pv := "prefixed-test-value"
			now = time.Now()
			srv.SetKVString(t, prefixed, pv)
			// Check for event if recurse is true, no event if recurse is false
			if tc.recurse {
				api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)
				eventCountNow := eventCount(t, taskName, cts.Port())
				eventCountBase++
				require.Equal(t, eventCountBase, eventCountNow,
					"event count did not increment once. task was not triggered as expected")
				validateModuleFile(t, tc.useAsModuleInput, true, resourcesPath, prefixed, pv)
			} else {
				time.Sleep(defaultWaitForNoEvent)
				eventCountNow = eventCount(t, taskName, cts.Port())
				require.Equal(t, eventCountBase, eventCountNow,
					"change in event count. task was unexpectedly triggered")
				validateModuleFile(t, tc.useAsModuleInput, false, resourcesPath, prefixed, "")
			}
		})
	}
}

// TestConditionConsulKV_ExistingKey runs the CTS binary using a task with a consul-kv
// condition block, where the monitored KV pair will exist initially in Consul. The
// test will update the value, delete the key, and then add the same key back.
func TestConditionConsulKV_ExistingKey(t *testing.T) {
	setParallelism(t)

	testcases := []struct {
		name             string
		recurse          bool
		useAsModuleInput bool
	}{
		{
			"single_key",
			false,
			false,
		},
		{
			"recurse",
			true,
			false,
		},
		{
			"single_key_use_as_module_input_true",
			false,
			true,
		},
		{
			"recurse_use_as_module_input_true",
			true,
			true,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			setParallelism(t)
			// Set up Consul server
			srv := newTestConsulServer(t)
			t.Cleanup(func() {
				_ = srv.Stop()
			})
			path := "test-key/path"
			value := "test-value"
			srv.SetKVString(t, path, value)

			childPath := path + "/test"
			childValue := "child value"
			srv.SetKVString(t, childPath, childValue)

			// Configure and start CTS
			taskName := "consul_kv_condition_existing_" + tc.name
			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, taskName)
			config := newKVTaskConfig(taskName, kvTaskOpts{
				path:             path,
				recurse:          tc.recurse,
				useAsModuleInput: tc.useAsModuleInput,
			})
			cts := ctsSetup(t, srv, tempDir, config)

			// Confirm only one event
			eventCountBase := eventCount(t, taskName, cts.Port())
			require.Equal(t, 1, eventCountBase)
			workingDir := fmt.Sprintf("%s/%s", tempDir, taskName)
			resourcesPath := filepath.Join(workingDir, resourcesDir)
			validateModuleFile(t, tc.useAsModuleInput, true, resourcesPath, path, value)

			// Confirm existence of child key depending on if recurse is true or not
			if tc.recurse {
				validateModuleFile(t, tc.useAsModuleInput, true, resourcesPath, childPath, childValue)
			} else {
				validateModuleFile(t, tc.useAsModuleInput, false, resourcesPath, childPath, "")
			}

			// Update key with new value, check for event
			now := time.Now()
			value = "new-test-value"
			srv.SetKVString(t, path, value)
			api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)
			eventCountNow := eventCount(t, taskName, cts.Port())
			eventCountBase++
			require.Equal(t, eventCountBase, eventCountNow,
				"event count did not increment once. task was not triggered as expected")
			validateModuleFile(t, tc.useAsModuleInput, true, resourcesPath, path, value)

			// Update child key with new value, check for event only if recurse
			now = time.Now()
			childValue = "new-child-value"
			srv.SetKVString(t, childPath, childValue)
			if tc.recurse {
				api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)
				eventCountNow := eventCount(t, taskName, cts.Port())
				eventCountBase++
				require.Equal(t, eventCountBase, eventCountNow,
					"event count did not increment once. task was not triggered as expected")
				validateModuleFile(t, tc.useAsModuleInput, true, resourcesPath, childPath, childValue)
			} else {
				time.Sleep(defaultWaitForNoEvent)
				eventCountNow = eventCount(t, taskName, cts.Port())
				require.Equal(t, eventCountBase, eventCountNow,
					"change in event count. task was unexpectedly triggered")
				validateModuleFile(t, tc.useAsModuleInput, false, resourcesPath, childPath, "")
			}

			// Delete key, check for event
			now = time.Now()
			testutils.DeleteKV(t, srv, path)
			api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)
			eventCountNow = eventCount(t, taskName, cts.Port())
			eventCountBase++
			require.Equal(t, eventCountBase, eventCountNow,
				"event count did not increment once. task was not triggered as expected")
			validateModuleFile(t, tc.useAsModuleInput, false, resourcesPath, path, "")

			// Add the key back, check for event
			now = time.Now()
			value = "new-test-value-2"
			srv.SetKVString(t, path, value)
			api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)
			eventCountNow = eventCount(t, taskName, cts.Port())
			eventCountBase++
			require.Equal(t, eventCountBase, eventCountNow,
				"event count did not increment once. task was not triggered as expected")
			validateModuleFile(t, tc.useAsModuleInput, true, resourcesPath, path, value)
		})
	}
}

// TestConditionConsulKV_SuppressTriggers runs the CTS binary using a task with a consul-kv
// condition block and tests that non-KV changes do not trigger the task.
func TestConditionConsulKV_SuppressTriggers(t *testing.T) {
	setParallelism(t)

	testcases := []struct {
		name             string
		recurse          bool
		useAsModuleInput bool
	}{
		{
			"single_key",
			false,
			false,
		},
		{
			"recurse",
			true,
			false,
		},
		{
			"single_key_use_as_module_input_true",
			false,
			true,
		},
		{
			"recurse_use_as_module_input_true",
			true,
			true,
		},
	}
	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			setParallelism(t)
			// Set up Consul server
			srv := newTestConsulServer(t)
			t.Cleanup(func() {
				_ = srv.Stop()
			})
			path := "test-key"
			value := "test-value"
			srv.SetKVString(t, path, value)

			// Configure and start CTS
			taskName := "consul_kv_condition_suppress_" + tc.name
			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, taskName)
			config := newKVTaskConfig(taskName, kvTaskOpts{
				path:             path,
				recurse:          tc.recurse,
				useAsModuleInput: tc.useAsModuleInput,
			})
			cts := ctsSetup(t, srv, tempDir, config)

			// Confirm one event at startup, check services files
			eventCountBase := eventCount(t, taskName, cts.Port())
			require.Equal(t, 1, eventCountBase)
			workingDir := fmt.Sprintf("%s/%s", tempDir, taskName)
			resourcesPath := filepath.Join(workingDir, resourcesDir)
			validateModuleFile(t, tc.useAsModuleInput, true, resourcesPath, path, value)
			validateServices(t, true, []string{"api", "web"}, resourcesPath)
			validateServices(t, false, []string{"db"}, resourcesPath)

			// Deregister a service, confirm no event and no update to service file
			testutils.DeregisterConsulService(t, srv, "web")
			time.Sleep(defaultWaitForNoEvent)
			eventCountNow := eventCount(t, taskName, cts.Port())
			require.Equal(t, eventCountBase, eventCountNow,
				"change in event count. task was unexpectedly triggered")
			validateServices(t, true, []string{"web"}, resourcesPath)

			// Update key, confirm event, confirm latest service information
			now := time.Now()
			value = "new-test-value"
			srv.SetKVString(t, path, value)
			api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)
			eventCountNow = eventCount(t, taskName, cts.Port())
			eventCountBase++
			require.Equal(t, eventCountBase, eventCountNow,
				"event count did not increment once. task was not triggered as expected")
			validateModuleFile(t, tc.useAsModuleInput, true, resourcesPath, path, value)
			validateServices(t, false, []string{"web"}, resourcesPath)
		})
	}
}

func TestConditionConsulKV_InvalidQueries(t *testing.T) {
	setParallelism(t)
	config := `task {
		name = "%s"
		module = "./test_modules/null_resource"
		module_input "services" {
			names  = ["api, web"]
		}
		condition "consul-kv" {
			path = "foo"
			%s
		}
	}`
	cases := []struct {
		name        string
		queryConfig string
		errMsg      string // client does not return detailed error message for Consul KV
	}{
		{
			"datacenter",
			`datacenter = "foo"`,
			"Unexpected response code: 500",
		},
		{
			"namespace_with_oss_consul",
			`namespace = "foo"`,
			"Unexpected response code: 400",
		},
	}

	for _, tc := range cases {
		tc := tc // rebind tc into this lexical scope for parallel use
		taskName := "condition_consul_kv_invalid_" + tc.name
		taskConfig := fmt.Sprintf(config, taskName, tc.queryConfig)
		testInvalidQueries(t, tc.name, taskName, taskConfig, tc.errMsg)
	}
}
