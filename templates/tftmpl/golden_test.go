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
		Module:      "namespace/consul-terraform-sync/consul//modules/test",
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
						"version": "1.1.0",
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
			Name:   "main.tf (catalog-services - render var)",
			Func:   newMainTF,
			Golden: "testdata/catalog-services/main.tf",
			Input: RootModuleInputData{
				Backend: map[string]interface{}{},
				Templates: []Template{
					&CatalogServicesTemplate{
						Regexp:    ".*",
						RenderVar: true,
					},
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
			Name:   "variables.tf (catalog-services - render var)",
			Func:   newVariablesTF,
			Golden: "testdata/catalog-services/variables_with_var.tf",
			Input: RootModuleInputData{
				TerraformVersion: goVersion.Must(goVersion.NewSemver("0.99.9")),
				Templates: []Template{
					&CatalogServicesTemplate{
						Regexp:    ".*",
						RenderVar: true,
					},
				},
				Task: task,
			},
		}, {
			Name:   "variables.tf (condition consul-kv)",
			Func:   newVariablesTF,
			Golden: "testdata/consul-kv/variables.tf",
			Input: RootModuleInputData{
				Templates: []Template{
					&ConsulKVTemplate{
						Path:       "key-path",
						Datacenter: "dc1",
						RenderVar:  true,
					},
				},
				TerraformVersion: goVersion.Must(goVersion.NewSemver("0.99.9")),
				Task:             task,
			},
		}, {
			Name:   "variables.tf (consul-kv - render var)",
			Func:   newVariablesTF,
			Golden: "testdata/consul-kv/variables.tf",
			Input: RootModuleInputData{
				Templates: []Template{
					&ConsulKVTemplate{
						Path:       "key-path",
						Datacenter: "dc1",
						RenderVar:  true,
					},
				},
				TerraformVersion: goVersion.Must(goVersion.NewSemver("0.99.9")),
				Task:             task,
			},
		},
		{
			Name:   "terraform.tfvars.tmpl (services)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/services/terraform.tfvars.tmpl",
			Input: RootModuleInputData{
				Templates: []Template{
					&ServicesTemplate{
						Names:      []string{"web", "api"},
						Namespace:  "ns1",
						Datacenter: "dc1",
						Filter:     "\"tag\" in Service.Tags",
						RenderVar:  true,
					},
				},
				Task: task,
			},
		},
		{
			Name:   "terraform.tfvars.tmpl (services deprecated)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/terraform.tfvars.tmpl",
			Input: RootModuleInputData{
				Templates: []Template{
					&ServicesTemplate{
						Names: []string{"web", "api"},
						Services: map[string]Service{
							"web": {},
							"api": {
								Datacenter: "dc1",
								Filter:     "\"tag\" in Service.Tags",
							},
						},
						RenderVar: true,
					},
				},
				Task: task,
			},
		},
		{
			Name:   "terraform.tfvars.tmpl (services regex - render var)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/terraform_services_module_input.tmpl",
			Input: RootModuleInputData{
				Task: task,
				Templates: []Template{
					&ServicesRegexTemplate{
						Regexp:     ".*",
						Datacenter: "dc1",
						Namespace:  "ns1",
						Filter:     "some-filter",
						RenderVar:  true,
					},
				},
			},
		},
		{
			Name:   "terraform.tfvars.tmpl (catalog-services - no var)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/catalog-services/terraform.tfvars.tmpl",
			Input: RootModuleInputData{
				Templates: []Template{
					&CatalogServicesTemplate{
						Regexp:    ".*",
						RenderVar: false,
					},
					&ServicesTemplate{
						Names: []string{"web", "api"},
						Services: map[string]Service{
							"web": {},
							"api": {
								Datacenter: "dc1",
								Filter:     "\"tag\" in Service.Tags",
							},
						},
						RenderVar: true,
					},
				},
				Task: task,
			},
		}, {
			Name:   "terraform.tfvars.tmpl (catalog-services - render var)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/catalog-services/terraform_with_var.tfvars.tmpl",
			Input: RootModuleInputData{
				Templates: []Template{
					&CatalogServicesTemplate{
						Regexp:    "^web.*|^api.*",
						RenderVar: true,
					},
					&ServicesTemplate{
						Names: []string{"web", "api"},
						Services: map[string]Service{
							"web": {},
							"api": {
								Datacenter: "dc1",
								Filter:     "\"tag\" in Service.Tags",
							},
						},
						RenderVar: true,
					},
				},
				Task: task,
			},
		}, {
			Name:   "terraform.tfvars.tmpl (catalog-services w filtering - render var)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/catalog-services/terraform_filter.tfvars.tmpl",
			Input: RootModuleInputData{
				Templates: []Template{
					&CatalogServicesTemplate{
						Regexp:     "^web.*|^api.*",
						Datacenter: "dc1",
						NodeMeta:   map[string]string{"k": "v"},
						RenderVar:  true,
					},
					&ServicesTemplate{
						Names: []string{"web", "api"},
						Services: map[string]Service{
							"web": {},
							"api": {
								Datacenter: "dc1",
								Filter:     "\"tag\" in Service.Tags",
							},
						},
						RenderVar: true,
					},
				},
				Task: task,
			},
		}, {
			Name:   "terraform.tfvars.tmpl (catalog-services & no services)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/catalog-services/terraform_no_services.tfvars.tmpl",
			Input: RootModuleInputData{
				Templates: []Template{
					&CatalogServicesTemplate{
						Regexp:    ".*",
						RenderVar: false,
					},
				},
				Task: task,
			},
		}, {
			Name:   "terraform.tfvars.tmpl (consul-kv w no namespace - no var)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/consul-kv/terraform.tfvars.tmpl",
			Input: RootModuleInputData{
				Templates: []Template{
					&ConsulKVTemplate{
						Path:       "key-path",
						Datacenter: "dc1",
						RenderVar:  false,
					},
					&ServicesTemplate{
						Names: []string{"web", "api"},
						Services: map[string]Service{
							"web": {},
							"api": {
								Datacenter: "dc1",
								Filter:     "\"tag\" in Service.Tags",
							},
						},
						RenderVar: true,
					},
				},
				Task: task,
			},
		}, {
			Name:   "terraform.tfvars.tmpl (consul-kv - no var)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/consul-kv/terraform_namespace.tfvars.tmpl",
			Input: RootModuleInputData{
				Templates: []Template{
					&ConsulKVTemplate{
						Path:       "key-path",
						Datacenter: "dc1",
						Namespace:  "test-ns",
						RenderVar:  false,
					},
					&ServicesTemplate{
						Names: []string{"web", "api"},
						Services: map[string]Service{
							"web": {},
							"api": {
								Datacenter: "dc1",
								Filter:     "\"tag\" in Service.Tags",
							},
						},
						RenderVar: true,
					},
				},
				Task: task,
			},
		}, {
			Name:   "terraform.tfvars.tmpl (consul-kv - render var)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/consul-kv/terraform_with_var.tfvars.tmpl",
			Input: RootModuleInputData{
				Templates: []Template{
					&ConsulKVTemplate{
						Path:       "key-path",
						Datacenter: "dc1",
						RenderVar:  true,
					},
					&ServicesTemplate{
						Names: []string{"web", "api"},
						Services: map[string]Service{
							"web": {},
							"api": {
								Datacenter: "dc1",
								Filter:     "\"tag\" in Service.Tags",
							},
						},
						RenderVar: true,
					},
				},
				Task: task,
			},
		}, {
			Name:   "terraform.tfvars.tmpl (consul-kv w recurse - render var)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/consul-kv/terraform_recurse_true.tfvars.tmpl",
			Input: RootModuleInputData{
				Templates: []Template{
					&ConsulKVTemplate{
						Path:       "key-path",
						Datacenter: "dc1",
						Recurse:    true,
						RenderVar:  true,
					},
					&ServicesTemplate{
						Names: []string{"web", "api"},
						Services: map[string]Service{
							"web": {},
							"api": {
								Datacenter: "dc1",
								Filter:     "\"tag\" in Service.Tags",
							},
						},
						RenderVar: true,
					},
				},
				Task: task,
			},
		}, {
			Name:   "terraform.tfvars.tmpl (consul-kv w recurse - no var)",
			Func:   newTFVarsTmpl,
			Golden: "testdata/consul-kv/terraform_recurse_true_no_var.tfvars.tmpl",
			Input: RootModuleInputData{
				Templates: []Template{
					&ConsulKVTemplate{
						Path:       "key-path",
						Datacenter: "dc1",
						Recurse:    true,
						RenderVar:  false,
					},
					&ServicesTemplate{
						Names: []string{"web", "api"},
						Services: map[string]Service{
							"web": {},
							"api": {
								Datacenter: "dc1",
								Filter:     "\"tag\" in Service.Tags",
							},
						},
						RenderVar: true,
					},
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
		}, {
			Name:   "variables_detailed.auto.tfvars",
			Func:   newVariablesTFVars,
			Golden: "testdata/variables.auto.tfvars",
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
			input.init()
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
