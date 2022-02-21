package tmplfunc

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-bexpr"
	"github.com/hashicorp/hcat"
	"github.com/hashicorp/hcat/dep"
	"github.com/pkg/errors"
)

var (
	// queryParamOptRe is the regular expression to distinguish between query
	// params and filters, excluding the regex parameter. Non-regex query parameters
	// only have one "=" where as filters can have "==" or "!=" operators.
	queryParamOptRe = regexp.MustCompile(`[\w\d\s]=[\w\d\s]`)

	_ hcatQuery = (*servicesRegexQuery)(nil)
)

// servicesRegexFunc returns information on registered Consul
// services that have a name that match a given regex. It queries
// the Catalog List Services API initially to get all the services
// and then queries the Health API for each matching service.
// It supports parameters filter, dc, ns, and node-meta on the
// Health API query only.
//
// Endpoints:
//   /v1/catalog/services
//   /v1/health/service/:service
// Template: {{ servicesRegex regexp=<regex> <options> ... }}
func servicesRegexFunc(recall hcat.Recaller) interface{} {
	return func(opts ...string) ([]*dep.HealthService, error) {
		result := []*dep.HealthService{}

		d, err := newServicesRegexQuery(opts)
		if err != nil {
			return nil, err
		}

		if value, ok := recall(d); ok {
			return value.([]*dep.HealthService), nil
		}

		return result, nil
	}
}

// servicesRegexQuery is the representation of the regex service
// query from inside a template.
type servicesRegexQuery struct {
	isConsul
	stopCh chan struct{}

	regexp *regexp.Regexp

	filter   string
	dc       string
	ns       string
	nodeMeta map[string]string
	opts     hcat.QueryOptions
}

// newServicesRegexQuery processes options in the format of
// "key=value" (e.g. "regexp=^web.*") with the exception of filters.
// Any option that is not a key/value pair is assumed to be a filter.
func newServicesRegexQuery(opts []string) (*servicesRegexQuery, error) {
	servicesRegexQuery := servicesRegexQuery{
		stopCh: make(chan struct{}, 1),
	}
	var filters []string
	for _, opt := range opts {
		if strings.TrimSpace(opt) == "" {
			continue
		}

		// Parse query paramters, excluding the filter which is not set as a parameter
		if queryParamOptRe.MatchString(opt) || strings.Contains(opt, "regexp=") {
			queryParam := strings.SplitN(opt, "=", 2)
			query := strings.TrimSpace(queryParam[0])
			value := strings.TrimSpace(queryParam[1])
			switch query {
			case "regexp":
				r, err := regexp.Compile(value)
				if err != nil {
					return nil, fmt.Errorf("service.regex: invalid regexp")
				}
				servicesRegexQuery.regexp = r
				continue
			case "dc", "datacenter":
				servicesRegexQuery.dc = value
				continue
			case "ns", "namespace":
				servicesRegexQuery.ns = value
				continue
			case "node-meta":
				if servicesRegexQuery.nodeMeta == nil {
					servicesRegexQuery.nodeMeta = make(map[string]string)
				}
				k, v, err := stringsSplit2(value, ":")
				if err != nil {
					return nil, fmt.Errorf(
						"service.regex: invalid format for query "+
							"parameter %q: %s", query, value)
				}
				servicesRegexQuery.nodeMeta[k] = v
				continue
			}
		}

		// Any option that was not already parsed is assumed to be a filter.
		// Evaluate the grammer of the filter before attempting to query Consul.
		// Defer to the Consul API to evaluate the kind and type of filter selectors.
		_, err := bexpr.CreateFilter(opt)
		if err != nil {
			return nil, fmt.Errorf(
				"service.regex: invalid filter: %q: %s", opt, err)
		}
		filters = append(filters, opt)
	}

	if len(filters) > 0 {
		servicesRegexQuery.filter = strings.Join(filters, " and ")
	}

	if servicesRegexQuery.regexp == nil {
		return nil, fmt.Errorf("service.regex: regexp option required")
	}

	return &servicesRegexQuery, nil
}

