//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTasksUpdate tests multiple tasks are triggered on service registration
// and de-registration by verifying the content of terraform.tfvars
func TestTasksUpdate(t *testing.T) {
	setParallelism(t)

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "multiple_tasks")
	cleanup := testutils.MakeTempDir(t, tempDir)

	apiTaskName := "e2e_task_api"
	apiTask := fmt.Sprintf(`
working_dir = "%s"

task {
	name = "%s"
	description = "basic read-write e2e task api only"
	condition "services" {
		names = ["api"]
	}
	providers = ["local"]
	module = "./test_modules/local_instances_file"
}
`, tempDir, apiTaskName)
	configPath := filepath.Join(tempDir, configFile)
	var config hclConfig
	config = config.appendConsulBlock(srv).appendTerraformBlock().
		appendDBTask().appendWebTask().appendString(apiTask)
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)

	t.Run("once mode", func(t *testing.T) {
		// Wait for all three tasks to execute once
		err := cts.WaitForAPI(defaultWaitForAPI * 3)
		require.NoError(t, err)

		// Verify Catalog information is reflected in terraform.tfvars
		expectedTaskServices := map[string][]string{
			apiTaskName: {"api"},
			dbTaskName:  {"api", "db"},
			webTaskName: {"api", "web"},
		}
		for taskName, expected := range expectedTaskServices {
			tfvarsFile := filepath.Join(tempDir, taskName, "terraform.tfvars")
			serviceIDs := loadTFVarsServiceIDs(t, tfvarsFile)
			if !assert.Equal(t, expected, serviceIDs) {
				t.Fail()
			}
		}
	})

	t.Run("register services", func(t *testing.T) {
		// Register api and web instances
		apiInstance := testutil.TestService{
			ID:      "api_new",
			Name:    "api",
			Address: "5.6.7.8",
			Port:    8080,
		}
		webInstance := testutil.TestService{
			ID:      "web_new",
			Name:    "web",
			Address: "5.6.7.8",
			Port:    8080,
		}

		now := time.Now()
		testutils.RegisterConsulService(t, srv, apiInstance, defaultWaitForRegistration)
		api.WaitForEvent(t, cts, webTaskName, now, defaultWaitForEvent) // only check one task

		now = time.Now()
		testutils.RegisterConsulService(t, srv, webInstance, defaultWaitForRegistration)
		// takes a little longer due to consecutive registrations
		api.WaitForEvent(t, cts, webTaskName, now, defaultWaitForEvent*2)

		// Verify updated Catalog information is reflected in terraform.tfvars
		expectedTaskServices := map[string][]string{
			apiTaskName: {"api", "api_new"},
			dbTaskName:  {"api", "api_new", "db"},
			webTaskName: {"api", "api_new", "web", "web_new"},
		}
		for taskName, expected := range expectedTaskServices {
			tfvarsFile := filepath.Join(tempDir, taskName, "terraform.tfvars")
			serviceIDs := loadTFVarsServiceIDs(t, tfvarsFile)
			if !assert.Equal(t, expected, serviceIDs) {
				t.Fail()
			}
		}
	})

	t.Run("deregister service", func(t *testing.T) {
		// Deregister service
		testutils.DeregisterConsulService(t, srv, "api_new")
		fullWait := defaultWaitForRegistration + defaultWaitForEvent
		now := time.Now()
		api.WaitForEvent(t, cts, apiTaskName, now, fullWait)
		api.WaitForEvent(t, cts, dbTaskName, now, fullWait)
		api.WaitForEvent(t, cts, webTaskName, now, fullWait)

		// Verify updated Catalog information is reflected in terraform.tfvars
		expectedTaskServices := map[string][]string{
			apiTaskName: {"api"},
			dbTaskName:  {"api", "db"},
			webTaskName: {"api", "web", "web_new"},
		}
		for taskName, expected := range expectedTaskServices {
			tfvarsFile := filepath.Join(tempDir, taskName, "terraform.tfvars")
			serviceIDs := loadTFVarsServiceIDs(t, tfvarsFile)
			if !assert.Equal(t, expected, serviceIDs) {
				t.Fail()
			}
		}
	})

	_ = cleanup()
}

func loadTFVarsServiceIDs(t *testing.T, file string) []string {
	// This is a bit hacky using regex but simpler than re-implementing syntax
	// parsing for Terraform variables
	content := testutils.CheckFile(t, true, file, "")

	var ids []string
	re := regexp.MustCompile(`\s+id\s+= "([^"]+)`)
	matches := re.FindAllSubmatch([]byte(content), -1)
	for _, match := range matches {
		ids = append(ids, string(match[1]))
	}

	sort.Strings(ids)
	return ids
}
