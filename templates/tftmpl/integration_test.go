// +build integration

package tftmpl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestInitRootModule(t *testing.T) {
	dir, err := ioutil.TempDir("", "consul-nia-tftmpl-")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	input := RootModuleInputData{
		Backend: map[string]interface{}{
			"consul": map[string]interface{}{
				"scheme": "https",
				"path":   "consul-nia/terraform",
			},
		},
		Providers: []map[string]interface{}{{
			"testProvider": map[string]interface{}{
				"alias": "tp",
				"attr":  "value",
				"count": 10,
			},
		}},
		ProviderInfo: map[string]interface{}{
			"testProvider": map[string]interface{}{
				"version": "1.0.0",
				"source":  "namespace/testProvider",
			},
		},
		Task: Task{
			Description: "user description for task named 'test'",
			Name:        "test",
			Source:      "namespace/consul-nia/consul//modules/test",
			Version:     "0.0.0",
		},
		Variables: Variables{
			"one":       cty.NumberIntVal(1),
			"bool_true": cty.BoolVal(true),
		},
	}
	input.Init()
	err = InitRootModule(&input, dir, false)
	assert.NoError(t, err)

	files := []struct {
		GoldenFile string
		ActualFile string
	}{
		{
			"testdata/main.tf",
			filepath.Join(dir, input.Task.Name, RootFilename),
		}, {
			"testdata/variables.tf",
			filepath.Join(dir, input.Task.Name, VarsFilename),
		},
	}

	for _, f := range files {
		actual, err := ioutil.ReadFile(f.ActualFile)
		require.NoError(t, err)
		checkGoldenFile(t, f.GoldenFile, actual)
	}
}

func TestRenderTFVarsTmpl(t *testing.T) {
	// Integration test to verify the tfvars templated file can be rendered
	// with Consul service information.

	// Setup Consul server
	log.SetOutput(ioutil.Discard)
	srv, err := testutil.NewTestServerConfig(func(c *testutil.TestServerConfig) {
		c.LogLevel = "warn"
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard

		// Hardcode node info so it doesn't change per run
		c.NodeName = "node-39e5a7f5-2834-e16d-6925-78167c9f50d8"
		c.NodeID = "39e5a7f5-2834-e16d-6925-78167c9f50d8"
	})
	require.NoError(t, err, "failed to start consul server 1")
	defer srv.Stop()

	// Register services
	srv.AddAddressableService(t, "api", testutil.HealthPassing,
		"1.2.3.4", 8080, []string{"tag"})
	srv.AddAddressableService(t, "web", testutil.HealthPassing,
		"1.1.1.1", 8000, []string{})

	// Register another api service instance (with unique ID)
	service := testutil.TestService{
		ID:      "api-2",
		Name:    "api",
		Tags:    []string{"tag"},
		Address: "5.6.7.8",
		Port:    8080,
	}
	registerService(t, srv, service, testutil.HealthPassing)

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
	goldenFile := "testdata/terraform.tfvars"
	contents, err := ioutil.ReadFile("testdata/terraform.tfvars.tmpl")
	require.NoError(t, err)

	input := hcat.TemplateInput{
		Contents:      string(contents),
		ErrMissingKey: true,
		FuncMapMerge:  HCLTmplFuncMap,
	}
	tmpl := hcat.NewTemplate(input)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	for {
		re, err := r.Run(tmpl, w)
		require.NoError(t, err)

		if re.Complete {
			checkGoldenFile(t, goldenFile, re.Contents)
			break
		}

		err = <-w.WaitCh(ctx)
		assert.NoError(t, err)
	}
}

// registerService is a helper function to regsiter a service to the Consul
// Catalog. The Consul sdk/testutil package currently does not support a method
// to register multiple service instances, distinguished by their IDs.
func registerService(t *testing.T, srv *testutil.TestServer, s testutil.TestService, health string) {
	var body bytes.Buffer
	enc := json.NewEncoder(&body)
	require.NoError(t, enc.Encode(&s))

	u := fmt.Sprintf("http://%s/v1/agent/service/register", srv.HTTPAddr)
	req, err := http.NewRequest("PUT", u, io.Reader(&body))
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	srv.AddCheck(t, s.ID, s.ID, testutil.HealthPassing)
}
