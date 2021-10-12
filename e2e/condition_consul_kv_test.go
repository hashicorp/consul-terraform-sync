// +build e2e

package e2e

import (
	"fmt"
	"os/exec"
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

type kvTaskOpts struct {
	path              string
	recurse           bool
	sourceIncludesVar bool
}

// newKVTaskConfig returns a task configuration with a consul-kv condition
// given the provided options.
func newKVTaskConfig(taskName string, opts kvTaskOpts) string {
	var module string
	if opts.sourceIncludesVar {
		module = "consul_kv_file"
	} else {
		module = "local_instances_file"
	}

	conditionTask := fmt.Sprintf(`task {
		name = "%s"
		services = ["web", "api"]
		source = "./test_modules/%s"
		condition "consul-kv" {
			path = "%s"
			source_includes_var = %t
			recurse = %t
		}
	}
	`, taskName, module, opts.path, opts.sourceIncludesVar, opts.recurse)
	return conditionTask
}

// ctsSetup executes the following setup steps:
// 1. Creates a temporary working directory,
// 2. Creates a CTS configuration file with the provided task
// 3. Starts CTS
// 4. Waits for the CTS API to start without error, indicating that all initialization is complete
func ctsSetup(t *testing.T, srv *testutil.TestServer, tempDir string, taskConfig string) *api.Client {
	cleanup := testutils.MakeTempDir(t, tempDir)
	t.Cleanup(func() {
		cleanup()
	})

	config := baseConfig(tempDir).appendConsulBlock(srv).appendTerraformBlock().
		appendString(taskConfig)
	configPath := filepath.Join(tempDir, configFile)
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath)
	t.Cleanup(func() {
		stop(t)
	})

	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)

	return cts
}

// TestConditionConsulKV_NewKey runs the CTS binary using a task with a consul-kv
// condition block, where the KV pair for the configured path will not exist initially
// in Consul. The test will add a key with the configured path, add a key with
// an unrelated path, and add a key prefixed by the configured path. The expected
// behavior of a prefixed path key will depend on whether recurse is set or not.
func TestConditionConsulKV_NewKey(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name    string
		recurse bool
	}{
		{
			"single key",
			false,
		},
		{
			"recurse",
			true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up Consul server
			srv := newTestConsulServer(t)
			t.Cleanup(func() {
				srv.Stop()
			})

			// Configure and start CTS
			taskName := "consul_kv_condition_new"
			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, taskName)
			path := "test-key/path"
			config := newKVTaskConfig(taskName, kvTaskOpts{
				path:              path,
				recurse:           tc.recurse,
				sourceIncludesVar: true,
			})
			cts := ctsSetup(t, srv, tempDir, config)

			// Confirm only one event. Confirm empty var consul_kv
			eventCountBase := eventCount(t, taskName, cts.Port())
			require.Equal(t, 1, eventCountBase)
			workingDir := fmt.Sprintf("%s/%s", tempDir, taskName)
			resourcesPath := filepath.Join(workingDir, resourcesDir)
			content := testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
			assert.Contains(t, content, "consul_kv = {\n}")

			// Add a key that is monitored by task, check for event
			now := time.Now()
			v := "test-value"
			srv.SetKVString(t, path, v)
			api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)
			eventCountNow := eventCount(t, taskName, cts.Port())
			eventCountBase++
			require.Equal(t, eventCountBase, eventCountNow,
				"event count did not increment once. task was not triggered as expected")
			content = testutils.CheckFile(t, true, resourcesPath, fmt.Sprintf("%s.txt", path))
			assert.Equal(t, v, content)

			// Add a key that is not monitored, check for no event
			ignored := "not/related/path"
			srv.SetKVString(t, ignored, "test")
			time.Sleep(defaultWaitForNoEvent)
			eventCountNow = eventCount(t, taskName, cts.Port())
			require.Equal(t, eventCountBase, eventCountNow,
				"change in event count. task was unexpectedly triggered")
			testutils.CheckFile(t, false, resourcesPath, fmt.Sprintf("%s.txt", ignored))

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
				content = testutils.CheckFile(t, true, resourcesPath, fmt.Sprintf("%s.txt", prefixed))
				assert.Equal(t, pv, content)
			} else {
				time.Sleep(defaultWaitForNoEvent)
				eventCountNow = eventCount(t, taskName, cts.Port())
				require.Equal(t, eventCountBase, eventCountNow,
					"change in event count. task was unexpectedly triggered")
				testutils.CheckFile(t, false, resourcesPath, fmt.Sprintf("%s.txt", prefixed))
			}
		})
	}
}

