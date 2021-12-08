//go:build e2e
// +build e2e

// Tests CTS CLI commands interacting using TLS/mTLS with a running CTS in daemon mode.
package e2e

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/command"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/stretchr/testify/assert"
)

const (
	invalidCACert = "../testutils/certs/consul_cert.pem"
	missingCACert = "../testutils/certs/localhost_cert2.pem"
)

// TestE2E_CommandTLSErrors tests error scenarios using CLI commands with TLS. This
// starts up a local Consul server and runs CTS with TLS in dev mode.
func TestE2E_CommandTLSErrors(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "command_tls_errors")

	tlsc := defaultTLSConfig()
	cts := ctsSetupTLS(t, srv, tempDir, dbTask(), tlsc)
	address := cts.FullAddress()

	// Get all test certs and move them to the CA path directory
	certs := []string{
		invalidCACert,
		missingCACert,
	}
	clientCAPath := copyClientCerts(t, certs, tempDir)

	commands := map[string][]string{
		"task disable": {"task", "disable"},
		"task enable":  {"task", "enable"},
	}

	cases := []struct {
		name           string
		args           []string
		envVariables   []string
		outputContains string
	}{
		{
			"connect using wrong scheme",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, strings.Replace(address, "https", "http", 1)),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			[]string{},
			"consider using https scheme",
		},
		{
			"connect using wrong scheme override right scheme from environment",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, strings.Replace(address, "https", "http", 1)),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			[]string{
				fmt.Sprintf("%s-%s", api.EnvAddress, address),
			},
			"consider using https scheme",
		},
		{
			"connect with invalid cert",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, address),
				fmt.Sprintf("-%s=%s", command.FlagCACert, invalidCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			[]string{},
			"signed by unknown authority",
		},
		{
			"connect with ca path that does not include the server certificate ca",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, address),
				fmt.Sprintf("-%s=%s", command.FlagCAPath, clientCAPath),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			[]string{},
			"signed by unknown authority",
		},
		{
			"connect with invalid cert override env to set SSL verify to true",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, address),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			[]string{"%s-%s", api.EnvTLSSSLVerify, "false"},
			"signed by unknown authority",
		},
	}

	for name, cmd := range commands {
		for _, tc := range cases {
			testName := fmt.Sprintf("%s_%s", tc.name, name)
			t.Run(testName, func(t *testing.T) {

				subcmd := cmd
				subcmd = append(subcmd, tc.args...)

				output, err := runSubcommand(t, "", subcmd...)
				assert.Contains(t, output, tc.outputContains)
				assert.Error(t, err)
			})
		}
	}
}

// TestE2E_CommandTLS tests CLI commands using TLS. This
// starts up a local Consul server and runs CTS with TLS in dev mode.
func TestE2E_CommandTLS(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "command_tls")

	tlsc := defaultTLSConfig()
	cts := ctsSetupTLS(t, srv, tempDir, dbTask(), tlsc)

	commands := []struct {
		name           string
		subcmd         []string
		outputContains string
	}{
		{
			name:           "task disable",
			subcmd:         []string{"task", "disable"},
			outputContains: "disable complete!",
		},
		{
			name:           "task enable",
			subcmd:         []string{"task", "enable"},
			outputContains: "enable complete!",
		},
	}

	cases := []struct {
		name         string
		args         []string
		envVariables []string
	}{
		{
			name: "happy path",
			args: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
		},
		{
			name: "happy path environment variables",
			args: []string{
				dbTaskName,
			},
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, cts.FullAddress()),
				fmt.Sprintf("%s=%s", api.EnvTLSCACert, defaultCTSCACert),
				fmt.Sprintf("%s=%s", api.EnvTLSSSLVerify, "true"),
			},
		},
		{
			name: "flags override environment variables",
			args: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, "bogus_address"),
				fmt.Sprintf("%s=%s", api.EnvTLSCACert, "bogus_cert"),
			},
		},
		{
			name: "ssl verify flag set to false",
			args: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "false"),
				dbTaskName,
			},
		},
		{
			name: "ssl verify environment set to false",
			args: []string{
				dbTaskName,
			},
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, cts.FullAddress()),
				fmt.Sprintf("%s=%s", api.EnvTLSSSLVerify, "false"),
			},
		},
	}

	for _, cmd := range commands {
		for _, tc := range cases {
			testName := fmt.Sprintf("%s_%s", tc.name, cmd.name)
			t.Run(testName, func(t *testing.T) {
				subcmd := cmd.subcmd
				subcmd = append(subcmd, tc.args...)

				output, err := runSubCommandWithEnvVars(t, "", tc.envVariables, subcmd...)
				assert.Contains(t, output, cmd.outputContains)
				assert.NoError(t, err)
			})
		}
	}
}

