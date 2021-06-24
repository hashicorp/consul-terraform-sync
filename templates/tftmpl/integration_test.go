// +build integration

package tftmpl

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	goVersion "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestInitRootModule(t *testing.T) {
	dir, err := ioutil.TempDir(".", "consul-terraform-sync-tftmpl-")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	// set directory permission to test files inheriting the permissions
	expectedPerm := os.FileMode(0660)

	input := RootModuleInputData{
		TerraformVersion: goVersion.Must(goVersion.NewSemver("0.99.9")),
		Backend: map[string]interface{}{
			"consul": map[string]interface{}{
				"scheme": "https",
				"path":   "consul-terraform-sync/terraform",
			},
		},
		Providers: []hcltmpl.NamedBlock{hcltmpl.NewNamedBlock(
			map[string]interface{}{
				"testProvider": map[string]interface{}{
					"alias": "tp",
					"obj": map[string]interface{}{
						"username": "name",
						"id":       "123",
					},
					"attr":  "value",
					"count": 10,
				},
			})},
		ProviderInfo: map[string]interface{}{
			"testProvider": map[string]interface{}{
				"version": "1.0.0",
				"source":  "namespace/testProvider",
			},
		},
		Task: Task{
			Description: "user description for task named 'test'",
			Name:        "test",
			Source:      "namespace/consul-terraform-sync/consul//modules/test",
			Version:     "0.0.0",
		},
		Variables: hcltmpl.Variables{
			"one":       cty.NumberIntVal(1),
			"bool_true": cty.BoolVal(true),
		},
	}
	input.Init()
	err = InitRootModule(&input, dir, expectedPerm, false)
	assert.NoError(t, err)

	files := []struct {
		GoldenFile string
		ActualFile string
	}{
		{
			"testdata/main.tf",
			filepath.Join(dir, RootFilename),
		}, {
			"testdata/variables.tf",
			filepath.Join(dir, VarsFilename),
		},
	}

	for _, f := range files {
		actual, err := ioutil.ReadFile(f.ActualFile)
		require.NoError(t, err)
		checkGoldenFile(t, f.GoldenFile, actual)

		info, err := os.Stat(f.ActualFile)
		require.NoError(t, err)
		assert.Equal(t, expectedPerm, info.Mode().Perm())
	}
}

func TestRenderTFVarsTmpl(t *testing.T) {
	// Integration test to verify the tfvars templated file can be rendered
	// with Consul service information.

	cases := []struct {
		name            string
		goldenFile      string
		registerAPI     bool
		registerAPI2    bool
		registerAPISrv2 bool
		registerWeb     bool
	}{
		{
			"happy path",
			"testdata/terraform.tfvars",
			true,
			true,
			false,
			true,
		},
		{
			"no instances of any service registered",
			"testdata/no_services.tfvars",
			false,
			false,
			false,
			false,
		},
		{
			"no instances of service alphabetically first registered",
			"testdata/only_web_service.tfvars",
			false,
			false,
			false,
			true,
		},
		{
			"no instances of service alphabetically last registered",
			"testdata/only_api_service.tfvars",
			true,
			true,
			true,
			false,
		},
	}

	for _, tc := range cases {
		tb := &testutils.TestingTB{}
		t.Run(tc.name, func(t *testing.T) {

			// Setup Consul server
			log.SetOutput(ioutil.Discard)
			srv, err := testutil.NewTestServerConfigT(tb,
				func(c *testutil.TestServerConfig) {
					c.LogLevel = "warn"
					c.Stdout = ioutil.Discard
					c.Stderr = ioutil.Discard

					// Hardcode node info so it doesn't change per run
					c.NodeName = "worker-01"
					c.NodeID = "39e5a7f5-2834-e16d-6925-78167c9f50d8"
				})
			require.NoError(t, err, "failed to start consul server 1")
			defer srv.Stop()

			// Register services
			if tc.registerAPI {
				srv.AddAddressableService(t, "api", testutil.HealthPassing,
					"1.2.3.4", 8080, []string{"tag"})
			}
			if tc.registerWeb {
				srv.AddAddressableService(t, "web", testutil.HealthPassing,
					"1.1.1.1", 8000, []string{})
			}

			// Register another api service instance (with unique ID)
			if tc.registerAPI2 {
				service := testutil.TestService{
					ID:      "api-2",
					Name:    "api",
					Tags:    []string{"tag"},
					Address: "5.6.7.8",
					Port:    8080,
				}
				testutils.RegisterConsulService(t, srv, service, testutil.HealthPassing, 5*time.Second)
			}

			// Setup another server with an identical API service
			if tc.registerAPISrv2 {
				srv2, err := testutil.NewTestServerConfigT(t,
					func(c *testutil.TestServerConfig) {
						c.Bootstrap = false
						c.LogLevel = "warn"
						c.Stdout = ioutil.Discard
						c.Stderr = ioutil.Discard

						// Hardcode node info so it doesn't change per run
						c.NodeName = "worker-02"
						c.NodeID = "d407a592-e93c-4d8e-8a6d-aba853d1e067"
					})
				require.NoError(t, err, "failed to start consul server 2")
				defer srv2.Stop()

				// Join the servers together
				srv.JoinLAN(t, srv2.LANAddr)

				srv2.AddAddressableService(t, "api", testutil.HealthPassing,
					"1.2.3.4", 8080, []string{"tag"})
			}

			// Setup watcher
			clients := hcat.NewClientSet()
			clients.AddConsul(hcat.ConsulInput{
				Address: srv.HTTPAddr,
			})
			defer clients.Stop()

			w := hcat.NewWatcher(hcat.WatcherInput{
				Clients: clients,
				Cache:   hcat.NewStore(),
			})
			r := hcat.NewResolver()

			// Load template from disk and render
			contents, err := ioutil.ReadFile("testdata/terraform.tfvars.tmpl")
			require.NoError(t, err)

			input := hcat.TemplateInput{
				Contents:      string(contents),
				ErrMissingKey: true,
				FuncMapMerge:  HCLTmplFuncMap(nil),
			}
			tmpl := hcat.NewTemplate(input)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
			defer cancel()

			gld, err := ioutil.ReadFile(tc.goldenFile)
			if err != nil {
				require.NoError(t, err)
			}
			retry := 0
			for {
				re, err := r.Run(tmpl, w)
				require.NoError(t, err)

				if re.Complete {
					// there may be a race with the consul services registering
					// let's retry once.
					if (string(gld) != string(re.Contents)) && retry < 1 {
						retry++
						continue
					}
					assert.Equal(t, string(gld), string(re.Contents))
					break
				}

				err = <-w.WaitCh(ctx)
				require.NoError(t, err)
			}
		})
	}
}
