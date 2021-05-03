// +build e2e

package e2e

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/testutils"
	ctsTestClient "github.com/hashicorp/consul-terraform-sync/testutils/cts"
	"github.com/stretchr/testify/require"
)

// TestCondition_CatalogServices_Include runs the CTS binary. It specifically
// tests a task configured with a catalog service condition with the
// source_includes_var = true. This test confirms that the catalog_service
// definition can be consumed by a module as expected.
func TestCondition_CatalogServices_Include(t *testing.T) {
	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "cs_condition_include")
	delete := testutils.MakeTempDir(t, tempDir)

	conditionTask := `task {
	name = "catalog_task"
	services = ["api"]
	source = "../../test_modules/local_tags_file"
	condition "catalog-services" {
		regexp = "db|web"
		source_includes_var = true
	}
}
`
	config := baseConfig().appendConsulBlock(srv).appendTerraformBlock(tempDir).
		appendString(conditionTask)
	configPath := filepath.Join(tempDir, configFile)
	config.write(t, configPath)

	ctsTestClient.StartCTS(t, configPath, ctsTestClient.CTSOnceModeFlag)

	// confirm that only two files were generated, one for db and one for web
	resourcesPath := fmt.Sprintf("%s/%s", tempDir, resourcesDir)
	files := testutils.CheckDir(t, true, resourcesPath)
	require.Equal(t, 2, len(files))

	contents := testutils.CheckFile(t, true, resourcesPath, "db_tags.txt")
	require.Equal(t, "tag3,tag4", string(contents))

	contents = testutils.CheckFile(t, true, resourcesPath, "web_tags.txt")
	require.Equal(t, "tag2", string(contents))

	delete()
}
