package tmplfunc

import (
	"fmt"
	"sort"
	"strings"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-bexpr"
	"github.com/hashicorp/hcat"
	"github.com/hashicorp/hcat/dep"
	"github.com/pkg/errors"
)

func nodesFunc(recall hcat.Recaller) interface{} {
	return func(opts ...string) ([]*dep.Node, error) {
		result := []*dep.Node{}

		d, err := newNodesQuery(opts)
		if err != nil {
			return nil, err
		}

		if value, ok := recall(d); ok {
			return value.([]*dep.Node), nil
		}

		return result, nil
	}
}

type nodesQuery struct {
	stopCh chan struct{}

	dc     string
	filter string
	opts   consulapi.QueryOptions
}

func newNodesQuery(opts []string) (*nodesQuery, error) {
	nodesQuery := nodesQuery{
		stopCh: make(chan struct{}, 1),
	}
	var filters []string
	for _, opt := range opts {
		if strings.TrimSpace(opt) == "" {
			continue
		}

		if queryParamOptRe.MatchString(opt) {
			queryParam := strings.SplitN(opt, "=", 2)
			query := strings.TrimSpace(queryParam[0])
			value := strings.TrimSpace(queryParam[1])
			switch query {
			case "dc", "datacenter":
				nodesQuery.dc = value
				continue
			}
		}

		_, err := bexpr.CreateFilter(opt)
		if err != nil {
			return nil, fmt.Errorf("nodes: invalid filter: %q: %s", opt, err)
		}
		filters = append(filters, opt)
	}

	if len(filters) > 0 {
		nodesQuery.filter = strings.Join(filters, " and ")
	}

	return &nodesQuery, nil
}

func (d *nodesQuery) Fetch(clients dep.Clients) (interface{}, *dep.ResponseMetadata, error) {
	select {
	case <-d.stopCh:
		return nil, nil, dep.ErrStopped
	default:
	}

	opts := d.opts
	if d.dc != "" {
		opts.Datacenter = d.dc
	}
	if d.filter != "" {
		opts.Filter = d.filter
	}
	catalog, qm, err := clients.Consul().Catalog().Nodes(&opts)
	if err != nil {
		return nil, nil, errors.Wrap(err, d.String())
	}

	var nodes []*dep.Node
	for _, n := range catalog {
		nodes = append(nodes, &dep.Node{
			ID:              n.ID,
			Node:            n.Node,
			Address:         n.Address,
			Datacenter:      n.Datacenter,
			TaggedAddresses: n.TaggedAddresses,
			Meta:            n.Meta,
		})
	}

	sort.Stable(ByID(nodes))

	rm := &dep.ResponseMetadata{
		LastIndex:   qm.LastIndex,
		LastContact: qm.LastContact,
	}
	d.opts.WaitIndex = qm.LastIndex

	return nodes, rm, nil
}

func (d *nodesQuery) String() string {
	var opts []string

	if d.dc != "" {
		opts = append(opts, fmt.Sprintf("dc=%s", d.dc))
	}

	sort.Strings(opts)
	return fmt.Sprintf("node(%s)", strings.Join(opts, "&"))
}

func (d *nodesQuery) Stop() {
	close(d.stopCh)
}

// ByID is a sortable slice of Node
type ByID []*dep.Node

// Len, Swap, and Less are used to implement the sort.Sort interface.
func (s ByID) Len() int      { return len(s) }
func (s ByID) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s ByID) Less(i, j int) bool {
	return s[i].ID < s[j].ID
}
