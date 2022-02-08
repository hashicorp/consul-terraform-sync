//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
)

const (
	dbTaskName  = "e2e_task_api_db"
	webTaskName = "e2e_task_api_web"

	fakeSuccessTaskName = "fake_handler_success_task"
	fakeFailureTaskName = "fake_handler_failure_task"
	disabledTaskName    = "disabled_task"
)

type hclConfig string

func (c hclConfig) appendString(s string) hclConfig {
	return hclConfig(string(c) + s)
}

func (c hclConfig) write(tb testing.TB, path string) {
	testutils.WriteFile(tb, path, string(c))
}

func (c hclConfig) appendPort(port int) hclConfig {
	return c.appendString(fmt.Sprintf("port = %d", port))
}

func (c hclConfig) appendConsulBlock(consul *testutil.TestServer) hclConfig {
	return c.appendString(fmt.Sprintf(`
consul {
  address = "%s"
  tls {
    enabled = true
    ca_cert = "%s"
  }
}
`, consul.HTTPSAddr, consul.Config.CertFile))
}

func (c hclConfig) appendTLSBlock(config tlsConfig) hclConfig {

	// Build block based on tlsConfig
	/* Example:
	tls {
	  enabled = true
	  cert = "../testutils/certs/localhost_cert.pem"
	  key = "../testutils/certs/localhost_key.pem"
	}
	*/
	s := `
tls {
  enabled = true`

	if config.clientCert != "" {
		s = fmt.Sprintf(s+`
  cert = "%s"`, config.clientCert)
	}

	if config.clientKey != "" {
		s = fmt.Sprintf(s+`
  key = "%s"`, config.clientKey)
	}

	if config.caCert != "" {
		s = fmt.Sprintf(s+`
  ca_cert = "%s"`, config.caCert)
	}

	if config.caPath != "" {
		s = fmt.Sprintf(s+`
  ca_path = "%s"`, config.caPath)
	}

	if config.verifyIncoming != nil {
		s = fmt.Sprintf(s+`
  verify_incoming = %s`, strconv.FormatBool(*config.verifyIncoming))
	}

	s = s + `
}
`
	return c.appendString(s)
}

func (c hclConfig) appendTerraformBlock(opts ...string) hclConfig {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	var optsConfig string
	if len(opts) > 0 {
		optsConfig = "\n" + strings.Join(opts, "\n")
	}

	return c.appendString(fmt.Sprintf(`
driver "terraform" {
	log = true
	path = "%s"%s
}
`, cwd, optsConfig))
}

func dbTask() string {
	return fmt.Sprintf(`
task {
	name = "%s"
	description = "basic read-write e2e task for api & db"
	providers = ["local"]
	module = "./test_modules/local_instances_file"
	condition "services" {
    	names = ["api", "db"]
	}
}`, dbTaskName)
}

func (c hclConfig) appendDBTask() hclConfig {
	return c.appendString(dbTask())
}

func (c hclConfig) appendWebTask() hclConfig {
	return c.appendString(fmt.Sprintf(`
task {
	name = "%s"
	description = "basic read-write e2e task api & web"
	providers = ["local"]
	module = "./test_modules/local_instances_file"
	condition "services" {
		names = ["api", "web"]
	}
}
`, webTaskName))
}

// appendModuleTask adds a task configuration with the given name and module, along with any additional
// task configurations (e.g., condition, providers) provided with the opts parameter
func (c hclConfig) appendModuleTask(name string, module string, opts ...string) hclConfig {
	return c.appendString(moduleTaskConfig(name, module, opts...))
}

// moduleTaskConfig generates a task configuration string with the given name
// and module, along with any additional task configurations (e.g., condition,
// providers) provided with the opts parameter
func moduleTaskConfig(name string, module string, opts ...string) string {
	var optsConfig string
	if len(opts) > 0 {
		optsConfig = "\n" + strings.Join(opts, "\n")
	}

	return fmt.Sprintf(`
task {
	name = "%s"
	description = "e2e test"
	module = "%s"
	condition "services" {
		names = ["api", "web"]
	}
	%s
}
`, name, module, optsConfig)
}

func baseConfig(wd string) hclConfig {
	return hclConfig(fmt.Sprintf(`log_level = "DEBUG"
working_dir = "%s"

buffer_period {
	enabled = false
}

terraform_provider "local" {}
`, wd))
}

// fakeHandlerConfig returns a config file with fake provider that has a handler
// Use for running in development to test cases success/failed events
func fakeHandlerConfig(dir string) hclConfig {
	return hclConfig(fmt.Sprintf(`
working_dir = "%s"

buffer_period {
	enabled = false
}

terraform_provider "fake-sync" {
	alias = "failure"
	name = "failure"
	err = true
	success_first = true
}

terraform_provider "fake-sync" {
	alias = "success"
	name = "success"
	err = false
}

task {
	name = "%s"
	description = "basic e2e task with fake handler. expected to error"
	providers = ["fake-sync.failure"]
	module = "./test_modules/local_instances_file"
	condition "services" {
		names = ["api"]
	}
}

task {
	name = "%s"
	description = "basic e2e task with fake handler. expected to not error"
	providers = ["fake-sync.success"]
	module = "./test_modules/local_instances_file"
	condition "services" {
		names = ["api"]
	}
}

task {
	name = "%s"
	description = "disabled task"
	enabled = false
	providers = ["fake-sync.success"]
	module = "./test_modules/local_instances_file"
	condition "services" {
		names = ["api"]
	}
}
`, dir, fakeFailureTaskName, fakeSuccessTaskName, disabledTaskName))
}

// disabledTaskConfig returns a config file with a task that is disabled
func disabledTaskConfig() string {
	return fmt.Sprintf(`
task {
	name = "%s"
	description = "task is configured as disabled"
	enabled = false
	providers = ["local"]
	module = "./test_modules/local_instances_file"
	condition "services" {
		names = ["api", "web"]
	}
}
`, disabledTaskName)
}
