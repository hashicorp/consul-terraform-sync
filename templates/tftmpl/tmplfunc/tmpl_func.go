package tmplfunc

import (
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/hashicorp/hcat"
	"github.com/hashicorp/hcat/dep"
	"github.com/hashicorp/hcat/tfunc"
)

// HCLMap is the map of template functions for rendering HCL
// to their respective implementations
func HCLMap(meta map[string]map[string]string) template.FuncMap {
	tmplFuncs := hcat.FuncMapConsulV1()
	tmplFuncs["indent"] = tfunc.Helpers()["indent"]
	tmplFuncs["subtract"] = tfunc.Math()["subtract"]
	tmplFuncs["joinStrings"] = joinStringsFunc
	tmplFuncs["HCLService"] = hclServiceFunc(meta)
	// catalog-services condition
	tmplFuncs["regexMatch"] = tfunc.Helpers()["regexMatch"]
	tmplFuncs["HCLServiceTags"] = hclServiceTagsFunc()
	return tmplFuncs
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

// hclServiceTagsFunc is a wrapper of the template function to marshal Consul
// catalog service tag information into HCL. It returns the list of tags with
// formatted like: "["tag1", "tag2"]". It returns an empty array string "[]"
// when no tags
func hclServiceTagsFunc() func(tags *dep.ServiceTags) string {
	return func(tags *dep.ServiceTags) string {
		t := make([]string, len(*tags))
		for ix, tag := range *tags {
			t[ix] = fmt.Sprintf("\"%s\"", tag)
		}
		sort.Strings(t)

		return fmt.Sprintf("[%s]", strings.Join(t, ", "))
	}
}
