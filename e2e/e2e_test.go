// +build e2e

package e2e

import (
	"errors"
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

// configFile is the name of the nia config file
const configFile = "config.hcl"

func TestE2EBasic(t *testing.T) {
	// Note: no t.Parallel() for this particular test. Choosing this test to run 'first'
	// since e2e test running simultaneously will download Terraform into shared
	// directory causes some flakiness. All other e2e tests, should have t.Parallel()

	srv, err := newTestConsulServer(t)
	require.NoError(t, err, "failed to start consul server")
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "basic")
	err = makeTempDir(tempDir)
	// no defer to delete directory: only delete at end of test if no errors
	require.NoError(t, err)

	configPath := filepath.Join(tempDir, configFile)
	err = makeConfig(configPath, twoTaskConfig(srv.HTTPAddr, tempDir))
	require.NoError(t, err)

	err = runConsulNIA(configPath, 20*time.Second)
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

	removeDir(tempDir)
}

func TestE2ERestartConsulNIA(t *testing.T) {
	t.Parallel()

	srv, err := newTestConsulServer(t)
	require.NoError(t, err, "failed to start consul server")
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "restart")
	err = makeTempDir(tempDir)
	// no defer to delete directory: only delete at end of test if no errors
	require.NoError(t, err)

	configPath := filepath.Join(tempDir, configFile)
	err = makeConfig(configPath, oneTaskConfig(srv.HTTPAddr, tempDir))
	require.NoError(t, err)

	err = runConsulNIA(configPath, 8*time.Second)
	require.NoError(t, err)

	// rerun nia. confirm no errors e.g. recreating workspaces
	err = runConsulNIA(configPath, 8*time.Second)
	require.NoError(t, err)

	removeDir(tempDir)
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

func makeTempDir(tempDir string) error {
	_, err := os.Stat(tempDir)
	if !os.IsNotExist(err) {
		log.Printf("[WARN] temp dir %s was not cleared out after last test. Deleting.", tempDir)
		if err = removeDir(tempDir); err != nil {
			return err
		}
	}
	return os.Mkdir(tempDir, os.ModePerm)
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

func runConsulNIA(configPath string, dur time.Duration) error {
	cmd := exec.Command("consul-terraform-sync", fmt.Sprintf("--config-file=%s", configPath))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	time.Sleep(dur)
	cmd.Process.Signal(os.Interrupt)
	sigintErr := errors.New("signal: interrupt")
	if err := cmd.Wait(); err != nil && err != sigintErr {
		return err
	}
	return nil
}

// removeDir removes temporary directory created for a test
func removeDir(tempDir string) error {
	return os.RemoveAll(tempDir)
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
