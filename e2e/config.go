package e2e

import (
	"fmt"
	"os"
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

func (c hclConfig) appendTerraformBlock(dir string, opts ...string) hclConfig {
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
	path = "%s"
	working_dir = "%s"%s
}
`, cwd, dir, optsConfig))
}

func (c hclConfig) appendDBTask() hclConfig {
	return c.appendString(fmt.Sprintf(`
task {
	name = "%s"
	description = "basic read-write e2e task for api & db"
	services = ["api", "db"]
	providers = ["local"]
	source = "./test_modules/local_instances_file"
}
`, dbTaskName))
}

func (c hclConfig) appendWebTask() hclConfig {
	return c.appendString(fmt.Sprintf(`
task {
	name = "%s"
	description = "basic read-write e2e task api & web"
	services = ["api", "web"]
	providers = ["local"]
	source = "./test_modules/local_instances_file"
}
`, webTaskName))
}

func baseConfig() hclConfig {
	return `log_level = "DEBUG"

buffer_period {
	enabled = false
}

service {
  name = "api"
  description = "backend"
}

service {
  name = "web"
  description = "frontend"
}

service {
    name = "db"
    description = "database"
}

terraform_provider "local" {}
`
}

// fakeHandlerConfig returns a config file with fake provider that has a handler
// Use for running in development to test cases success/failed events
func fakeHandlerConfig() hclConfig {
	return hclConfig(fmt.Sprintf(`
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
	services = ["api"]
	providers = ["fake-sync.failure"]
	source = "./test_modules/local_instances_file"
}

task {
	name = "%s"
	description = "basic e2e task with fake handler. expected to not error"
	services = ["api"]
	providers = ["fake-sync.success"]
	source = "./test_modules/local_instances_file"
}

task {
	name = "%s"
	description = "disabled task"
	enabled = false
	services = ["api"]
	providers = ["fake-sync.success"]
	source = "./test_modules/local_instances_file"
}
`, fakeFailureTaskName, fakeSuccessTaskName, disabledTaskName))
}

// disabledTaskConfig returns a config file with a task that is disabled
func disabledTaskConfig() hclConfig {
	return hclConfig(fmt.Sprintf(`
task {
	name = "%s"
	description = "task is configured as disabled"
	enabled = false
	services = ["api", "web"]
	providers = ["local"]
	source = "./test_modules/local_instances_file"
}

service {
	name = "api"
	description = "backend"
}
`, disabledTaskName))
}

func panosBadCredConfig() hclConfig {
	return `log_level = "trace"
terraform_provider "panos" {
	hostname = "10.10.10.10"
	api_key = "badapikey_1234"
}

task {
	name = "panos-bad-cred-e2e-test"
	description = "panos handler should error and stop sync after once"
	source = "findkim/ngfw/panos"
	version = "0.0.1-beta5"
	providers = ["panos"]
	services = ["web"]
}
`
}
