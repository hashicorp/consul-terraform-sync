package api

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/stretchr/testify/require"
)

const (
	// CTSOnceModeFlag is an optional flag to run CTS
	CTSOnceModeFlag = "-once"
	// CTSDevModeFlag is an optional flag to run CTS with development client
	CTSDevModeFlag = "--client-type=development"
	// CTSInspectFlag is an optional flag to run CTS in inspect mode
	CTSInspectFlag = "-inspect"
)

// StartCTS starts the CTS from binary and returns a function to stop CTS. If
// running once-mode, the function will block until complete so no need to use
// stop function
func StartCTS(t *testing.T, configPath string, opts ...string) (*Client, func(t *testing.T)) {
	return configureCTS(t, HTTPScheme, configPath, TLSConfig{}, opts...)
}

// StartCTSSecure starts the CTS from binary using the https scheme for connections and returns a function to stop CTS. If
// running once-mode, the function will block until complete so no need to use
// stop function
func StartCTSSecure(t *testing.T, configPath string, tlsConfig TLSConfig, opts ...string) (*Client, func(t *testing.T)) {
	return configureCTS(t, HTTPSScheme, configPath, tlsConfig, opts...)
}

func configureCTS(t *testing.T, scheme string, configPath string, tlsConfig TLSConfig, opts ...string) (*Client, func(t *testing.T)) {
	opts = append(opts, fmt.Sprintf("--config-file=%s", configPath))
	cmd := exec.Command("consul-terraform-sync", opts...)
	// capture logging to output on error
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	// uncomment to see all logs
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	// run CTS in once-mode
	for _, opt := range opts {
		if opt == CTSOnceModeFlag || opt == CTSInspectFlag {
			err := cmd.Run() // blocking
			require.NoError(t, err, buf.String())
			return nil, func(t *testing.T) {}
		}
	}

	// Grab port for API when CTS is running as a daemon and append to config file
	port := testutils.FreePort(t)
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	require.NoError(t, err)
	_, err = f.WriteString(fmt.Sprintf("port = %d\n", port))
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)

	// start CTS regularly
	err = cmd.Start()
	require.NoError(t, err)

	require.NoError(t, err)

	clientConfig := &ClientConfig{
		URL:       fmt.Sprintf("%s://localhost:%d", scheme, port),
		TLSConfig: tlsConfig,
	}

	ctsClient, err := NewClient(clientConfig, nil)
	require.NoError(t, err)

	return ctsClient, func(t *testing.T) {
		defer func() {
			logLen := buf.Len()
			// if the test has failed for any reason, print log snippet
			if t.Failed() && logLen > 0 {
				logStr := buf.String()
				numChar := 6000
				if logLen < 6000 {
					numChar = logLen
				}
				t.Logf("\nFailed Test: %s\nCTS logs:\n\n...%s", t.Name(),
					logStr[logLen-numChar:])
			}
		}()

		err = cmd.Process.Signal(os.Interrupt)
		require.NoError(t, err)

		sigintErr := errors.New("signal: interrupt")
		if err := cmd.Wait(); err != nil {
			require.Equal(t, sigintErr, err)
		}
	}
}

func WaitForEvent(t *testing.T, client *Client, taskName string, start time.Time, timeout time.Duration) {
	polling := make(chan struct{})
	stopPolling := make(chan struct{})

	go func() {
		for {
			select {
			case <-stopPolling:
				return
			default:
				q := &QueryParam{IncludeEvents: true}
				results, err := client.Status().Task(taskName, q)
				if err != nil {
					continue
				}
				task, ok := results[taskName]
				if !ok {
					continue
				}
				if len(task.Events) == 0 {
					continue
				}
				mostRecent := task.Events[0]
				if mostRecent.StartTime.After(start) && mostRecent.EndTime.After(start) {
					// start is the time before a trigger occurs, so this checks if the
					// most recent event started and completed after the trigger
					polling <- struct{}{}
					return
				}
				time.Sleep(time.Second)
			}
		}
	}()

	select {
	case <-polling:
		return
	case <-time.After(timeout):
		close(stopPolling)
		t.Fatalf("\nError: timed out after waiting for %v for new event for task %q\n",
			timeout, taskName)
	}
}
