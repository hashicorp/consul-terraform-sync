package driver

import (
	"testing"

	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

func TestTerraformProviderBlocks_NewTerraformProviderBlocks(t *testing.T) {
	namedBlock := hcltmpl.NewNamedBlocksTest([]map[string]interface{}{
		{"local": map[string]interface{}{
			"configs": "stuff",
		}},
		{"null": map[string]interface{}{}},
		{"providerA": map[string]interface{}{
			"task_env": map[string]interface{}{
				"PROVIDER_TOKEN": "TEST_PROVIDER_TOKEN",
			},
		}},
	})

	expectedTerraformProviderBlocks := TerraformProviderBlocks{
		{
			block: namedBlock[0],
			env:   make(map[string]string),
		},
		{
			block: namedBlock[1],
			env:   make(map[string]string),
		},
		{
			block: namedBlock[2].Copy(), // Use a copy because we are going to delete a variable
			env: map[string]string{
				"PROVIDER_TOKEN": "TEST_PROVIDER_TOKEN",
			},
		},
	}
	delete(expectedTerraformProviderBlocks[2].block.Variables, "task_env")

	providerBlocks := NewTerraformProviderBlocks(namedBlock)
	assert.ElementsMatch(t, expectedTerraformProviderBlocks, providerBlocks)
}

func TestTerraformProviderBlock_Copy(t *testing.T) {
	cases := []struct {
		name          string
		providerBlock TerraformProviderBlock
	}{
		{
			name: "happy path",
			providerBlock: TerraformProviderBlock{
				block: hcltmpl.NamedBlock{
					Name: "local",
					Variables: hcltmpl.Variables{
						"attr": cty.StringVal("value"),
					},
				},
				env: make(map[string]string),
			},
		},
		{
			name: "with environment set",
			providerBlock: TerraformProviderBlock{
				block: hcltmpl.NamedBlock{
					Name: "local",
					Variables: hcltmpl.Variables{
						"attr": cty.StringVal("value"),
					},
				},
				env: map[string]string{
					"TOKEN": "TOKEN_VALUE",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cp := tc.providerBlock.Copy()

			// Test copies is equal to original
			assert.Equal(t, tc.providerBlock, cp)

			// Modify copy and assert that the original is now not equal
			delete(cp.block.Variables, "attr")
			assert.NotEqual(t, tc.providerBlock, cp)
		})
	}
}

func TestTerraformProviderBlock_Name(t *testing.T) {
	expectedName := "local"
	tpb := TerraformProviderBlock{
		block: hcltmpl.NamedBlock{
			Name: expectedName,
			Variables: hcltmpl.Variables{
				"attr": cty.StringVal("value"),
			},
		},
		env: make(map[string]string),
	}

	assert.Equal(t, expectedName, tpb.Name())
}

func TestTerraformProviderBlock_ProviderBlock(t *testing.T) {
	expectedName := "local"
	tpb := TerraformProviderBlock{
		block: hcltmpl.NamedBlock{
			Name: expectedName,
			Variables: hcltmpl.Variables{
				"attr": cty.StringVal("value"),
			},
		},
		env: make(map[string]string),
	}

	assert.Equal(t, tpb.block, tpb.ProviderBlock())
}

func TestTerraformProviderBlock_Env(t *testing.T) {
	expectedName := "local"
	tpb := TerraformProviderBlock{
		block: hcltmpl.NamedBlock{
			Name: expectedName,
			Variables: hcltmpl.Variables{
				"attr": cty.StringVal("value"),
			},
		},
		env: map[string]string{
			"TOKEN": "TOKEN_VALUE",
		},
	}

	assert.Equal(t, tpb.env, tpb.Env())
}

func TestTerraformProviderBlocks_ProviderBlocks(t *testing.T) {
	expectedName := "local"
	tpbs := TerraformProviderBlocks{
		{
			block: hcltmpl.NamedBlock{
				Name: expectedName,
				Variables: hcltmpl.Variables{
					"attr": cty.StringVal("value"),
				},
			},
			env: make(map[string]string),
		},
		{
			block: hcltmpl.NamedBlock{
				Name: expectedName,
				Variables: hcltmpl.Variables{
					"attr": cty.StringVal("value2"),
				},
			},
			env: make(map[string]string),
		},
	}

	providerBlocks := tpbs.ProviderBlocks()
	assert.Equal(t, tpbs[0].block, providerBlocks[0])
	assert.Equal(t, tpbs[1].block, providerBlocks[1])
}

func TestTerraformProviderBlocks_Env(t *testing.T) {
	expectedEnv := map[string]string{
		"TOKEN":  "TOKEN_VALUE",
		"TOKEN2": "TOKEN_VALUE2",
	}
	tpbs := TerraformProviderBlocks{
		{
			block: hcltmpl.NamedBlock{
				Name: "local",
				Variables: hcltmpl.Variables{
					"attr": cty.StringVal("value"),
				},
			},
			env: map[string]string{
				"TOKEN": "TOKEN_VALUE",
			},
		},
		{
			block: hcltmpl.NamedBlock{
				Name: "local",
				Variables: hcltmpl.Variables{
					"attr": cty.StringVal("value"),
				},
			},
			env: map[string]string{
				"TOKEN2": "TOKEN_VALUE2",
			},
		},
	}

	actualEnv := tpbs.Env()
	assert.Equal(t, len(tpbs), len(expectedEnv))
	for k, v := range expectedEnv {
		val, ok := actualEnv[k]
		assert.True(t, ok)
		assert.Equal(t, v, val)
	}
}

func TestTerraformProviderBlocks_Copy(t *testing.T) {
	cases := []struct {
		name           string
		providerBlocks TerraformProviderBlocks
	}{
		{
			name: "happy path",
			providerBlocks: TerraformProviderBlocks{
				{
					block: hcltmpl.NamedBlock{
						Name: "local",
						Variables: hcltmpl.Variables{
							"attr": cty.StringVal("value"),
						},
					},
					env: make(map[string]string),
				},
				{
					block: hcltmpl.NamedBlock{
						Name: "local",
						Variables: hcltmpl.Variables{
							"attr": cty.StringVal("value2"),
						},
					},
					env: make(map[string]string),
				},
			},
		},
		{
			name: "with environment set",
			providerBlocks: TerraformProviderBlocks{
				{
					block: hcltmpl.NamedBlock{
						Name: "local",
						Variables: hcltmpl.Variables{
							"attr": cty.StringVal("value"),
						},
					},
					env: map[string]string{
						"TOKEN": "TOKEN_VALUE",
					},
				},
				{
					block: hcltmpl.NamedBlock{
						Name: "local",
						Variables: hcltmpl.Variables{
							"attr": cty.StringVal("value"),
						},
					},
					env: map[string]string{
						"TOKEN": "TOKEN_VALUE2",
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cp := tc.providerBlocks.Copy()

			// Test copies is equal to original
			assert.ElementsMatch(t, tc.providerBlocks, cp)

			// Modify copy and assert that the original is now not equal
			delete(cp[0].block.Variables, "attr")
			assert.NotEqual(t, tc.providerBlocks[0], cp)
		})
	}
}
