package driver

import "github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"

// TerraformProviderBlock contains provider arguments and environment variables
// for the Terraform provider.
type TerraformProviderBlock struct {
	block hcltmpl.NamedBlock
	env   map[string]string
}

// TerraformProviderBlocks are a list of providers and their arguments and env
type TerraformProviderBlocks []TerraformProviderBlock

// NewTerraformProviderBlocks creates a new list of provider blocks with the
// environment variables separated from provider arguments from the base hcl
// block.
func NewTerraformProviderBlocks(blocks []hcltmpl.NamedBlock) TerraformProviderBlocks {
	providers := make([]TerraformProviderBlock, len(blocks))
	for i, p := range blocks {
		providers[i] = NewTerraformProviderBlock(p)
	}
	return providers
}

// NewTerraformProviderBlock creates a provider block with the environment
// variables separated from provider arguments from the base hcl block.
func NewTerraformProviderBlock(b hcltmpl.NamedBlock) TerraformProviderBlock {
	env := make(map[string]string)

	cp := b.Copy()
	for k, v := range cp.Variables {
		if k == "task_env" && v.Type().IsObjectType() {
			for envKey, envVal := range v.AsValueMap() {
				env[envKey] = envVal.AsString()
			}
			delete(cp.Variables, k)
			break
		}
	}

	return TerraformProviderBlock{
		block: cp,
		env:   env,
	}
}

func (p TerraformProviderBlock) Copy() TerraformProviderBlock {
	env := make(map[string]string)
	for k, v := range p.env {
		env[k] = v
	}
	return TerraformProviderBlock{
		block: p.block.Copy(),
		env:   env,
	}
}

// Name returns the name of the provider. This is the label of the HCL named
// block.
func (p TerraformProviderBlock) Name() string {
	return p.block.Name
}

// ProviderBlock returns the arguments for the Terraform provider block.
func (p TerraformProviderBlock) ProviderBlock() hcltmpl.NamedBlock {
	return p.block
}

// Env returns the configured environment variables for the Terraform provider.
// These values are set for the task workspace and are not written to any
// generated Terraform configuration file.
func (p TerraformProviderBlock) Env() map[string]string {
	return p.env
}

// ProviderBlocks returns a list of the provider blocks.
func (p TerraformProviderBlocks) ProviderBlocks() []hcltmpl.NamedBlock {
	blocks := make([]hcltmpl.NamedBlock, len(p))
	for i, b := range p {
		blocks[i] = b.ProviderBlock()
	}
	return blocks
}

// Env returns a merged map of environment variables across all providers.
func (p TerraformProviderBlocks) Env() map[string]string {
	env := make(map[string]string)
	for _, b := range p {
		for k, v := range b.Env() {
			env[k] = v
		}
	}
	return env
}

func (p TerraformProviderBlocks) Copy() TerraformProviderBlocks {
	cp := make(TerraformProviderBlocks, len(p))
	for k, v := range p {
		cp[k] = v.Copy()
	}
	return cp
}
