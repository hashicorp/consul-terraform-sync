package testutils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	// CTSOnceModeFlag is an optional flag to run CTS
	CTSOnceModeFlag = "-once"
	// CTSDevModeFlag is an optional flag to run CTS with development client
	CTSDevModeFlag = "--client-type=development"
)

// StartCTS starts the CTS from binary and returns a function to stop CTS. If
// running once-mode, the function will block until complete so no need to use
// stop function
func StartCTS(t *testing.T, configPath string, opts ...string) func(t *testing.T) {
	opts = append(opts, fmt.Sprintf("--config-file=%s", configPath))
	cmd := exec.Command("consul-terraform-sync", opts...)
	// uncomment to see logs
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	// run CTS in once-mode
	for _, opt := range opts {
		if opt == CTSOnceModeFlag {
			cmd.Run() // blocking
			return func(t *testing.T) {}
		}
	}

	// start CTS regularly
	err := cmd.Start()
	require.NoError(t, err)

	return func(t *testing.T) {
		cmd.Process.Signal(os.Interrupt)
		sigintErr := errors.New("signal: interrupt")
		if err := cmd.Wait(); err != nil {
			require.Equal(t, sigintErr, err)
		}
	}
}
