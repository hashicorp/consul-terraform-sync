// Code here is copied over from hcat's (https://github.com/hashicorp/hcat)
// internal dependency package that is used for creating custom template
// functions for CTS

package tmplfunc

import (
	"sort"

	"github.com/hashicorp/hcat"
	"github.com/hashicorp/hcat/dep"
)

type hcatQuery interface {
	hcat.QueryOptionsSetter
	dep.Dependency
	Consul()
}

// isConsul satisfies the hcat dependency interface to denote Consul type for
// managing the Consul retry function.
type isConsul struct{}

func (isConsul) Consul() {}

// deepCopyAndSortTags deep copies the tags in the given string slice and then
// sorts and returns the copied result.
func deepCopyAndSortTags(tags []string) []string {
	newTags := make([]string, 0, len(tags))
	newTags = append(newTags, tags...)
	sort.Strings(newTags)
	return newTags
}
