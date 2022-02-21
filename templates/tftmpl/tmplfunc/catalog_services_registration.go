package tmplfunc

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/hcat"
	"github.com/hashicorp/hcat/dep"
	"github.com/pkg/errors"
)

var _ hcatQuery = (*catalogServicesRegistrationQuery)(nil)

// catalogServicesRegistrationFunc returns information on registered Consul
// services. It queries the Catalog List Services API and supports the query
// parameters dc, ns, and node-meta. It also adds an additional layer of
// custom functionality on the API response:
//  - Adds regex filtering on service name option e.g. "regexp=api"
//
// Endpoint: /v1/catalog/services
// Template: {{ catalogServicesRegistration  <filter options> ... }}
func catalogServicesRegistrationFunc(recall hcat.Recaller) interface{} {
	return func(opts ...string) ([]*dep.CatalogSnippet, error) {
		result := []*dep.CatalogSnippet{}

		d, err := newCatalogServicesRegistrationQuery(opts)
		if err != nil {
			return nil, err
		}

		if value, ok := recall(d); ok {
			return value.([]*dep.CatalogSnippet), nil
		}

		return result, nil
	}
}

// catalogServicesRegistrationQuery is the representation of a requested catalog
// service registration query from inside a template.
type catalogServicesRegistrationQuery struct {
	isConsul
	stopCh chan struct{}

	regexp   *regexp.Regexp // custom
	dc       string
	ns       string
	nodeMeta map[string]string
	opts     hcat.QueryOptions
}

// newCatalogServicesRegistrationQuery processes options in the format of
// "key=value" e.g. "dc=dc1"
func newCatalogServicesRegistrationQuery(opts []string) (*catalogServicesRegistrationQuery, error) {
	query := catalogServicesRegistrationQuery{
		stopCh: make(chan struct{}, 1),
	}

	for _, opt := range opts {
		if strings.TrimSpace(opt) == "" {
			continue
		}

		param, value, err := stringsSplit2(opt, "=")
		if err != nil {
			return nil, fmt.Errorf("catalog.services.registration: invalid "+
				"query parameter format: %q", opt)
		}
		switch param {
		case "regexp":
			r, err := regexp.Compile(value)
			if err != nil {
				return nil, fmt.Errorf("catalog.services.registration: invalid regexp")
			}
			query.regexp = r
		case "dc", "datacenter":
			query.dc = value
		case "ns", "namespace":
			query.ns = value
		case "node-meta":
			if query.nodeMeta == nil {
				query.nodeMeta = make(map[string]string)
			}
			k, v, err := stringsSplit2(value, ":")
			if err != nil {
				return nil, fmt.Errorf(
					"catalog.services.registration: invalid format for query "+
						"parameter %q: %s", param, value)
			}
			query.nodeMeta[k] = v
		default:
			return nil, fmt.Errorf(
				"catalog.services.registration: invalid query parameter: %q", opt)
		}
	}

	return &query, nil
}

// Fetch queries the Consul API defined by the given client and returns a slice
// of CatalogSnippet objects.
func (d *catalogServicesRegistrationQuery) Fetch(clients dep.Clients) (interface{}, *dep.ResponseMetadata, error) {
	select {
	case <-d.stopCh:
		return nil, nil, dep.ErrStopped
	default:
	}

	hcatOpts := d.opts.Merge(&hcat.QueryOptions{
		Datacenter: d.dc,
		Namespace:  d.ns,
	})
	opts := hcatOpts.ToConsulOpts()
	if len(d.nodeMeta) != 0 {
		opts.NodeMeta = d.nodeMeta
	}

	entries, qm, err := clients.Consul().Catalog().Services(opts)
	if err != nil {
		return nil, nil, errors.Wrap(err, d.String())
	}

	var catalogServices []*dep.CatalogSnippet
	for name, tags := range entries {
		if d.regexp != nil && !d.regexp.MatchString(name) {
			continue
		}
		catalogServices = append(catalogServices, &dep.CatalogSnippet{
			Name: name,
			Tags: dep.ServiceTags(deepCopyAndSortTags(tags)),
		})
	}

	sort.Stable(ByName(catalogServices))

	rm := &dep.ResponseMetadata{
		LastIndex:   qm.LastIndex,
		LastContact: qm.LastContact,
	}

	// Tasks monitoring CatalogServices will likely also monitor Services
	// information. Without a delay, CatalogServices may have services that
	// are not yet propagated to Services template function as healthy services
	// since services are initially set as critical.
	// https://www.consul.io/docs/discovery/checks#initial-health-check-status
	//
	// This adds an additional delay to process new Services used elsewhere for
	// this task. 1 second is used to account for Consul cluster propagation of
	// the change at scale. https://www.hashicorp.com/blog/hashicorp-consul-global-scale-benchmark
	//
	// This affects template functions that have propagation depencies on
	// services. KV query is not affected by this because it is a different
	// entity within Consul.
	//
	// Note: this creates unnecessary latency for tasks that monitor
	// CatalogServices but not Services. Currently this use-case seems unlikely
	// so have not implemented this complexity at this time
	time.Sleep(1 * time.Second)

	return catalogServices, rm, nil
}

// SetOptions satisfies the hcat.QueryOptionsSetter interface which enables
// blocking queries.
func (d *catalogServicesRegistrationQuery) SetOptions(opts hcat.QueryOptions) {
	d.opts = opts
}

// ID returns the human-friendly version of this query.
func (d *catalogServicesRegistrationQuery) ID() string {
	var opts []string
	if d.regexp != nil {
		opts = append(opts, fmt.Sprintf("regexp=%s", d.regexp.String()))
	}
	if d.dc != "" {
		opts = append(opts, fmt.Sprintf("dc=%s", d.dc))
	}
	if d.ns != "" {
		opts = append(opts, fmt.Sprintf("ns=%s", d.ns))
	}
	for k, v := range d.nodeMeta {
		opts = append(opts, fmt.Sprintf("node-meta=%s:%s", k, v))
	}
	if len(opts) > 0 {
		sort.Strings(opts)
		return fmt.Sprintf("catalog.services.registration(%s)",
			strings.Join(opts, "&"))
	}
	return "catalog.services.registration"
}

// Stringer interface reuses ID
func (d *catalogServicesRegistrationQuery) String() string {
	return d.ID()
}

// Stop halts the query's fetch function.
func (d *catalogServicesRegistrationQuery) Stop() {
	close(d.stopCh)
}

// ByName is a sortable slice of CatalogSnippet structs.
type ByName []*dep.CatalogSnippet

func (s ByName) Len() int      { return len(s) }
func (s ByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s ByName) Less(i, j int) bool {
	return s[i].Name <= s[j].Name
}

// stringsSplit2 splits a string
func stringsSplit2(s string, sep string) (string, string, error) {
	split := strings.Split(s, sep)
	if len(split) != 2 {
		return "", "", fmt.Errorf("unexpected split on separator %q: %s", sep, s)
	}
	return strings.TrimSpace(split[0]), strings.TrimSpace(split[1]), nil
}