// TestE2E_CommandTLS_CAPath tests CLI commands using TLS providing a CA path instead of a CA cert file. This
// starts up a local Consul server and runs CTS with TLS in dev mode.
func TestE2E_CommandTLS_CAPath(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "command_tls_capath")

	tlsc := tlsConfigWithCAPath(tempDir)
	cts := ctsSetupTLS(t, srv, tempDir, dbTask(), tlsc)

	commands := []struct {
		name           string
		subcmd         []string
		outputContains string
	}{
		{
			name:           "task disable",
			subcmd:         []string{"task", "disable"},
			outputContains: "disable complete!",
		},
		{
			name:           "task enable",
			subcmd:         []string{"task", "enable"},
			outputContains: "enable complete!",
		},
	}

	cases := []struct {
		name         string
		args         []string
		envVariables []string
	}{
		{
			name: "ca path flag",
			args: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCAPath, tlsc.caPath),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
		},
		{
			name: "ca path environment variables",
			args: []string{
				dbTaskName,
			},
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, cts.FullAddress()),
				fmt.Sprintf("%s=%s", api.EnvTLSCAPath, tlsc.caPath),
				fmt.Sprintf("%s=%s", api.EnvTLSSSLVerify, "true"),
			},
		},
		{
			name: "ca path flags override environment variables",
			args: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCAPath, tlsc.caPath),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, "bogus_address"),
				fmt.Sprintf("%s=%s", api.EnvTLSCAPath, "/path/bogus_cert"),
			},
		},
		{
			name: "ca path overrides ca cert",
			args: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagCAPath, "/path/bogus_cert"),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
		},
	}

	for _, cmd := range commands {
		for _, tc := range cases {
			testName := fmt.Sprintf("%s_%s", tc.name, cmd.name)
			t.Run(testName, func(t *testing.T) {
				subcmd := cmd.subcmd
				subcmd = append(subcmd, tc.args...)

				output, err := runSubCommandWithEnvVars(t, "", tc.envVariables, subcmd...)
				assert.Contains(t, output, cmd.outputContains)
				assert.NoError(t, err)
			})
		}
	}
}

// TestE2E_CommandMTLSErrors tests error scenarios using CLI commands with mTLS. This
// starts up a local Consul server and runs CTS with TLS in dev mode.
func TestE2E_CommandMTLSErrors(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "command_mtls_errors")

	tlsc := mtlsConfigWithCAPath(tempDir)
	cts := ctsSetupTLS(t, srv, tempDir, dbTask(), tlsc)

	// Get all test certs and move them to the CA path directory
	certs := []string{
		missingCACert,
		invalidCACert,
	}

	clientCAPath := copyClientCerts(t, certs, tempDir)

	commands := map[string][]string{
		"task disable": {"task", "disable"},
		"task enable":  {"task", "enable"},
	}

	cases := []struct {
		name           string
		args           []string
		outputContains string
	}{
		{
			"connect with invalid ca cert",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, invalidCACert),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, defaultCTSClientCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, defaultCTSClientKey),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			"signed by unknown authority",
		},
		{
			"connect with client ca path that does not include server cert ca",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCAPath, clientCAPath),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, defaultCTSClientCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, defaultCTSClientKey),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			"signed by unknown authority",
		},
		{
			"no client cert key pair provided",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			"bad certificate",
		},
		{
			"ssl verify disabled and no cert key pair provided",
			[]string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "false"),
				dbTaskName,
			},
			"bad certificate",
		},
	}

	for name, cmd := range commands {
		for _, tc := range cases {
			testName := fmt.Sprintf("%s_%s", tc.name, name)
			t.Run(testName, func(t *testing.T) {
				subcmd := cmd
				subcmd = append(subcmd, tc.args...)

				output, err := runSubcommand(t, "", subcmd...)
				assert.Contains(t, output, tc.outputContains)
				assert.Error(t, err)
			})
		}
	}
}

// TestE2E_CommandMTLS tests CLI commands using mTLS. This
// starts up a local Consul server and runs CTS with TLS in dev mode.
func TestE2E_CommandMTLS(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "command_mtls")

	tlsc := defaultMTLSConfig()
	cts := ctsSetupTLS(t, srv, tempDir, dbTask(), tlsc)

	commands := []struct {
		name           string
		subcmd         []string
		outputContains string
	}{
		{
			name:           "task disable",
			subcmd:         []string{"task", "disable"},
			outputContains: "disable complete!",
		},
		{
			name:           "task enable",
			subcmd:         []string{"task", "enable"},
			outputContains: "enable complete!",
		},
	}

	cases := []struct {
		name         string
		args         []string
		envVariables []string
	}{
		{
			name: "happy path",
			args: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, defaultCTSClientCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, defaultCTSClientKey),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
		},
		{
			name: "happy path environment variables",
			args: []string{
				dbTaskName,
			},
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, cts.FullAddress()),
				fmt.Sprintf("%s=%s", api.EnvTLSCACert, defaultCTSCACert),
				fmt.Sprintf("%s=%s", api.EnvTLSSSLVerify, "true"),
				fmt.Sprintf("%s=%s", api.EnvTLSClientCert, defaultCTSClientCert),
				fmt.Sprintf("%s=%s", api.EnvTLSClientKey, defaultCTSClientKey),
			},
		},
		{
			name: "flags override environment variables",
			args: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, defaultCTSClientCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, defaultCTSClientKey),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, "bogus_address"),
				fmt.Sprintf("%s=%s", api.EnvTLSCACert, "bogus_ca_cert"),
				fmt.Sprintf("%s=%s", api.EnvTLSSSLVerify, "true"),
				fmt.Sprintf("%s=%s", api.EnvTLSClientCert, "bogus_client_cert"),
				fmt.Sprintf("%s=%s", api.EnvTLSClientKey, "bogus_client_key"),
			},
		},
		{
			name: "ssl verify disabled",
			args: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, defaultCTSClientCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, defaultCTSClientKey),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "false"),
				dbTaskName,
			},
		},
	}

	for _, cmd := range commands {
		for _, tc := range cases {
			testName := fmt.Sprintf("%s_%s", tc.name, cmd.name)
			t.Run(testName, func(t *testing.T) {
				subcmd := cmd.subcmd
				subcmd = append(subcmd, tc.args...)

				output, err := runSubCommandWithEnvVars(t, "", tc.envVariables, subcmd...)
				assert.Contains(t, output, cmd.outputContains)
				assert.NoError(t, err)
			})
		}
	}
}

