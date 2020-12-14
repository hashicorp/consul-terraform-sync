package driver

import "github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"

type TerraformProviderBlock struct {
	block hcltmpl.NamedBlock
	env   map[string]string
}

type TerraformProviderBlocks []TerraformProviderBlock

func NewTerraformProviderBlocks(blocks []hcltmpl.NamedBlock) TerraformProviderBlocks {
	providers := make([]TerraformProviderBlock, len(blocks))
	for i, p := range blocks {
		providers[i] = NewTerraformProviderBlock(p)
	}
	return providers
}

func NewTerraformProviderBlock(b hcltmpl.NamedBlock) TerraformProviderBlock {
	env := make(map[string]string)

	copy := b.Copy()
	for k, v := range copy.Variables {
		if k == "task_env" && v.Type().IsObjectType() {
			for envKey, envVal := range v.AsValueMap() {
				env[envKey] = envVal.AsString()
			}
			delete(copy.Variables, k)
			break
		}
	}

	return TerraformProviderBlock{
		block: copy,
		env:   env,
	}
}

func (p TerraformProviderBlock) Name() string {
	return p.block.Name
}

func (p TerraformProviderBlock) ProviderBlock() hcltmpl.NamedBlock {
	return p.block
}

func (p TerraformProviderBlock) Env() map[string]string {
	return p.env
}

func (p TerraformProviderBlocks) ProviderBlocks() []hcltmpl.NamedBlock {
	blocks := make([]hcltmpl.NamedBlock, len(p))
	for i, b := range p {
		blocks[i] = b.ProviderBlock()
	}
	return blocks
}

func (p TerraformProviderBlocks) Env() map[string]string {
	env := make(map[string]string)
	for _, b := range p {
		for k, v := range b.Env() {
			env[k] = v
		}
	}
	return env
}
