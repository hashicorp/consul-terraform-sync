package tftmpl

import (
	"strings"

	"github.com/hashicorp/hcat/dep"
	"github.com/hashicorp/hcat/tfunc"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// HCLTmplFuncMap are template functions for rendering HCL
var HCLTmplFuncMap = map[string]interface{}{
	"indent":   tfunc.Helpers()["indent"],
	"subtract": tfunc.Math()["subtract"],

	"joinStrings": joinStringsFunc,
	"HCLService":  hclServiceFunc,
}

// JoinStrings joins an optional number of strings with the separator while
// omitting empty strings from the combined string. This is useful for
// templating a number of strings that are not contained in a slice.
func joinStringsFunc(sep string, values ...string) string {
	if len(values) == 0 {
		return ""
	}

	cleaned := make([]string, 0, len(values))
	for _, v := range values {
		if v != "" {
			cleaned = append(cleaned, v)
		}
	}

	return strings.Join(cleaned, sep)
}

func hclServiceFunc(sDep *dep.HealthService) string {
	if sDep == nil {
		return ""
	}

	s := newHealthService(sDep)

	f := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(s, f.Body())
	return strings.TrimSpace(string(f.Bytes()))
}
