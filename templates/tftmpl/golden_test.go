package tftmpl

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

func TestNewFiles(t *testing.T) {
	testCases := []struct {
		Name   string
		Func   func(io.Writer, *RootModuleInputData) error
		Golden string
		Input  RootModuleInputData
	}{
		{
			Name:   "main.tf",
			Func:   NewMainTF,
			Golden: "testdata/main.tf.golden",
			Input: RootModuleInputData{
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
			},
		}, {
			Name:   "variables.tf",
			Func:   NewVariablesTF,
			Golden: "testdata/variables.tf.golden",
			Input: RootModuleInputData{
				Providers: []map[string]interface{}{{
					"testProvider": map[string]interface{}{
						"alias": "tp",
						"attr":  "value",
						"count": 10,
					},
				}},
			},
		}, {
			Name:   "terraform.tfvars.tmpl",
			Func:   NewTFVarsTmpl,
			Golden: "testdata/terraform.tfvars.tmpl.golden",
			Input: RootModuleInputData{
				Providers: []map[string]interface{}{{
					"testProvider": map[string]interface{}{
						"alias": "tp",
						"attr":  "value",
						"count": 10,
					},
				}},
				Services: []*Service{
					{
						Name:        "web",
						Namespace:   "ns",
						Datacenter:  "dc1",
						Description: "web service",
					}, {
						Name:        "api",
						Namespace:   "",
						Datacenter:  "dc1",
						Description: "api service for web",
						Tag:         "tag",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			input := tc.Input
			input.Init()
			b := new(bytes.Buffer)
			err := tc.Func(b, &input)
			require.NoError(t, err)
			checkGoldenFile(t, tc.Golden, b.Bytes())
		})
	}
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
