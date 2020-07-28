package tftmpl

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTFVarsTmpl(t *testing.T) {
	goldenFile := filepath.Join("testdata", "terraform.tfvars.tmpl.golden")
	input := RootModuleInputData{
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
	}
	input.Init()
	b := new(bytes.Buffer)
	err := NewTFVarsTmpl(b, &input)
	require.NoError(t, err)
	checkGoldenFile(t, goldenFile, b.Bytes())
}
