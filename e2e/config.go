package e2e

import (
	"fmt"
	"os"
)

const (
	dbTaskName  = "e2e_task_api_db"
	webTaskName = "e2e_task_api_web"

	fakeSuccessTaskName = "fake_handler_success_task"
	fakeFailureTaskName = "fake_handler_failure_task"
)

// oneTaskConfig returns a basic config file with a single task
// Use for testing runtime errors
func oneTaskConfig(consulAddr, tempDir string) string {
	return baseConfig() + consulBlock(consulAddr) + terraformBlock(tempDir) + dbTask()
}

// twoTaskConfig returns a basic use case config file
// Use for confirming specific resource / statefile output
func twoTaskConfig(consulAddr, tempDir string) string {
	return oneTaskConfig(consulAddr, tempDir) + webTask()
}

// panosConfig returns a config file with panos provider with bad config
// Use for testing handlers erroring out
func panosConfig(consulAddr, tempDir string) string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	tfBlock := fmt.Sprintf(`
driver "terraform" {
	log = true
	path = "%s"
	working_dir = "%s"
	required_providers {
		panos = {
			source = "paloaltonetworks/panos"
		}
	}
}`, cwd, tempDir)

	return panosBadCredConfig() + consulBlock(consulAddr) + tfBlock
}

// fakeHandlerConfig returns a config file with fake provider that has a handler
// Use for running in development to test cases success/failed events
func fakeHandlerConfig(consulAddr, tempDir string, port int) string {
	fakeHandlerConfig := fmt.Sprintf(`
port = %d

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
	source = "../../test_modules/e2e_basic_task"
}

task {
	name = "%s"
	description = "basic e2e task with fake handler. expected to not error"
	services = ["api"]
	providers = ["fake-sync.success"]
	source = "../../test_modules/e2e_basic_task"
}
`, port, fakeFailureTaskName, fakeSuccessTaskName)

	return fakeHandlerConfig + consulBlock(consulAddr) + terraformBlock(tempDir)
}

// twoTaskCustomBackendConfig returns a basic config file with two tasks for
// custom backend. Use for confirming resources / state file for custom backend.
//
// Example of customBackend:
// `backend "local" {
// 	path = "custom/terraform.tfstate"
// }`
func twoTaskCustomBackendConfig(consulAddr, tempDir, customBackend string) string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	terraformBlock := fmt.Sprintf(`
driver "terraform" {
	log = true
	path = "%s"
	working_dir = "%s"
	%s
}
`, cwd, tempDir, customBackend)

	return baseConfig() + consulBlock(consulAddr) +
		terraformBlock + dbTask() + webTask()
}

func consulBlock(addr string) string {
	return fmt.Sprintf(`
consul {
    address = "%s"
}
`, addr)
}

func terraformBlock(dir string) string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`
driver "terraform" {
	log = true
	path = "%s"
	working_dir = "%s"
}
`, cwd, dir)
}

func dbTask() string {
	return fmt.Sprintf(`
task {
	name = "%s"
	description = "basic read-write e2e task for api & db"
	services = ["api", "db"]
	providers = ["local"]
	source = "../../test_modules/e2e_basic_task"
}
`, dbTaskName)
}

func webTask() string {
	return fmt.Sprintf(`
task {
	name = "%s"
	description = "basic read-write e2e task api & web"
	services = ["api", "web"]
	providers = ["local"]
	source = "../../test_modules/e2e_basic_task"
}
`, webTaskName)
}

func baseConfig() string {
	return `log_level = "INFO"

# port 0 will automatically select next free port
port = 0

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

func panosBadCredConfig() string {
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
