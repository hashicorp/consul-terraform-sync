package tftmpl

import (
	"sort"

	"github.com/hashicorp/terraform/configs/hcl2shim"
	"github.com/zclconf/go-cty/cty"
)

// namedBlock represents an HCL block with one label and an arbitrary number of
// attributes of varying types.
//
// 	block "name" {
//		attr = "str"
//		count = 10
// 	}
type namedBlock struct {
	Name  string
	Block Variables

	blockKeysCache   []string
	objectTypeCache  *cty.Type
	objectValueCache *cty.Value
}

func newNamedBlock(b map[string]interface{}) *namedBlock {
	// Remove one layer of the nesting to use for block label
	var name string
	var rawBlock map[string]interface{}
	var ok bool
	for k, v := range b {
		name = k
		rawBlock, ok = v.(map[string]interface{})
		if !ok {
			return nil
		}
		break
	}

	// Convert interface to usable cty.Value type
	block := make(Variables, len(rawBlock))
	for k, v := range rawBlock {
		block[k] = hcl2shim.HCL2ValueFromConfigValue(v)
	}

	return &namedBlock{
		Name:  name,
		Block: block,
	}
}

// SortedAttributes returns a list of sorted attribute names
func (b *namedBlock) SortedAttributes() []string {
	if b.blockKeysCache != nil {
		return b.blockKeysCache
	}

	sorted := make([]string, 0, len(b.Block))
	for key := range b.Block {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)
	b.blockKeysCache = sorted
	return sorted
}

// ObjectType converts the named block to an Object
func (b *namedBlock) ObjectType() *cty.Type {
	if b.objectTypeCache != nil {
		return b.objectTypeCache
	}

	attrTypes := make(map[string]cty.Type, len(b.Block))
	for k, v := range b.Block {
		attrTypes[k] = v.Type()
	}

	obj := cty.Object(attrTypes)
	b.objectTypeCache = &obj
	return b.objectTypeCache
}

func (b *namedBlock) ObjectVal() *cty.Value {
	if b.objectValueCache != nil {
		return b.objectValueCache
	}

	obj := cty.ObjectVal(b.Block)
	b.objectValueCache = &obj
	return b.objectValueCache
}
