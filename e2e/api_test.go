// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_StatusEndpoints(t *testing.T) {
	t.Parallel()

	srv, err := newTestConsulServer(t)
	require.NoError(t, err, "failed to start consul server")
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "status_endpoints")
	err = makeTempDir(tempDir)
	// no defer to delete directory: only delete at end of test if no errors
	require.NoError(t, err)

	configPath := filepath.Join(tempDir, configFile)

	port, err := api.FreePort()
	require.NoError(t, err)

	err = makeConfig(configPath, fakeHandlerConfig(srv.HTTPAddr, tempDir, port))
	require.NoError(t, err)

	cmd, err := runSyncDevMode(configPath)
	require.NoError(t, err)

	// wait to run once before registering another instance to collect another event
	time.Sleep(7 * time.Second)
	service := testutil.TestService{
		ID:      "api-2",
		Name:    "api",
		Address: "5.6.7.8",
		Port:    8080,
	}
	registerService(t, srv, service, testutil.HealthPassing)

	// wait and then retrieve status
	time.Sleep(5 * time.Second)

	taskCases := []struct {
		name     string
		path     string
		expected map[string]api.TaskStatus
	}{
		{
			"all task statuses",
			"status/tasks",
			map[string]api.TaskStatus{
				fakeSuccessTaskName: api.TaskStatus{
					TaskName:  fakeSuccessTaskName,
					Status:    api.StatusHealthy,
					Providers: []string{"fake-sync"},
					Services:  []string{"api"},
					EventsURL: "/v1/status/tasks/fake_handler_success_task?include=events",
				},
				fakeFailureTaskName: api.TaskStatus{
					TaskName:  fakeFailureTaskName,
					Status:    api.StatusDegraded,
					Providers: []string{"fake-sync"},
					Services:  []string{"api"},
					EventsURL: "/v1/status/tasks/fake_handler_failure_task?include=events",
				},
			},
		},
		{
			"existing single task status",
			"status/tasks/" + fakeSuccessTaskName,
			map[string]api.TaskStatus{
				fakeSuccessTaskName: api.TaskStatus{
					TaskName:  fakeSuccessTaskName,
					Status:    api.StatusHealthy,
					Providers: []string{"fake-sync"},
					Services:  []string{"api"},
					EventsURL: "/v1/status/tasks/fake_handler_success_task?include=events",
				},
			},
		},
		{
			"non-existing single task status",
			"status/tasks/" + "non-existing-task",
			map[string]api.TaskStatus{
				"non-existing-task": api.TaskStatus{
					TaskName:  "non-existing-task",
					Status:    api.StatusUndetermined,
					Providers: []string{},
					Services:  []string{},
					EventsURL: "",
				},
			},
		},
	}

	for _, tc := range taskCases {
		t.Run(tc.name, func(t *testing.T) {
			u := fmt.Sprintf("http://localhost:%d/%s/%s", port, "v1", tc.path)
			resp, err := http.Get(u)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, resp.StatusCode, http.StatusOK)

			var taskStatuses map[string]api.TaskStatus
			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&taskStatuses)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, taskStatuses)
		})
	}

	overallCases := []struct {
		name     string
		path     string
		expected api.OverallStatus
	}{
		{
			"overall status",
			"status",
			api.OverallStatus{
				Status: api.StatusDegraded,
			},
		},
	}

	for _, tc := range overallCases {
		t.Run(tc.name, func(t *testing.T) {
			u := fmt.Sprintf("http://localhost:%d/%s/%s", port, "v1", tc.path)
			resp, err := http.Get(u)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, resp.StatusCode, http.StatusOK)

			var overallStatus api.OverallStatus
			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&overallStatus)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, overallStatus)
		})
	}

	err = stopCommand(cmd)
	require.NoError(t, err)
	removeDir(tempDir)
}

// runSyncDevMode runs the daemon in development which does not run or download
// Terraform.
func runSyncDevMode(configPath string) (*exec.Cmd, error) {
	cmd := exec.Command("consul-terraform-sync",
		fmt.Sprintf("--config-file=%s", configPath), "--client-type=development")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func stopCommand(cmd *exec.Cmd) error {
	cmd.Process.Signal(os.Interrupt)
	sigintErr := errors.New("signal: interrupt")
	if err := cmd.Wait(); err != nil && err != sigintErr {
		return err
	}
	return nil
}

// registerService is a helper function to regsiter a service to the Consul
// Catalog. The Consul sdk/testutil package currently does not support a method
// to register multiple service instances, distinguished by their IDs.
func registerService(t *testing.T, srv *testutil.TestServer, s testutil.TestService, health string) {
	var body bytes.Buffer
	enc := json.NewEncoder(&body)
	require.NoError(t, enc.Encode(&s))

	u := fmt.Sprintf("http://%s/v1/agent/service/register", srv.HTTPAddr)
	req, err := http.NewRequest("PUT", u, io.Reader(&body))
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	srv.AddCheck(t, s.ID, s.ID, testutil.HealthPassing)
}
