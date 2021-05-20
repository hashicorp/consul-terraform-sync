// +build integration
// +build vault

package testutils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	vaultAPI "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/require"
)

const vaultTestToken = "cts-test-vault-token"

type TestVaultServerConfig struct {
	Token string // default test token will be used if unset
	Port  int    // random port is used if unset
}

// NewTestVaultServer starts a local vault server in dev mode.
func NewTestVaultServer(tb testing.TB, config TestVaultServerConfig) (*vaultAPI.Client, func(tb testing.TB)) {
	token := config.Token
	if token == "" {
		token = vaultTestToken
	}

	var err error
	port := config.Port
	if port == 0 {
		port, err = FreePort()
		require.NoError(tb, err)
	}
	addr := fmt.Sprintf("http://127.0.0.1:%d", port)

	cmd := exec.Command("vault", "server", "-dev")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("VAULT_DEV_LISTEN_ADDRESS=127.0.0.1:%d", port),
		fmt.Sprintf("VAULT_DEV_ROOT_TOKEN_ID=%s", vaultTestToken),
	)
	// uncomment to see logs
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	err = cmd.Start()
	require.NoError(tb, err)

	vaultConf := vaultAPI.DefaultConfig()
	vaultConf.Address = addr
	client, err := vaultAPI.NewClient(vaultConf)
	require.NoError(tb, err)
	client.SetToken(token)

	return client, func(tb testing.TB) {
		cmd.Process.Signal(os.Interrupt)
		if err := cmd.Wait(); err != nil {
			sigintErr := errors.New("signal: interrupt")
			require.Equal(tb, sigintErr, err)
		}
	}
}
