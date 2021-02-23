// +build e2e

package e2e

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
)

// tempDirPrefix is the prefix for the directory for a given e2e test
// where files generated from e2e are stored. This directory is
// destroyed after e2e testing if no errors.
const tempDirPrefix = "tmp_"

// resourcesDir is the sub-directory of tempDir where the
// Terraform resources created from running consul-terraform-sync are stored
const resourcesDir = "resources"

// configFile is the name of the sync config file
const configFile = "config.hcl"

func TestE2EBasic(t *testing.T) {
	// Note: no t.Parallel() for this particular test. Choosing this test to run 'first'
	// since e2e test running simultaneously will download Terraform into shared
	// directory causes some flakiness. All other e2e tests, should have t.Parallel()

	srv, err := newTestConsulServer(t)
	require.NoError(t, err, "failed to start consul server")
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "basic")
	delete, err := testutils.MakeTempDir(tempDir)
	// no defer to delete directory: only delete at end of test if no errors
	require.NoError(t, err)

	configPath := filepath.Join(tempDir, configFile)
	err = makeConfig(configPath, twoTaskConfig(srv.HTTPAddr, tempDir))
	require.NoError(t, err)

	err = runSyncStop(configPath, 20*time.Second)
	require.NoError(t, err)

	files, err := ioutil.ReadDir(fmt.Sprintf("%s/%s", tempDir, resourcesDir))
	require.NoError(t, err)
	require.Equal(t, 3, len(files))

	contents, err := ioutil.ReadFile(fmt.Sprintf("%s/%s/consul_service_api.txt", tempDir, resourcesDir))
	require.NoError(t, err)
	require.Equal(t, "1.2.3.4", string(contents))

	contents, err = ioutil.ReadFile(fmt.Sprintf("%s/%s/consul_service_web.txt", tempDir, resourcesDir))
	require.NoError(t, err)
	require.Equal(t, "5.6.7.8", string(contents))

	contents, err = ioutil.ReadFile(fmt.Sprintf("%s/%s/consul_service_db.txt", tempDir, resourcesDir))
	require.NoError(t, err)
	require.Equal(t, "10.10.10.10", string(contents))

	// check statefiles exist
	status, err := checkStateFile(srv.HTTPAddr, dbTaskName)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)

	status, err = checkStateFile(srv.HTTPAddr, webTaskName)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)

	delete()
}

func TestE2ERestartSync(t *testing.T) {
	t.Parallel()

	srv, err := newTestConsulServer(t)
	require.NoError(t, err, "failed to start consul server")
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "restart")
	delete, err := testutils.MakeTempDir(tempDir)
	// no defer to delete directory: only delete at end of test if no errors
	require.NoError(t, err)

	configPath := filepath.Join(tempDir, configFile)
	err = makeConfig(configPath, oneTaskConfig(srv.HTTPAddr, tempDir, 0))
	require.NoError(t, err)

	err = runSyncStop(configPath, 8*time.Second)
	require.NoError(t, err)

	// rerun sync. confirm no errors e.g. recreating workspaces
	err = runSyncStop(configPath, 8*time.Second)
	require.NoError(t, err)

	delete()
}

func TestE2EPanosHandlerError(t *testing.T) {
	t.Parallel()

	srv, err := newTestConsulServer(t)
	require.NoError(t, err, "failed to start consul server")
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "panos_handler")
	delete, err := testutils.MakeTempDir(tempDir)
	// no defer to delete directory: only delete at end of test if no errors
	require.NoError(t, err)

	configPath := filepath.Join(tempDir, configFile)
	err = makeConfig(configPath, panosConfig(srv.HTTPAddr, tempDir))
	require.NoError(t, err)

	err = runSyncOnce(configPath)
	require.Error(t, err)

	delete()
}

