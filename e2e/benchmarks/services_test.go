// +build e2e

// Runs benchmarks using the Null provider and working with Consul setup with
// multiple Services or Service Instances.

package benchmarks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/command"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
)

const configFile = "config.hcl"
const tempDirPrefix = "tmp_"

// Benchmark N Services Instance (all the same service)
func Benchmark1Instance(b *testing.B) {
	benchmarkInstances(b, 1)
}
func Benchmark100Instances(b *testing.B) {
	benchmarkInstances(b, 100)
}
func Benchmark1000Instances(b *testing.B) {
	benchmarkInstances(b, 1000)
}
func benchmarkInstances(b *testing.B, N int) {
	srv := testutils.NewTestConsulServer(b, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../../testutils",
	})
	defer srv.Stop()
	path, cleanup := setupServiceInstances(b, srv, N)
	defer cleanup() // comment out if you want to keep the config files
	//_ = cleanup

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		require.NoError(b, runSyncOnce(path))
	}
}

// Benchmark N Services
func Benchmark1Service(b *testing.B) {
	benchmarkServices(b, 1)
}
func Benchmark100Service(b *testing.B) {
	benchmarkServices(b, 100)
}
func Benchmark1000Service(b *testing.B) {
	benchmarkServices(b, 1000)
}
func benchmarkServices(b *testing.B, N int) {
	srv := testutils.NewTestConsulServer(b, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../../testutils",
	})
	defer srv.Stop()
	path, cleanup := setupMultiServices(b, srv, N)
	defer cleanup() // comment out if you want to keep the config files
	//_ = cleanup

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		require.NoError(b, runSyncOnce(path))
	}
}

// ----------------------------------------------------------------------------

// Run CTS in once mode from the cli level
func runSyncOnce(configPath string) error {
	cli := command.NewCLI(os.Stdout, os.Stderr)
	args := []string{"cts", "-once", fmt.Sprintf("--config-file=%s", configPath)}
	exitCode := cli.Run(args)
	errstr := "Exit Code Error: %s (%d)\n"
	switch exitCode {
	case command.ExitCodeOK:
		return nil
	case command.ExitCodeError:
		return fmt.Errorf(errstr, "ExitCodeError", exitCode)
	case command.ExitCodeInterrupt:
		return fmt.Errorf(errstr, "ExitCodeInterrupt", exitCode)
	case command.ExitCodeRequiredFlagsError:
		return fmt.Errorf(errstr, "ExitCodeRequiredFlagsError", exitCode)
	case command.ExitCodeParseFlagsError:
		return fmt.Errorf(errstr, "ExitCodeParseFlagsError", exitCode)
	case command.ExitCodeConfigError:
		return fmt.Errorf(errstr, "ExitCodeConfigError", exitCode)
	case command.ExitCodeDriverError:
		return fmt.Errorf(errstr, "ExitCodeDriverError", exitCode)
	default:
		return fmt.Errorf(errstr, "UNKNOWN", exitCode)
	}
}

// Sets up consul and config to test setup with N service instances
func setupServiceInstances(t testing.TB, srv *testutil.TestServer, N int) (string, func() error) {
	services := make([]string, 1)
	for _, s := range generateServices(N, true) {
		testutils.RegisterConsulService(t, srv, s, testutil.HealthPassing)
		services[0] = s.Name
	}
	return makeConfig(t, services, srv.HTTPAddr, false)
}

// add generated services as separte services
func setupMultiServices(t testing.TB, srv *testutil.TestServer, N int) (string, func() error) {
	services := make([]string, N)
	for i, s := range generateServices(N, false) {
		testutils.RegisterConsulService(t, srv, s, testutil.HealthPassing)
		services[i] = s.Name
	}
	return makeConfig(t, services, srv.HTTPSAddr, true)
}

// generate service instance entries
func generateServices(n int, instances bool) []testutil.TestService {
	baseport := 30000
	services := make([]testutil.TestService, n)
	for i := 0; i < n; i++ {
		name, id := "test_service", fmt.Sprintf("svc_%d", i)
		if !instances {
			name = id
		}
		services[i] = testutil.TestService{
			Name:    name,
			ID:      id,
			Address: "127.0.0.2",
			Port:    int(baseport + i),
			Tags:    []string{},
		}
	}
	return services
}

// writes the config files out to disk, returns path to them and cleanup func
func makeConfig(t testing.TB, services []string, addr string, tls bool,
) (string, func() error) {
	tmpDir := fmt.Sprintf("%s%s", tempDirPrefix, "test_services")
	cleanup := testutils.MakeTempDir(t, tmpDir)
	modDir := filepath.Join(tmpDir, "module")
	testutils.MakeTempDir(t, modDir)

	configPath := filepath.Join(tmpDir, configFile)
	err := writeConfig(configPath, configContents(addr, tmpDir, tls, services))
	require.NoError(t, err)
	modulePath := filepath.Join(modDir, "main.tf")
	err = writeConfig(modulePath, nullModuleContents())
	require.NoError(t, err)

	return configPath, cleanup
}

// write the (config) contents to the path
func writeConfig(configPath, contents string) error {
	f, err := os.Create(configPath)
	if err != nil {
		return nil
	}
	defer f.Close()
	config := []byte(contents)
	_, err = f.Write(config)
	return err
}

// returns templated contents of config file
func configContents(address, dir string, tls bool, services []string) string {
	sservices := `["` + strings.Join(services, `","`) + `"]`
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`
log_level = "WARN"

consul {
  address = "%s"
  tls {
	  enabled = %t
	  verify = false
  }
}

driver "terraform" {
	path = "%s"
	backend "local" {}
}

terraform_provider "local" {}

task {
  name = "test-services-task"
  source = "../../%s/module"
  providers = ["local"]
  services = %s
}

`, address, tls, pwd, dir, sservices)
}

// null resource, services module
func nullModuleContents() string {
	return `
resource "null_resource" "address" {}

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

// ---------------------------------------------------------------------------
// The below functions are not used but handy to keep around for when you need
// to check on what data Consul has.

// All registered services
func showMeServices(t testing.TB, srv *testutil.TestServer) {
	u := fmt.Sprintf("http://%s/v1/agent/services", srv.HTTPAddr)
	resp := testutils.RequestHTTP(t, http.MethodGet, u, "")
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	defer resp.Body.Close()
}

// Health status for all services
func showMeHealth(t testing.TB, srv *testutil.TestServer) {
	// get the node
	u := fmt.Sprintf("http://%s/v1/health/service/%s", srv.HTTPAddr, "svc_0")
	resp := testutils.RequestHTTP(t, http.MethodGet, u, "")
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	resp.Body.Close()
	nodes := []struct{ Node struct{ Node string } }{}
	json.Unmarshal(b, &nodes)
	node := nodes[0].Node.Node

	// all services on that node
	u = fmt.Sprintf("http://%s/v1/health/node/%s", srv.HTTPAddr, node)
	resp = testutils.RequestHTTP(t, http.MethodGet, u, "")
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	resp.Body.Close()

	fmt.Println(string(b))
}
