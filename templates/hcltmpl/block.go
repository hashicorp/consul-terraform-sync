// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hcltmpl

import (
	"sort"

	"github.com/hashicorp/consul-terraform-sync/internal/hcl2shim"
	"github.com/zclconf/go-cty/cty"
)

// Variables are the cty value representation of HCL block arguments
type Variables map[string]cty.Value

// NamedBlock represents an HCL block with one label and an arbitrary number of
// attributes of varying types.
//
//	block "name" {
//		attr = "str"
//		count = 10
//	}
type NamedBlock struct {
	Name      string
	Variables Variables

	blockKeysCache   []string
	objectTypeCache  *cty.Type
	objectValueCache *cty.Value
	rawConfig        map[string]interface{}
}

// Keys return a list of sorted variable names
func (v Variables) Keys() []string {
	sorted := make([]string, 0, len(v))
	for key := range v {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)
	return sorted
}

// NewNamedBlock converts a decoding of an HCL named block into a struct
// representation with cty types.
func NewNamedBlock(b map[string]interface{}) NamedBlock {
	// Remove one layer of the nesting to use for block label
	var name string
	var rawBlock map[string]interface{}
	var ok bool
	for k, v := range b {
		name = k
		rawBlock, ok = v.(map[string]interface{})
		if !ok {
			return NamedBlock{}
		}
		break
	}

	// Convert interface to usable cty.Value type
	vars := make(Variables, len(rawBlock))
	for k, v := range rawBlock {
		vars[k] = hcl2shim.HCL2ValueFromConfigValue(v)
	}

	return NamedBlock{
		Name:      name,
		Variables: vars,
		rawConfig: rawBlock,
	}
}

// Copy creates a copy of the NamedBlock with a shallow copy of the raw config
func (b *NamedBlock) Copy() NamedBlock {
	vars := make(map[string]cty.Value)
	for k, v := range b.Variables {
		vars[k] = v
	}

	return NamedBlock{
		Name:      b.Name,
		Variables: vars,
		rawConfig: b.rawConfig,
	}
}

// SortedAttributes returns a list of sorted attribute names
func (b *NamedBlock) SortedAttributes() []string {
	if b.blockKeysCache != nil {
		return b.blockKeysCache
	}

	sorted := make([]string, 0, len(b.Variables))
	for key := range b.Variables {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)
	b.blockKeysCache = sorted
	return sorted
}

// ObjectType converts the named block to an Object
func (b *NamedBlock) ObjectType() *cty.Type {
	if b.objectTypeCache != nil {
		return b.objectTypeCache
	}

	attrTypes := make(map[string]cty.Type, len(b.Variables))
	for k, v := range b.Variables {
		attrTypes[k] = v.Type()
	}

	obj := cty.Object(attrTypes)
	b.objectTypeCache = &obj
	return b.objectTypeCache
}

func (b *NamedBlock) ObjectVal() *cty.Value {
	if b.objectValueCache != nil {
		return b.objectValueCache
	}

	obj := cty.ObjectVal(b.Variables)
	b.objectValueCache = &obj
	return b.objectValueCache
}

func (b NamedBlock) RawConfig() map[string]interface{} {
	return b.rawConfig
}

// NewNamedBlocksTest is used to simplify testing
func NewNamedBlocksTest(rawBlocks []map[string]interface{}) []NamedBlock {
	blocks := make([]NamedBlock, len(rawBlocks))
	for i, b := range rawBlocks {
		blocks[i] = NewNamedBlockTest(b)
	}
	return blocks
}

// NewNamedBlockTest is used to simplify testing
func NewNamedBlockTest(b map[string]interface{}) NamedBlock {
	block := NewNamedBlock(b)
	block.rawConfig = nil
	return block
}