func TestE2ELocalBackend(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		tempDirPrefix    string
		backendConfig    string
		dbStateFilePath  string
		webStateFilePath string
	}{
		{
			"no parameters configured",
			"local_backend_default",
			`backend "local" {}`,
			fmt.Sprintf("tmp_local_backend_default/%s/terraform.tfstate.d/%s",
				dbTaskName, dbTaskName),
			fmt.Sprintf("tmp_local_backend_default/%s/terraform.tfstate.d/%s",
				webTaskName, webTaskName),
		},
		{
			"workspace_dir configured",
			"local_backend_ws_dir",
			`backend "local" {
				workspace_dir = "test-workspace"
			}`,
			fmt.Sprintf("tmp_local_backend_ws_dir/%s/test-workspace/%s",
				dbTaskName, dbTaskName),
			fmt.Sprintf("tmp_local_backend_ws_dir/%s/test-workspace/%s",
				webTaskName, webTaskName),
		},
		{
			"workspace_dir configured with tasks sharing a workspace dir",
			"local_backend_shared_ws_dir",
			`backend "local" {
				workspace_dir = "../shared-workspace"
			}`,
			fmt.Sprintf("tmp_local_backend_shared_ws_dir/shared-workspace/%s",
				dbTaskName),
			fmt.Sprintf("tmp_local_backend_shared_ws_dir/shared-workspace/%s",
				webTaskName),
		},
		{
			"path configured",
			"local_backend_path",
			`backend "local" {
				# Setting path is meaningless in Sync. TF only uses it for
				# default workspace; Sync only uses non-default workspaces. This
				# value is overridden by the workspace directory for non-default
				# workspaces.
				path = "this-will-be-replaced-by-default-dir/terraform.tfstate"
			}`,
			fmt.Sprintf("tmp_local_backend_path/%s/terraform.tfstate.d/%s",
				dbTaskName, dbTaskName),
			fmt.Sprintf("tmp_local_backend_path/%s/terraform.tfstate.d/%s",
				webTaskName, webTaskName),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, err := newTestConsulServer(t)
			require.NoError(t, err, "failed to start consul server")
			defer srv.Stop()

			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, tc.tempDirPrefix)
			delete, err := testutils.MakeTempDir(tempDir)
			// no defer to delete directory: only delete at end of test if no errors
			require.NoError(t, err)

			configPath := filepath.Join(tempDir, configFile)
			err = makeConfig(configPath,
				twoTaskCustomBackendConfig(srv.HTTPAddr, tempDir, tc.backendConfig))
			require.NoError(t, err)

			err = runSyncOnce(configPath)
			require.NoError(t, err)

			// check that statefile was created locally
			exists, err := checkStateFileLocally(tc.dbStateFilePath)
			require.NoError(t, err)
			require.True(t, exists)

			exists, err = checkStateFileLocally(tc.webStateFilePath)
			require.NoError(t, err)
			require.True(t, exists)

			delete()
		})
	}
}

func newTestConsulServer(t *testing.T) (*testutil.TestServer, error) {
	log.SetOutput(ioutil.Discard)
	srv, err := testutil.NewTestServerConfig(func(c *testutil.TestServerConfig) {
		c.LogLevel = "warn"
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	if err != nil {
		return nil, err
	}

	// Register services
	srv.AddAddressableService(t, "api", testutil.HealthPassing,
		"1.2.3.4", 8080, []string{})
	srv.AddAddressableService(t, "web", testutil.HealthPassing,
		"5.6.7.8", 8000, []string{})
	srv.AddAddressableService(t, "db", testutil.HealthPassing,
		"10.10.10.10", 8000, []string{})
	return srv, nil
}

func makeConfig(configPath, contents string) error {
	f, err := os.Create(configPath)
	if err != nil {
		return nil
	}
	defer f.Close()
	config := []byte(contents)
	_, err = f.Write(config)
	return err
}

func runSyncStop(configPath string, dur time.Duration) error {
	cmd, err := runSync(configPath)
	if err != nil {
		return err
	}
	time.Sleep(dur)
	return stopCommand(cmd)
}

func runSyncOnce(configPath string) error {
	cmd := exec.Command("consul-terraform-sync", "-once", fmt.Sprintf("--config-file=%s", configPath))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func checkStateFile(consulAddr, taskname string) (int, error) {
	u := fmt.Sprintf("http://%s/v1/kv/%s-env:%s", consulAddr, config.DefaultTFBackendKVPath, taskname)

	resp, err := http.Get(u)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

// checkStateFileLocally returns whether or not a statefile exists
func checkStateFileLocally(stateFilePath string) (bool, error) {
	files, err := ioutil.ReadDir(stateFilePath)
	if err != nil {
		return false, err
	}

	if len(files) != 1 {
		return false, nil
	}

	stateFile := files[0]
	if stateFile.Name() != "terraform.tfstate" {
		return false, nil
	}

	return true, nil
}
