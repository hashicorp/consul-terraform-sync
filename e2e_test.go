// +build e2e

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
)

// tempDir is the overall directory where files generated from
// e2e are stored. This directory is destroyed after e2e testing
// if no errors.
const tempDir = "e2e_tmp"

// resourcesDir is the sub-directory of tempDir where the
// Terraform resources created from running consul-nia are stored
const resourcesDir = "resources"

// configFile is the name of the nia config file
const configFile = "config.hcl"

func TestE2E(t *testing.T) {
	// set up Consul
	log.SetOutput(ioutil.Discard)
	srv, err := testutil.NewTestServerConfig(func(c *testutil.TestServerConfig) {
		c.LogLevel = "warn"
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	require.NoError(t, err, "failed to start consul server")
	defer srv.Stop()

	// Register services
	srv.AddAddressableService(t, "api", testutil.HealthPassing,
		"1.2.3.4", 8080, []string{})
	srv.AddAddressableService(t, "web", testutil.HealthPassing,
		"5.6.7.8", 8000, []string{})

	// set up temporary directory
	_, err = os.Stat(tempDir)
	if !os.IsNotExist(err) {
		log.Println("[WARN] temp dir was not cleared out after last test. Deleting.")
		err = os.RemoveAll(tempDir)
		require.NoError(t, err)
	}
	err = os.Mkdir(tempDir, os.ModePerm)
	// no defer to delete directory: only delete at end of test if no errors
	require.NoError(t, err)

	// create config file
	configPath := filepath.Join(tempDir, configFile)
	f, err := os.Create(configPath)
	require.NoError(t, err)
	defer f.Close()
	config := []byte(basicConfigFile(srv.HTTPAddr))
	_, err = f.Write(config)
	require.NoError(t, err)

	// call nia. set output to stdout
	cmd := exec.Command("sudo", "consul-nia", fmt.Sprintf("--config-file=%s", configPath))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	require.NoError(t, err)

	contents, err := ioutil.ReadFile(fmt.Sprintf("%s/%s/consul_service_api.0.txt", tempDir, resourcesDir))
	require.NoError(t, err)
	require.Equal(t, "1.2.3.4", string(contents))

	contents, err = ioutil.ReadFile(fmt.Sprintf("%s/%s/consul_service_web.0.txt", tempDir, resourcesDir))
	require.NoError(t, err)
	require.Equal(t, "5.6.7.8", string(contents))

	os.RemoveAll(tempDir)
}

// basicConfigFile returns a config file with the dynamically generated
// consul address
func basicConfigFile(consulAddr string) string {

	consulBlock := fmt.Sprintf(`
consul {
	address = "%s"
}`, consulAddr)

	terraformBlock := fmt.Sprintf(`
driver "terraform" {
	skip_verify = true
	path = "/usr/local/bin/"
	data_dir = "%s"
	working_dir = "%s"
}
`, tempDir, tempDir)

	return consulBlock + terraformBlock + `
log_level = "trace"

service {
  name = "api"
  description = "backend"
}

service {
  name = "web"
  description = "frontend"
}

provider "local" {}

task {
  name = "e2e_basic_task"
  description = "basic read-write e2e task"
  services = ["api", "web"]
  providers = ["local"]
  source = "../../test_modules/e2e_basic_task"
}`
}
