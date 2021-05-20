package api

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/testutils"
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
func StartCTS(t *testing.T, configPath string, opts ...string) (*Client, func(t *testing.T)) {
	opts = append(opts, fmt.Sprintf("--config-file=%s", configPath))
	cmd := exec.Command("consul-terraform-sync", opts...)
	// uncomment to see logs
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	// run CTS in once-mode
	for _, opt := range opts {
		if opt == CTSOnceModeFlag {
			cmd.Run() // blocking
			return nil, func(t *testing.T) {}
		}
	}

	// Grab port for API when CTS is running as a daemon and append to config file
	port, err := testutils.FreePort()
	require.NoError(t, err)
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	require.NoError(t, err)
	_, err = f.WriteString(fmt.Sprintf("port = %d\n", port))
	f.Close()
	require.NoError(t, err)

	// start CTS regularly
	err = cmd.Start()
	require.NoError(t, err)

	ctsClient := NewClient(&ClientConfig{Port: port}, nil)

	return ctsClient, func(t *testing.T) {
		cmd.Process.Signal(os.Interrupt)
		sigintErr := errors.New("signal: interrupt")
		if err := cmd.Wait(); err != nil {
			require.Equal(t, sigintErr, err)
		}
	}
}
