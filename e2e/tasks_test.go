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
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "multiple_tasks")
	delete := testutils.MakeTempDir(t, tempDir)
	testutils.MakeTempDir(t, tempDir)

	apiTaskName := "e2e_task_api"
	apiTask := fmt.Sprintf(`
task {
	name = "%s"
	description = "basic read-write e2e task api only"
	services = ["api"]
	providers = ["local"]
	source = "./test_modules/e2e_basic_task"
}
`, apiTaskName)
	configPath := filepath.Join(tempDir, configFile)
	var config hclConfig
	config = config.appendConsulBlock(srv).appendTerraformBlock(tempDir).
		appendDBTask().appendWebTask().appendString(apiTask)
	config.write(t, configPath)

	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)

	t.Run("once mode", func(t *testing.T) {
		// Wait for tasks to execute once
		err := cts.WaitForAPI(15 * time.Second)
		require.NoError(t, err)

		// Verify Catalog information is reflected in terraform.tfvars
		expectedTaskServices := map[string][]string{
			apiTaskName: []string{"api"},
			dbTaskName:  []string{"api", "db"},
			webTaskName: []string{"api", "web"},
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
		testutils.RegisterConsulService(t, srv, apiInstance, testutil.HealthPassing)
		testutils.RegisterConsulService(t, srv, webInstance, testutil.HealthPassing)

		// Wait for CTS to detect changes and run tasks
		time.Sleep(15 * time.Second)

		// Verify updated Catalog information is reflected in terraform.tfvars
		expectedTaskServices := map[string][]string{
			apiTaskName: []string{"api", "api_new"},
			dbTaskName:  []string{"api", "api_new", "db"},
			webTaskName: []string{"api", "api_new", "web", "web_new"},
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
		time.Sleep(15 * time.Second)

		// Verify updated Catalog information is reflected in terraform.tfvars
		expectedTaskServices := map[string][]string{
			apiTaskName: []string{"api"},
			dbTaskName:  []string{"api", "db"},
			webTaskName: []string{"api", "web", "web_new"},
		}
		for taskName, expected := range expectedTaskServices {
			tfvarsFile := filepath.Join(tempDir, taskName, "terraform.tfvars")
			serviceIDs := loadTFVarsServiceIDs(t, tfvarsFile)
			if !assert.Equal(t, expected, serviceIDs) {
				t.Fail()
			}
		}
	})

	delete()
}

func loadTFVarsServiceIDs(t *testing.T, file string) []string {
	// This is a bit hacky using regex but simpler than re-implementing syntax
	// parsing for Terraform variables
	content := testutils.CheckFile(t, true, file, "")

	var ids []string
	re := regexp.MustCompile(`\s+id\s+\= \"([^"]+)`)
	matches := re.FindAllSubmatch([]byte(content), -1)
	for _, match := range matches {
		ids = append(ids, string(match[1]))
	}

	sort.Strings(ids)
	return ids
}
