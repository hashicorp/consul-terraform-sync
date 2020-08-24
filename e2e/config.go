package e2e

import (
	"fmt"
	"os"
)

const (
	dbTaskName  = "e2e_task_api_db"
	webTaskName = "e2e_task_api_web"
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
	skip_verify = true
	path = "%s"
	data_dir = "%s"
	working_dir = "%s"
}
`, cwd, dir, dir)
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
	return `log_level = "trace"

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

provider "local" {}
`
}
