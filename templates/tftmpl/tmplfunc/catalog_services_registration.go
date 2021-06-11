package tmplfunc

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/hcat"
	"github.com/hashicorp/hcat/dep"
	"github.com/pkg/errors"
)

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
	stopCh chan struct{}

	regexp   *regexp.Regexp // custom
	dc       string
	ns       string
	nodeMeta map[string]string
	opts     consulapi.QueryOptions
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

	opts := d.opts
	if d.dc != "" {
		opts.Datacenter = d.dc
	}
	if d.ns != "" {
		opts.Namespace = d.ns
	}
	if len(d.nodeMeta) != 0 {
		opts.NodeMeta = d.nodeMeta
	}

	entries, qm, err := clients.Consul().Catalog().Services(&opts)
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

	return catalogServices, rm, nil
}

// String returns the human-friendly version of this query.
func (d *catalogServicesRegistrationQuery) String() string {
	var opts []string
	if d.regexp != nil {
		opts = append(opts, fmt.Sprintf("regexp=%s", d.regexp.String()))
	}
	if d.dc != "" {
		opts = append(opts, fmt.Sprintf("@%s", d.dc))
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

// Stop halts the query's fetch function.
func (d *catalogServicesRegistrationQuery) Stop() {
	close(d.stopCh)
}

func (d *catalogServicesRegistrationQuery) SetOptions(opts consulapi.QueryOptions) {
	d.opts = opts
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
