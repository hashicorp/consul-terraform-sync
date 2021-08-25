package hcltmpl

import (
	"context"
	"fmt"
	"log"
	"regexp"

	tmpls "github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/hcat"
	"github.com/hashicorp/hcat/tfunc"
	"github.com/zclconf/go-cty/cty"
)

var (
	dynamicTmplRegexp = regexp.MustCompile(`\{\{\s*(env|key|with secret)\s+\\?\".+\\?\"\s*\}\}`)
	vaultTmplRegexp   = regexp.MustCompile(`\{\{\s*with secret\s+\\?\".+\\?\"\s*\}\}`)
)

// ContainsDynamicTemplate reports whether the template syntax supported by CTS
// to load from env, Consul KV, and Vault is within s.
func ContainsDynamicTemplate(s string) bool {
	return dynamicTmplRegexp.MatchString(s)
}

// ContainsVaultSecret reports whether the template syntax to fetch secrets
// from Vault is within s.
func ContainsVaultSecret(s string) bool {
	return vaultTmplRegexp.MatchString(s)
}

// LoadDynamicConfig converts a decoding of HCL named blocks into a struct
// representation with cty types and fetches dynamic values with templated
// configuration.
func LoadDynamicConfig(ctx context.Context, w tmpls.Watcher, r tmpls.Resolver,
	config map[string]interface{}) (NamedBlock, error) {
	block := NewNamedBlock(config)

	// First pass, check if the block has any templated variables before continuing
	// with slower processing
	if !ContainsDynamicTemplate(fmt.Sprint(config)) {
		return block, nil
	}

	log.Printf("[INFO] (templates.hcltmpl) evaluating dynamic configuration for %q", block.Name)

	// Traverse all variables and nested variables to evaluate any dynamic values
	for attrName, v := range block.Variables {
		value, err := dynamicValue(ctx, w, r, v)
		if err != nil {
			return block, err
		}
		block.Variables[attrName] = value
	}

	return block, nil
}

func dynamicValue(ctx context.Context, w tmpls.Watcher, r tmpls.Resolver, v cty.Value) (cty.Value, error) {
	select {
	case <-ctx.Done():
		return cty.Value{}, ctx.Err()
	default:
	}

	// Match regex {{ [env|key|secret] ".*" }} to check whether the value
	// contains template syntax to be evaluated.
	if !ContainsDynamicTemplate(v.GoString()) {
		// Value is not dynamic, return early
		return v, nil
	}

	// Recursively check nested values of collection types for templates.
	t := v.Type()
	switch {
	case t.IsPrimitiveType() && t == cty.String:
		// String types that have gotten to this point contain supported
		// template(s) to be rendered.
		//
		// Render the template to evaluate the dynamic value and return
		// out of recursion.
		tmpl := hcat.NewTemplate(hcat.TemplateInput{
			Contents:     v.AsString(),
			FuncMapMerge: tfunc.Env(),
		})
		w.Register(tmpl)
		rendered, err := renderDynamicValue(ctx, w, r, tmpl)
		if err != nil {
			return cty.Value{}, err
		}
		return cty.StringVal(rendered), nil

	case t.IsListType(), t.IsTupleType():
		values := v.AsValueSlice()
		for i, value := range values {
			dValue, err := dynamicValue(ctx, w, r, value)
			if err != nil {
				return cty.Value{}, err
			}
			values[i] = dValue
		}
		return cty.TupleVal(values), nil

	case t.IsMapType(), t.IsObjectType():
		values := v.AsValueMap()
		for attrName, value := range values {
			dValue, err := dynamicValue(ctx, w, r, value)
			if err != nil {
				return cty.Value{}, err
			}
			values[attrName] = dValue
		}

		if t.IsMapType() {
			return cty.MapVal(values), nil
		}
		return cty.ObjectVal(values), nil
	}

	return v, nil
}

func renderDynamicValue(ctx context.Context, w tmpls.Watcher,
	r tmpls.Resolver, tmpl tmpls.Template) (string, error) {
	for {
		re, err := r.Run(tmpl, w)
		if err != nil {
			return "", err
		}
		if re.Complete {
			return string(re.Contents), nil
		}

		select {
		case err = <-w.WaitCh(ctx):
			if err != nil {
				return "", err
			}
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}
