// Code here is copied over from hcat's (https://github.com/hashicorp/hcat)
// internal dependency package that is used for creating custom template
// functions for CTS

package tmplfunc

import (
	"context"
	"sort"
	"time"

	consulapi "github.com/hashicorp/consul/api"
)

// QueryOptions is a list of options to send with the query. These options are
// client-agnostic, and the dependency determines which, if any, of the options
// to use.
type QueryOptions struct {
	AllowStale        bool
	Datacenter        string
	Filter            string
	Namespace         string
	Near              string
	RequireConsistent bool
	VaultGrace        time.Duration
	WaitIndex         uint64
	WaitTime          time.Duration
	DefaultLease      time.Duration

	ctx context.Context
}

func (q *QueryOptions) Merge(o *QueryOptions) *QueryOptions {
	var r QueryOptions

	if q == nil {
		if o == nil {
			return &QueryOptions{}
		}
		r = *o
		return &r
	}

	r = *q

	if o == nil {
		return &r
	}

	if o.AllowStale {
		r.AllowStale = o.AllowStale
	}

	if o.Datacenter != "" {
		r.Datacenter = o.Datacenter
	}

	if o.Filter != "" {
		r.Filter = o.Filter
	}

	if o.Namespace != "" {
		r.Namespace = o.Namespace
	}

	if o.Near != "" {
		r.Near = o.Near
	}

	if o.RequireConsistent {
		r.RequireConsistent = o.RequireConsistent
	}

	if o.WaitIndex != 0 {
		r.WaitIndex = o.WaitIndex
	}

	if o.WaitTime != 0 {
		r.WaitTime = o.WaitTime
	}

	return &r
}

func (q *QueryOptions) SetContext(ctx context.Context) QueryOptions {
	var q2 QueryOptions
	if q != nil {
		q2 = *q
	}
	q2.ctx = ctx
	return q2
}

func (q *QueryOptions) ToConsulOpts() *consulapi.QueryOptions {
	cq := consulapi.QueryOptions{
		AllowStale:        q.AllowStale,
		Datacenter:        q.Datacenter,
		Filter:            q.Filter,
		Namespace:         q.Namespace,
		Near:              q.Near,
		RequireConsistent: q.RequireConsistent,
		WaitIndex:         q.WaitIndex,
		WaitTime:          q.WaitTime,
	}

	if q.ctx != nil {
		return cq.WithContext(q.ctx)
	}
	return &cq
}

// deepCopyAndSortTags deep copies the tags in the given string slice and then
// sorts and returns the copied result.
func deepCopyAndSortTags(tags []string) []string {
	newTags := make([]string, 0, len(tags))
	newTags = append(newTags, tags...)
	sort.Strings(newTags)
	return newTags
}
