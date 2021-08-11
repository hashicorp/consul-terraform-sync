package hcltmpl

import (
	"context"
	"testing"
	"time"

	mocks "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zclconf/go-cty/cty"
)

func TestContainsDynamicTemplate(t *testing.T) {
	testCases := []struct {
		name          string
		s             string
		contains      bool
		containsVault bool
	}{
		{
			"empty",
			"",
			false,
			false,
		}, {}, {
			"some value",
			"my-argument-with-random {{",
			false,
			false,
		}, {
			"env",
			"{{ env \"ENV_NAME\" }}",
			true,
			false,
		}, {
			"consul kv",
			"{{ key \"foo/bar\" }}",
			true,
			false,
		}, {
			"vault secret",
			"{{ with secret \"foo/bar\" }}",
			true,
			true,
		}, {
			"no spacing",
			"{{with secret \"foo/bar\"}}",
			true,
			true,
		}, {
			"extra spacing",
			"{{		with secret     \"foo/bar\"   }}",
			true,
			true,
		}, {
			"substring",
			"substring {{ env \"ENV_NAME\"}} is valid",
			true,
			false,
		}, {
			"missing parameter quotes",
			"{{ key foo }}",
			false,
			false,
		}, {
			"missing parameter",
			"{{ key \"\" }}",
			false,
			false,
		}, {
			"vault missing with secret",
			"{{ secret \"foo/bar\" }}",
			false,
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			contains := ContainsDynamicTemplate(tc.s)
			assert.Equal(t, tc.contains, contains)

			containsVault := ContainsVaultSecret(tc.s)
			assert.Equal(t, tc.containsVault, containsVault)
		})
	}
}

func TestLoadDynamicConfig(t *testing.T) {
	tmplStr := "<rendered placeholder>"
	tmplVal := cty.StringVal(tmplStr)

	r := new(mocks.Resolver)
	r.On("Run", mock.Anything, mock.Anything).
		Return(hcat.ResolveEvent{Complete: true, Contents: []byte(tmplStr)}, nil)

	w := new(mocks.Watcher)
	w.On("WaitCh", mock.Anything, mock.Anything).Return(nil)
	w.On("Register", mock.Anything).Return(nil)

	testCases := []struct {
		name   string
		config map[string]interface{}
		block  NamedBlock
	}{
		{
			"empty",
			make(map[string]interface{}),
			NamedBlock{Variables: make(Variables)},
		}, {
			"basic",
			map[string]interface{}{
				"foo": map[string]interface{}{
					"attr": "value",
					"block": map[string]interface{}{
						"inner": 1,
					},
				},
			},
			NamedBlock{
				Name: "foo",
				Variables: map[string]cty.Value{
					"attr": cty.StringVal("value"),
					"block": cty.ObjectVal(map[string]cty.Value{
						"inner": cty.MustParseNumberVal("1"),
					}),
				},
			},
		}, {
			"dynamic",
			map[string]interface{}{
				"foo": map[string]interface{}{
					"attr":    "value",
					"dynamic": "{{ secret \"mysecret\" }}",
					"block": map[string]interface{}{
						"inner":          1,
						"dynamic_nested": "{{ key \"mykey\" }}",
					},
					"list": []interface{}{"{{ env \"MY_ENV\" }}", "item"},
				},
			},
			NamedBlock{
				Name: "foo",
				Variables: map[string]cty.Value{
					"attr":    cty.StringVal("value"),
					"dynamic": tmplVal,
					"block": cty.ObjectVal(map[string]cty.Value{
						"inner":          cty.MustParseNumberVal("1"),
						"dynamic_nested": tmplVal,
					}),
					"list": cty.TupleVal([]cty.Value{tmplVal, cty.StringVal("item")}),
				},
			},
		}, {
			"dynamic task_env",
			map[string]interface{}{
				"foo": map[string]interface{}{
					"task_env": map[string]interface{}{
						"FOO_TOKEN": "{{ env \"CTS_FOO_TOKEN\" }}",
					},
				},
			},
			NamedBlock{
				Name: "foo",
				Variables: map[string]cty.Value{
					"task_env": cty.ObjectVal(map[string]cty.Value{
						"FOO_TOKEN": tmplVal,
					}),
				},
			},
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			block, err := LoadDynamicConfig(ctxTimeout, w, r, tc.config)
			assert.NoError(t, err)
			assert.Len(t, block.Variables, len(tc.block.Variables))
			for k, v := range block.Variables {
				actual := block.Variables[k]
				equals := v.Equals(actual)
				assert.True(t, equals.True(), k)
			}
		})
	}
}
