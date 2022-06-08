//go:build e2e
// +build e2e

package compatibility

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This technically isn't an e2e test, but it downloads a file from the internet,
// so it likely shouldn't be run frequently with other tests.
func TestCompatibility_TFDownload(t *testing.T) {
	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "Compatibility_TFDownload")
	cleanup := testutils.MakeTempDir(t, tempDir)
	defer cleanup()

	conf := config.DefaultTerraformConfig()
	conf.Path = &tempDir
	// Use blank as a default value. Nil results in a panic.
	tfVersion := ""
	conf.Version = &tfVersion

	binaryPath := path.Join(tempDir, "terraform")
	info, err := os.Stat(binaryPath)
	// We want to ensure the file does not exist and remove it if found.
	if !os.IsNotExist(err) {
		require.NoError(t, err, "An unexpected error occurred while checking for old terraform executable.")
	}

	if info != nil {
		err = os.Remove(binaryPath)
		require.NoError(t, err, "An unexpected error occurred while removing an old terraform executable.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	err = driver.InstallTerraform(ctx, conf)
	require.NoError(t, err, "Unable to download terraform executable.")

	// Check to ensure the terraform binary exists.
	info, err = os.Stat(binaryPath)
	require.NoError(t, err, "Unable to find terraform executable that was expected to be installed.")

	ensureTFVersionExecutes(t, binaryPath)

	// Trigger install again. It shouldn't re-download the file.
	firstInstallTime := info.ModTime()
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = driver.InstallTerraform(ctx, conf)
	require.NoError(t, err, "Unexpected error while checking for existing terraform install.")
	info, err = os.Stat(binaryPath)
	require.NoError(t, err, "Unable to find terraform executable that was expected to be installed.")
	assert.Equal(t, firstInstallTime, info.ModTime(), "The Terraform installer should not have downloaded twice.")

	ensureTFVersionExecutes(t, binaryPath)
}

// ensureTFVersionExecutes calls the terraform executable at the provided path.
// Results in a test failure if `terraform version` returns a non-zero status code
// or if the text "Terraform" is not found in the output.
func ensureTFVersionExecutes(t *testing.T, binaryPath string) {
	cmd := exec.Command(binaryPath, "version")
	output, err := cmd.Output()
	require.NoError(t, err, "An error occurred while attempting to execute the downloaded terraform binary.")
	assert.Equal(t, 0, cmd.ProcessState.ExitCode(), "Bad exit code encountered while checking the terraform binary.")
	assert.Contains(t, string(output), "Terraform", "The `terraform version` call did not provide the expected output.")
}
