package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/event"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// tempDirPrefix is the prefix for the directory for a given e2e test
	// where files generated from e2e are stored. This directory is
	// destroyed after e2e testing if no errors.
	tempDirPrefix = "tmp_"

	// resourcesDir is the sub-directory of tempDir where the
	// Terraform resources created from running consul-terraform-sync are stored
	resourcesDir = "resources"

	// configFile is the name of the sync config file
	configFile = "config.hcl"

	// liberal default times to wait
	defaultWaitForRegistration = 8 * time.Second
	defaultWaitForEvent        = 8 * time.Second
	defaultWaitForAPI          = 30 * time.Second

	// liberal wait time to ensure event doesn't happen
	defaultWaitForNoEvent = 6 * time.Second
)

func newTestConsulServer(t *testing.T) *testutil.TestServer {
	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../testutils",
	})

	// Register services
	srv.AddAddressableService(t, "api", testutil.HealthPassing,
		"1.2.3.4", 8080, []string{"tag1"})
	srv.AddAddressableService(t, "web", testutil.HealthPassing,
		"5.6.7.8", 8000, []string{"tag2"})
	srv.AddAddressableService(t, "db", testutil.HealthPassing,
		"10.10.10.10", 8000, []string{"tag3", "tag4"})
	srv.AddAddressableService(t, "unhealthy-service", testutil.HealthCritical,
		"1.2.3.4", 8080, []string{})

	return srv
}

func runSyncStop(t *testing.T, configPath string, dur time.Duration) {
	cts, stop := api.StartCTS(t, configPath)
	cts.WaitForAPI(dur)
	stop(t)
}

// checkStateFileLocally checks if statefile exists
func checkStateFileLocally(t *testing.T, stateFilePath string) {
	files := testutils.CheckDir(t, true, stateFilePath)
	require.Equal(t, 1, len(files))

	stateFile := files[0]
	require.Equal(t, "terraform.tfstate", stateFile.Name())
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

// eventCount returns number of events that are stored for a given task by
// querying the Task Status API. Note: events have a storage limit (currently 5)
func eventCount(t *testing.T, taskName string, port int) int {
	events := events(t, taskName, port)
	return len(events)
}

// events returns the events that are stored for a given task by querying the
// Task Status API. Note: events have a storage limit (currently 5)
func events(t *testing.T, taskName string, port int) []event.Event {
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
	require.True(t, ok, taskStatuses)
	return taskStatus.Events
}