// TestE2E_CommandMTLS_CAPath tests CLI commands using mTLS providing a CAPath rather than a CA cert file. This
// starts up a local Consul server and runs CTS with TLS in dev mode.
func TestE2E_CommandMTLS_CAPath(t *testing.T) {
	t.Parallel()

	srv := newTestConsulServer(t)
	defer srv.Stop()

	tempDir := fmt.Sprintf("%s%s", tempDirPrefix, "command_mtls_capath")

	tlsc := mtlsConfigWithCAPath(tempDir)
	cts := ctsSetupTLS(t, srv, tempDir, dbTask(), tlsc)

	commands := []struct {
		name           string
		subcmd         []string
		outputContains string
	}{
		{
			name:           "task disable",
			subcmd:         []string{"task", "disable"},
			outputContains: "disable complete!",
		},
		{
			name:           "task enable",
			subcmd:         []string{"task", "enable"},
			outputContains: "enable complete!",
		},
	}

	cases := []struct {
		name         string
		args         []string
		envVariables []string
	}{
		{
			name: "ca path flag",
			args: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCAPath, tlsc.caPath),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, defaultCTSClientCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, defaultCTSClientKey),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
		},
		{
			name: "using alternate cert",
			args: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCACert, defaultCTSCACert),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, alternateCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, alternateKey),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
		},
		{
			name: "ca path environment variables",
			args: []string{
				dbTaskName,
			},
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, cts.FullAddress()),
				fmt.Sprintf("%s=%s", api.EnvTLSCAPath, tlsc.caPath),
				fmt.Sprintf("%s=%s", api.EnvTLSSSLVerify, "true"),
				fmt.Sprintf("%s=%s", api.EnvTLSClientCert, defaultCTSClientCert),
				fmt.Sprintf("%s=%s", api.EnvTLSClientKey, defaultCTSClientKey),
			},
		},
		{
			name: "ca override environment variables",
			args: []string{
				fmt.Sprintf("-%s=%s", command.FlagHTTPAddr, cts.FullAddress()),
				fmt.Sprintf("-%s=%s", command.FlagCAPath, tlsc.caPath),
				fmt.Sprintf("-%s=%s", command.FlagClientCert, defaultCTSClientCert),
				fmt.Sprintf("-%s=%s", command.FlagClientKey, defaultCTSClientKey),
				fmt.Sprintf("-%s=%s", command.FlagSSLVerify, "true"),
				dbTaskName,
			},
			envVariables: []string{
				fmt.Sprintf("%s=%s", api.EnvAddress, "bogus_address"),
				fmt.Sprintf("%s=%s", api.EnvTLSCAPath, "path/bogus_ca_path"),
				fmt.Sprintf("%s=%s", api.EnvTLSSSLVerify, "true"),
				fmt.Sprintf("%s=%s", api.EnvTLSClientCert, "bogus_client_cert"),
				fmt.Sprintf("%s=%s", api.EnvTLSClientKey, "bogus_client_key"),
			},
		},
	}

	for _, cmd := range commands {
		for _, tc := range cases {
			testName := fmt.Sprintf("%s_%s", tc.name, cmd.name)
			t.Run(testName, func(t *testing.T) {
				subcmd := cmd.subcmd
				subcmd = append(subcmd, tc.args...)

				output, err := runSubCommandWithEnvVars(t, "", tc.envVariables, subcmd...)
				assert.Contains(t, output, cmd.outputContains)
				assert.NoError(t, err)
			})
		}
	}
}

func copyClientCerts(t *testing.T, certsToCopy []string, tempDir string) string {
	// Get all test certs and move them to the CA path directory
	clientCAPath := filepath.Join(tempDir, "clientCert")
	delClientDir := testutils.MakeTempDir(t, clientCAPath)
	t.Cleanup(func() {
		delClientDir()
	})
	testutils.CopyFiles(t, certsToCopy, clientCAPath)

	return clientCAPath
}
