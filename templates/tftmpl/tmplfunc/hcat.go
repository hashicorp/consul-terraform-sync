// Code here is copied over from hcat's (https://github.com/hashicorp/hcat)
// internal dependency package that is used for creating custom template
// functions for CTS

package tmplfunc

import (
	"sort"
)

// deepCopyAndSortTags deep copies the tags in the given string slice and then
// sorts and returns the copied result.
func deepCopyAndSortTags(tags []string) []string {
	newTags := make([]string, 0, len(tags))
	newTags = append(newTags, tags...)
	sort.Strings(newTags)
	return newTags
}
