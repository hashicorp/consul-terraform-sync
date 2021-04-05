// +build e2e

// Runs benchmarks using the Null provider and working with Consul setup with
// multiple Services or Service Instances.

package benchmarks

import (
	"fmt"
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

// Sets up consul and config with N service instances of the same service
func setupServiceInstances(t testing.TB, srv *testutil.TestServer, N int) (string, func() error) {
	testInstances := testutils.TestInstances(N)
	testutils.AddServices(t, srv, testInstances)
	services := []string{testInstances[0].Name} // shared name
	return makeConfig(t, services, srv.HTTPAddr, false)
}

// Sets up consul and config with N unique services
func setupMultiServices(t testing.TB, srv *testutil.TestServer, N int) (string, func() error) {
	testServices := testutils.TestServices(N)
	testutils.AddServices(t, srv, testServices)

	services := make([]string, N)
	for i, s := range testServices {
		services[i] = s.Name
	}
	return makeConfig(t, services, srv.HTTPSAddr, true)
}

// writes the config files out to disk, returns path to them and cleanup func
func makeConfig(t testing.TB, services []string, addr string, tls bool,
) (string, func() error) {
	tmpDir := fmt.Sprintf("%s%s", tempDirPrefix, "test_services")
	cleanup := testutils.MakeTempDir(t, tmpDir)

	configPath := filepath.Join(tmpDir, configFile)
	err := testutils.WriteFile(configPath, configContents(addr, tls, services))
	require.NoError(t, err)

	return configPath, cleanup
}

// returns templated contents of config file
func configContents(address string, tls bool, services []string) string {
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
  source = "../../../test_modules/null_resource"
  providers = ["local"]
  services = %s
}

`, address, tls, pwd, sservices)
}
