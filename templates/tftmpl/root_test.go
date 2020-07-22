package tftmpl

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	update = flag.Bool("update", false, "update golden files")
	long   = flag.Bool("long", false, "run long tests")
)

func TestInitRootModule(t *testing.T) {
	if !*long {
		t.Skip("test writes to disk, skipping")
	}

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
	}
	input.Init()
	err = InitRootModule(&input, dir, false)
	assert.NoError(t, err)

	files := []struct {
		GoldenFile string
		ActualFile string
	}{
		{
			"testdata/main.tf.golden",
			filepath.Join(dir, input.Task.Name, RootFilename),
		}, {
			"testdata/variables.tf.golden",
			filepath.Join(dir, input.Task.Name, VarsFilename),
		},
	}

	for _, f := range files {
		actual, err := ioutil.ReadFile(f.ActualFile)
		require.NoError(t, err)
		checkGoldenFile(t, f.GoldenFile, actual)
	}
}

func TestNewMainTF(t *testing.T) {
	goldenFile := filepath.Join("testdata", "main.tf.golden")
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
	}
	input.Init()
	b := new(bytes.Buffer)
	err := NewMainTF(b, &input)
	require.NoError(t, err)
	checkGoldenFile(t, goldenFile, b.Bytes())
}

func checkGoldenFile(t *testing.T, goldenFile string, actual []byte) {
	// update golden files if necessary
	if *update {
		if err := ioutil.WriteFile(goldenFile, actual, 0644); err != nil {
			require.NoError(t, err)
		}
	}

	gld, err := ioutil.ReadFile(goldenFile)
	if err != nil {
		require.NoError(t, err)
	}

	assert.Equal(t, string(gld), string(actual))
}