// TestConditionConsulKV_ExistingKey runs the CTS binary using a task with a consul-kv
// condition block, where the monitored KV pair will exist initially in Consul. The
// test will update the value, delete the key, and then add the same key back.
func TestConditionConsulKV_ExistingKey(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name    string
		recurse bool
	}{
		{
			"single key",
			false,
		},
		{
			"recurse",
			true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up Consul server
			srv := newTestConsulServer(t)
			t.Cleanup(func() {
				srv.Stop()
			})
			path := "test-key/path"
			value := "test-value"
			srv.SetKVString(t, path, value)

			childPath := path + "/test"
			childValue := "child value"
			srv.SetKVString(t, childPath, childValue)

			// Configure and start CTS
			taskName := "consul_kv_condition_existing"
			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, taskName)
			config := newKVTaskConfig(taskName, kvTaskOpts{
				path:              path,
				recurse:           tc.recurse,
				sourceIncludesVar: true,
			})
			cts := ctsSetup(t, srv, tempDir, config)

			// Confirm only one event
			eventCountBase := eventCount(t, taskName, cts.Port())
			require.Equal(t, 1, eventCountBase)
			workingDir := fmt.Sprintf("%s/%s", tempDir, taskName)
			resourcesPath := filepath.Join(workingDir, resourcesDir)
			content := testutils.CheckFile(t, true, resourcesPath, fmt.Sprintf("%s.txt", path))
			assert.Equal(t, value, content)

			// Confirm existence of child key depending on if recurse is true or not
			if tc.recurse {
				content := testutils.CheckFile(t, true, resourcesPath, fmt.Sprintf("%s.txt", childPath))
				assert.Equal(t, childValue, content)
			} else {
				testutils.CheckFile(t, false, resourcesPath, fmt.Sprintf("%s.txt", childPath))
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
			content = testutils.CheckFile(t, true, resourcesPath, fmt.Sprintf("%s.txt", path))
			assert.Equal(t, value, content)

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
				content := testutils.CheckFile(t, true, resourcesPath, fmt.Sprintf("%s.txt", childPath))
				assert.Equal(t, childValue, content)
			} else {
				time.Sleep(defaultWaitForNoEvent)
				eventCountNow = eventCount(t, taskName, cts.Port())
				require.Equal(t, eventCountBase, eventCountNow,
					"change in event count. task was unexpectedly triggered")
				testutils.CheckFile(t, false, resourcesPath, fmt.Sprintf("%s.txt", childPath))
			}

			// Delete key, check for event
			now = time.Now()
			testutils.DeleteKV(t, srv, path)
			api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)
			eventCountNow = eventCount(t, taskName, cts.Port())
			eventCountBase++
			require.Equal(t, eventCountBase, eventCountNow,
				"event count did not increment once. task was not triggered as expected")
			content = testutils.CheckFile(t, false, resourcesPath, fmt.Sprintf("%s.txt", path))

			// Add the key back, check for event
			now = time.Now()
			value = "new-test-value-2"
			srv.SetKVString(t, path, value)
			api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)
			eventCountNow = eventCount(t, taskName, cts.Port())
			eventCountBase++
			require.Equal(t, eventCountBase, eventCountNow,
				"event count did not increment once. task was not triggered as expected")
			content = testutils.CheckFile(t, true, resourcesPath, fmt.Sprintf("%s.txt", path))
			assert.Equal(t, value, content)
		})
	}
}

