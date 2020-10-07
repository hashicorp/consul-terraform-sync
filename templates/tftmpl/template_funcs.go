package tftmpl

import (
	"fmt"
	"sort"
	"strings"
)

// HCLTmplFuncMap are template functions for rendering HCL
var HCLTmplFuncMap = map[string]interface{}{
	"hclString":     HCLString,
	"hclStringList": HCLStringList,
	"hclStringMap":  HCLStringMap,
	"joinStrings":   JoinStrings,
}

// JoinStrings joins an optional number of strings with the separator while
// omitting empty strings from the combined string. This is useful for
// templating a number of strings that are not contained in a slice.
func JoinStrings(sep string, values ...string) string {
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

// HCLString formats a string into HCL string with null as the default
func HCLString(s string) string {
	if s == "" {
		return "null"
	}
	return fmt.Sprintf("%q", s)
}

// HCLStringList formats a list of strings into HCL
func HCLStringList(l []string) string {
	if len(l) == 0 {
		return "[]"
	}

	hclList := make([]string, len(l))
	for i, s := range l {
		hclList[i] = fmt.Sprintf("%q", s)
	}
	return fmt.Sprintf("[%s]", strings.Join(hclList, ", "))
}

// HCLStringMap formats a map of strings into HCL
func HCLStringMap(m map[string]string, indent int) string {
	if len(m) == 0 {
		return "{}"
	}

	sortedKeys := sortKeys(m)

	if indent < 1 {
		keyValues := make([]string, len(m))
		for i, k := range sortedKeys {
			v := m[k]
			keyValues[i] = fmt.Sprintf("%s = \"%s\"", k, v)
		}
		return fmt.Sprintf("{ %s }", strings.Join(keyValues, ", "))
	}

	// Find the longest key to align values with proper Terraform fmt spacing
	var longestKeyLen int
	for _, k := range sortedKeys {
		keyLen := len(k)
		if longestKeyLen < keyLen {
			longestKeyLen = keyLen
		}
	}

	indentStr := strings.Repeat("  ", indent)
	indentStrClosure := strings.Repeat("  ", indent-1)

	var keyValues string
	for _, k := range sortedKeys {
		v := m[k]
		tfFmtSpaces := strings.Repeat(" ", longestKeyLen-len(k))
		keyValues = fmt.Sprintf("%s\n%s%s%s = \"%s\"", keyValues, indentStr, k, tfFmtSpaces, v)
	}
	return fmt.Sprintf("{%s\n%s}", keyValues, indentStrClosure)
}

func sortKeys(m map[string]string) []string {
	sorted := make([]string, 0, len(m))
	for key := range m {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)
	return sorted
}