// Fetch queries the Consul API defined by the given client and returns a slice
// of HealthService objects for services that match the set regex.
func (d *servicesRegexQuery) Fetch(clients dep.Clients) (interface{}, *dep.ResponseMetadata, error) {
	select {
	case <-d.stopCh:
		return nil, nil, dep.ErrStopped
	default:
	}

	// Fetch all services via catalog services
	hcatOpts := d.opts.Merge(&hcat.QueryOptions{
		Datacenter: d.dc,
		Namespace:  d.ns,
	})
	opts := hcatOpts.ToConsulOpts()
	if len(d.nodeMeta) != 0 {
		opts.NodeMeta = d.nodeMeta
	}
	catalog, qm, err := clients.Consul().Catalog().Services(opts)
	if err != nil {
		return nil, nil, errors.Wrap(err, d.String())
	}

	// Track the indexes of catalog services, not the individual health services
	rm := &dep.ResponseMetadata{
		LastIndex:   qm.LastIndex,
		LastContact: qm.LastContact,
	}

	// Filter out only the services that match the regex
	var matchServices []string
	for name := range catalog {
		if d.regexp != nil && !d.regexp.MatchString(name) {
			continue
		}
		matchServices = append(matchServices, name)
	}

	// Fetch the health of each matching service. We aren't tracking the latest
	// index for each service, only for the catalog, so we'll do synchronous API
	// requests to fetch the latest
	hcatOpts = &hcat.QueryOptions{
		Datacenter: d.dc,
		Namespace:  d.ns,
		Filter:     d.filter,
	}
	opts = hcatOpts.ToConsulOpts()
	if len(d.nodeMeta) != 0 {
		opts.NodeMeta = d.nodeMeta
	}

	// This Fetch depends on multiple API calls for update. This adds an
	// additional delay to process new updates to this query result. 1 second is
	// used to account for Consul cluster propagation of the change at scale.
	// https://www.hashicorp.com/blog/hashicorp-consul-global-scale-benchmark
	//
	// This affects template functions that have propagation depencies on
	// services. KV query is not affected by this because it is a different
	// entity within Consul.
	//
	// Without this delay, CatalogServices may have services that are not yet
	// propagated to HealthServices as healthy services since services are initially
	// set as critical. https://www.consul.io/docs/discovery/checks#initial-health-check-status
	time.Sleep(1 * time.Second)

	var services []*dep.HealthService
	for _, s := range matchServices {
		var entries []*consulapi.ServiceEntry
		entries, _, err = clients.Consul().Health().Service(s, "", true, opts)
		if err != nil {
			return nil, nil, errors.Wrap(err, d.String())
		}
		for _, entry := range entries {
			address := entry.Service.Address
			if address == "" {
				address = entry.Node.Address
			}
			services = append(services, &dep.HealthService{
				Node:                entry.Node.Node,
				NodeID:              entry.Node.ID,
				Kind:                string(entry.Service.Kind),
				NodeAddress:         entry.Node.Address,
				NodeDatacenter:      entry.Node.Datacenter,
				NodeTaggedAddresses: entry.Node.TaggedAddresses,
				NodeMeta:            entry.Node.Meta,
				ServiceMeta:         entry.Service.Meta,
				Address:             address,
				ID:                  entry.Service.ID,
				Name:                entry.Service.Service,
				Tags: dep.ServiceTags(
					deepCopyAndSortTags(entry.Service.Tags)),
				Status:    entry.Checks.AggregatedStatus(),
				Checks:    entry.Checks,
				Port:      entry.Service.Port,
				Weights:   entry.Service.Weights,
				Namespace: entry.Service.Namespace,
			})
		}
	}

	sort.Stable(ByNodeThenID(services))
	return services, rm, nil
}

// SetOptions satisfies the hcat.QueryOptionsSetter interface which enables
// blocking queries.
func (d *servicesRegexQuery) SetOptions(opts hcat.QueryOptions) {
	d.opts = opts
}

// ID returns the human-friendly version of this query.
func (d *servicesRegexQuery) ID() string {
	var opts []string
	opts = append(opts, fmt.Sprintf("regexp=%s", d.regexp.String()))

	if d.dc != "" {
		opts = append(opts, fmt.Sprintf("dc=%s", d.dc))
	}
	if d.ns != "" {
		opts = append(opts, fmt.Sprintf("ns=%s", d.ns))
	}
	for k, v := range d.nodeMeta {
		opts = append(opts, fmt.Sprintf("node-meta=%s:%s", k, v))
	}
	if d.filter != "" {
		opts = append(opts, fmt.Sprintf("filter=%s", d.filter))
	}

	sort.Strings(opts)
	return fmt.Sprintf("service.regex(%s)",
		strings.Join(opts, "&"))
}

// Stringer interface reuses ID
func (d *servicesRegexQuery) String() string {
	return d.ID()
}

// Stop halts the query's fetch function.
func (d *servicesRegexQuery) Stop() {
	close(d.stopCh)
}

// ByNodeThenID is a sortable slice of Service
type ByNodeThenID []*dep.HealthService

// Len, Swap, and Less are used to implement the sort.Sort interface.
func (s ByNodeThenID) Len() int      { return len(s) }
func (s ByNodeThenID) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s ByNodeThenID) Less(i, j int) bool {
	if s[i].Node < s[j].Node {
		return true
	} else if s[i].Node == s[j].Node {
		return s[i].ID <= s[j].ID
	}
	return false
}