// TestConditionConsulKV_SuppressTriggers runs the CTS binary using a task with a consul-kv
// condition block and tests that non-KV changes do not trigger the task.
func TestConditionConsulKV_SuppressTriggers(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name    string
		recurse bool
	}{
		{
			"single key",
			false,
		},
		{
			"recurse",
			true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up Consul server
			srv := newTestConsulServer(t)
			t.Cleanup(func() {
				srv.Stop()
			})
			path := "test-key"
			pathFile := fmt.Sprintf("%s.txt", path)
			value := "test-value"
			srv.SetKVString(t, path, value)

			// Configure and start CTS
			taskName := "consul_kv_condition_suppress_triggers"
			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, taskName)
			config := newKVTaskConfig(taskName, kvTaskOpts{
				path:              path,
				recurse:           tc.recurse,
				sourceIncludesVar: true,
			})
			cts := ctsSetup(t, srv, tempDir, config)

			// Confirm one event at startup, check services files
			eventCountBase := eventCount(t, taskName, cts.Port())
			require.Equal(t, 1, eventCountBase)
			workingDir := fmt.Sprintf("%s/%s", tempDir, taskName)
			resourcesPath := filepath.Join(workingDir, resourcesDir)
			testutils.CheckFile(t, true, resourcesPath, pathFile)
			testutils.CheckFile(t, true, resourcesPath, "api.txt")
			testutils.CheckFile(t, true, resourcesPath, "web.txt")
			testutils.CheckFile(t, false, resourcesPath, "db.txt")

			// Deregister a service, confirm no event and no update to service file
			testutils.DeregisterConsulService(t, srv, "web")
			time.Sleep(defaultWaitForNoEvent)
			eventCountNow := eventCount(t, taskName, cts.Port())
			require.Equal(t, eventCountBase, eventCountNow,
				"change in event count. task was unexpectedly triggered")
			testutils.CheckFile(t, true, resourcesPath, "web.txt")

			// Update key, confirm event, confirm latest service information
			now := time.Now()
			value = "new-test-value"
			srv.SetKVString(t, path, value)
			api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)
			eventCountNow = eventCount(t, taskName, cts.Port())
			eventCountBase++
			require.Equal(t, eventCountBase, eventCountNow,
				"event count did not increment once. task was not triggered as expected")
			content := testutils.CheckFile(t, true, resourcesPath, pathFile)
			assert.Equal(t, value, content)
			testutils.CheckFile(t, false, resourcesPath, "web.txt")
		})
	}
}

// TestConditionConsul_namespace_oss tests conditions with configured namespace
// meanwhile connecting with Consul OSS.
func TestConditionConsul_namespace_oss(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	testCases := []struct {
		name string
		task string
	}{
		{
			name: "catalog-services",
			task: `
task {
  name = "catalog-services"
  source = "./test_modules/null_resource"
  condition "catalog-services" {
    regexp = ".*"
    namespace = "dne"
  }
}`,
		}, {
			name: "consul-kv",
			task: `
task {
  name = "consul-kv"
  source = "./test_modules/null_resource"
  services = ["foobar"]
  condition "consul-kv" {
    path = "foo"
    namespace = "dne"
  }
}`,
		},
	}

	port := testutils.FreePort(t)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := fmt.Sprintf("%snamespace_oss_%s", tempDirPrefix, tc.name)
			cleanup := testutils.MakeTempDir(t, tempDir)
			defer cleanup()

			config := baseConfig(tempDir).appendConsulBlock(srv).appendString(tc.task).
				appendString(fmt.Sprintf("port = %d", port))
			configPath := filepath.Join(tempDir, configFile)
			config.write(t, configPath)

			errCh := make(chan error, 1)
			cmd := exec.Command("consul-terraform-sync", "--once", fmt.Sprintf("--config-file=%s", configPath))
			go func() {
				errCh <- cmd.Run()
			}()

			timeout := time.After(defaultWaitForAPI)
			select {
			case err := <-errCh:
				assert.Error(t, err, "namespace query should error and cause once-mode to not stop successfully")
			case <-timeout:
				t.Fatalf("expected CTS to error during once mode with first 400 API response from Consul")
			}
		})
	}
}
