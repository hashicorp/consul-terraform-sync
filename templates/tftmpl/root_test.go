package tftmpl

import (
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

func TestNewRootModule(t *testing.T) {
	goldenFile := filepath.Join("testdata", "root.tf.golden")
	input := RootModuleInputData{
		Backend: map[string]interface{}{
			"consul": map[string]interface{}{
				"scheme": "https",
				"path":   "consul-nia/terraform",
			},
		},
		Providers: []map[string]interface{}{{
			"null": map[string]interface{}{
				"alias": "dropped",
				"attr":  "value",
				"count": 10,
			},
		}},
		Task: Task{
			Description: "user description for task named 'test'",
			Name:        "test",
			Source:      "namespace/consul-nia/consul//modules/test",
			Version:     "0.0.0",
		},
	}
	f, err := NewRootModule(input)
	require.NoError(t, err)
	checkGoldenFile(t, goldenFile, f.Bytes())
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
