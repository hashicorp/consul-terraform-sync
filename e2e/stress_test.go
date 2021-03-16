// build stress

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

	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
)

const configFile = "config.hcl"
const tempDirPrefix = "tmp_"

func TestNullProvider(t *testing.T) {
	srv := newConsulServer(t)
	defer srv.Stop()
	//addServices(t, srv)
	addServiceInstances(t, srv)

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "stress")
	delete := testutils.MakeTempDir(t, tempDir)
	modDir := filepath.Join(tempDir, "module")
	testutils.MakeTempDir(t, modDir)
	_ = delete
	//defer delete()

	configPath := filepath.Join(tempDir, configFile)
	err := makeConfig(configPath, configContents(srv.HTTPAddr))
	require.NoError(t, err)
	modulePath := filepath.Join(modDir, "main.tf")
	err = makeConfig(modulePath, moduleContents())
	require.NoError(t, err)

	err = runSyncOnce(configPath)
	require.NoError(t, err)
}

// ----------------------------------------------------------------------------

// Run CTS in once mode
func runSyncOnce(configPath string) error {
	cmd := exec.Command("consul-terraform-sync", "-once", fmt.Sprintf("--config-file=%s", configPath))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// generate service instance entries
func generateServices(n int) []testutil.TestService {
	baseport := 30000
	services := make([]testutil.TestService, n)
	for i := 0; i < n; i++ {
		services[i] = testutil.TestService{
			Name:    "stress",
			ID:      fmt.Sprintf("svc_%d", i),
			Address: "127.0.0.2",
			Port:    int(baseport + i),
			Tags:    []string{},
		}
	}
	return services
}

// add generated services as service instances
func addServiceInstances(t *testing.T, srv *testutil.TestServer) {
	for _, s := range generateServices(2000) {
		testutils.RegisterConsulService(t, srv, s, testutil.HealthPassing)
	}
}

// add generated services as separte services
func addServices(t *testing.T, srv *testutil.TestServer) {
	for _, s := range generateServices(2000) {
		srv.AddAddressableService(t,
			s.ID, testutil.HealthPassing, s.Address, s.Port, s.Tags)
	}
}

// so I can see if added services look good
func showMeServices(t *testing.T, srv *testutil.TestServer) {
	u := fmt.Sprintf("http://%s/v1/agent/services", srv.HTTPAddr)
	resp := testutils.RequestHTTP(t, http.MethodGet, u, "")
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	defer resp.Body.Close()
}

// mostly c-n-p without the services being added
func newConsulServer(t *testing.T) *testutil.TestServer {
	log.SetOutput(ioutil.Discard)
	srv, err := testutil.NewTestServerConfigT(t,
		func(c *testutil.TestServerConfig) {
			c.LogLevel = "warn"
			c.Stdout = ioutil.Discard
			c.Stderr = ioutil.Discard
		})
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

func configContents(address string) string {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`
log_level = "WARN"

consul {
  address = "%s"
}

driver "terraform" {
	path = "%s"
	backend "local" {}
}

terraform_provider "local" {}

task {
  name = "stress-test"
  source = "../../tmp_stress/module"
  providers = ["local"]
  services = ["stress"]
}

`, address, pwd)
}

func moduleContents() string {
	return `
resource "local_file" "address" {
    for_each = var.services
    content = each.value.address
    filename = "../../tmp_stress/out/service_${each.value.id}.txt"
}

variable "services" {
  description = "Consul services monitored by Consul Terraform Sync"
  type = map(
    object({
      id        = string
      name      = string
      kind      = string
      address   = string
      port      = number
      meta      = map(string)
      tags      = list(string)
      namespace = string
      status    = string

      node                  = string
      node_id               = string
      node_address          = string
      node_datacenter       = string
      node_tagged_addresses = map(string)
      node_meta             = map(string)

      cts_user_defined_meta = map(string)
    })
  )
}
`
}

// ----------------------------------------------------------------------------
// c-n-p .. tmp, here in case I don't leave this in e2e module
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
