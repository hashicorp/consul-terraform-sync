package tftmpl

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
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
			Golden: "testdata/main.tf",
			Input: RootModuleInputData{
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
			},
		}, {
			Name:   "variables.tf",
			Func:   NewVariablesTF,
			Golden: "testdata/variables.tf",
			Input: RootModuleInputData{
				Providers: []hcltmpl.NamedBlock{hcltmpl.NewNamedBlock(
					map[string]interface{}{
						"testProvider": map[string]interface{}{
							"alias": "tp",
							"attr":  "value",
							"count": 10,
						},
					})},
			},
		}, {
			Name:   "terraform.tfvars.tmpl",
			Func:   NewTFVarsTmpl,
			Golden: "testdata/terraform.tfvars.tmpl",
			Input: RootModuleInputData{
				Providers: []hcltmpl.NamedBlock{hcltmpl.NewNamedBlock(
					map[string]interface{}{
						"testProvider": map[string]interface{}{
							"alias": "tp",
							"attr":  "value",
							"count": 10,
						},
					})},
				Services: []Service{
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
		}, {
			Name:   "variables.module.tf",
			Func:   NewModuleVariablesTF,
			Golden: "testdata/variables.module.tf",
			Input: RootModuleInputData{
				Variables: hcltmpl.Variables{
					"num": cty.NumberIntVal(10),
					"b":   cty.BoolVal(true),
					"key": cty.StringVal("some_key"),
					"obj": cty.ObjectVal(map[string]cty.Value{
						"argStr": cty.StringVal("value"),
						"argNum": cty.NumberFloatVal(-0.52),
						"argList": cty.ListVal([]cty.Value{
							cty.StringVal("a"),
							cty.StringVal("b"),
							cty.StringVal("c"),
						}),
						"argMap": cty.MapValEmpty(cty.Bool),
					}),
					"l": cty.ListVal([]cty.Value{
						cty.NumberIntVal(4),
						cty.NumberIntVal(0),
					}),
					"tup": cty.TupleVal([]cty.Value{
						cty.StringVal("abc"),
						cty.NumberIntVal(123),
						cty.BoolVal(false),
					}),
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
