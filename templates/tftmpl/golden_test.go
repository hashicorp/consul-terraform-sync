package tftmpl

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	goVersion "github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

var update = flag.Bool("update", false, "update golden files")

func TestNewFiles(t *testing.T) {
	task := Task{
		Description: "user description for task named 'test'",
		Name:        "test",
		Source:      "namespace/consul-terraform-sync/consul//modules/test",
		Version:     "0.0.0",
	}

	testCases := []struct {
		Name   string
		Func   func(io.Writer, string, *RootModuleInputData) error
		Golden string
		Input  RootModuleInputData
	}{
		{
			Name:   "main.tf",
			Func:   newMainTF,
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
				Task: task,
				Variables: hcltmpl.Variables{
					"one":       cty.NumberIntVal(1),
					"bool_true": cty.BoolVal(true),
				},
			},
		}, {
			Name:   "main.tf (catalog-services condition - source_includes_var)",
			Func:   newMainTF,
			Golden: "testdata/catalog-services-condition/main_include.tf",
			Input: RootModuleInputData{
				Backend: map[string]interface{}{},
				Condition: &CatalogServicesCondition{
					Regexp:            ".*",
					SourceIncludesVar: true,
				},
				Task: task,
			},
		}, {
			Name:   "variables.tf",
			Func:   newVariablesTF,
			Golden: "testdata/variables.tf",
			Input: RootModuleInputData{
				TerraformVersion: goVersion.Must(goVersion.NewSemver("0.99.9")),
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
				Task: task,
			},
		}, {
			Name:   "variables.tf (catalog-services condition - source_includes_var)",
			Func:   newVariablesTF,
			Golden: "testdata/catalog-services-condition/variables_include.tf",
			Input: RootModuleInputData{
				TerraformVersion: goVersion.Must(goVersion.NewSemver("0.99.9")),
				Condition: &CatalogServicesCondition{
					Regexp:            ".*",
					SourceIncludesVar: true,
				},
				Task: task,
			},
		}, {
			Name:   "terraform.tfvars.tmpl (services condition)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/terraform.tfvars.tmpl",
			Input: RootModuleInputData{
				Condition: &ServicesCondition{},
				Services: []Service{
					{
						Name:        "web",
						Description: "web service",
					}, {
						Name:        "api",
						Namespace:   "",
						Datacenter:  "dc1",
						Description: "api service for web",
						Tag:         "tag",
					},
				},
				Task: task,
			},
		}, {
			Name:   "terraform.tfvars.tmpl (catalog-services condition)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/catalog-services-condition/terraform.tfvars.tmpl",
			Input: RootModuleInputData{
				Condition: &CatalogServicesCondition{
					Regexp: ".*",
				},
				Services: []Service{
					{
						Name:        "web",
						Description: "web service",
					}, {
						Name:        "api",
						Namespace:   "",
						Datacenter:  "dc1",
						Description: "api service for web",
						Tag:         "tag",
					},
				},
				Task: task,
			},
		}, {
			Name:   "terraform.tfvars.tmpl (catalog-services condition - source_includes_var)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/catalog-services-condition/terraform_include.tfvars.tmpl",
			Input: RootModuleInputData{
				Condition: &CatalogServicesCondition{
					Regexp:            ".*",
					SourceIncludesVar: true,
				},
				Services: []Service{
					{
						Name:        "web",
						Description: "web service",
					}, {
						Name:        "api",
						Namespace:   "",
						Datacenter:  "dc1",
						Description: "api service for web",
						Tag:         "tag",
					},
				},
				Task: task,
			},
		}, {
			Name:   "terraform.tfvars.tmpl (catalog-services condition - filtering)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/catalog-services-condition/terraform_filter.tfvars.tmpl",
			Input: RootModuleInputData{
				Condition: &CatalogServicesCondition{
					Regexp:            ".*",
					SourceIncludesVar: true,
					Datacenter:        "dc1",
					NodeMeta:          map[string]string{"k": "v"},
				},
				Services: []Service{
					{
						Name:        "web",
						Description: "web service",
					}, {
						Name:        "api",
						Namespace:   "",
						Datacenter:  "dc1",
						Description: "api service for web",
						Tag:         "tag",
					},
				},
				Task: task,
			},
		}, {
			Name:   "terraform.tfvars.tmpl (catalog-services condition - no services)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/catalog-services-condition/terraform_no_services.tfvars.tmpl",
			Input: RootModuleInputData{
				Condition: &CatalogServicesCondition{
					Regexp: ".*",
				},
				Task: task,
			},
		}, {
			Name:   "providers.tfvars",
			Func:   newProvidersTFVars,
			Golden: "testdata/providers.tfvars",
			Input: RootModuleInputData{
				Providers: []hcltmpl.NamedBlock{hcltmpl.NewNamedBlock(
					map[string]interface{}{
						"testProvider": map[string]interface{}{
							"alias": "tp",
							"attr":  "value",
							"count": 10,
						},
					})},
				Task: task,
			},
		}, {
			Name:   "variables.module.tf",
			Func:   newModuleVariablesTF,
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
				Task: task,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			input := tc.Input
			input.Init()
			b := new(bytes.Buffer)
			err := tc.Func(b, tc.Name, &input)
			require.NoError(t, err)
			checkGoldenFile(t, tc.Golden, b.String())
		})
	}
}

func checkGoldenFile(t *testing.T, goldenFile string, actual string) {
	// update golden files if necessary
	if *update {
		if err := ioutil.WriteFile(goldenFile, []byte(actual), 0644); err != nil {
			require.NoError(t, err)
		}
	}

	gld := testutils.CheckFile(t, true, goldenFile, "")
	assert.Equal(t, gld, actual)
}
