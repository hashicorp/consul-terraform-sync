package tmplfunc

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcat"
	"github.com/hashicorp/hcat/dep"
	"github.com/pkg/errors"
)

var (
	_ hcatQuery = (*intentionsQuery)(nil)
)

// returns info on registered consul services
// reference from catalog_services_registration
func intentionsFunc(recall hcat.Recaller) interface{} {
	return func(opts ...string) ([]*dep.Intention, error) {
		result := []*dep.Intention{}

		d, err := newIntentionsQuery(opts)
		if err != nil {
			return nil, err
		}

		if value, ok := recall(d); ok {
			return value.([]*dep.Intention), nil
		}

		return result, nil
	}
}

type ServiceTags []string

// intentionsQuery is the representation of a requested
// intention query from inside a template.
type intentionsQuery struct {
	isConsul
	stopCh chan struct{}

	id      string
	name    string
	address string
	//port     	int
	kind string
	//tags      ServiceTags
	namespace string
	status    string
	nodeMeta  map[string]string
	opts      hcat.QueryOptions
}

// newIntentionsQuery processes options
func newIntentionsQuery(opts []string) (*intentionsQuery, error) {
	query := intentionsQuery{
		stopCh: make(chan struct{}, 1),
	}

	for _, opt := range opts {
		if strings.TrimSpace(opt) == "" {
			continue
		}

		param, value, err := stringsSplit2(opt, "=")
		if err != nil {
			return nil, fmt.Errorf("intentions.services: invalid "+
				"query parameter format: %q", opt)
		}
		switch param {
		case "id":
			query.id = value
		case "name":
			query.name = value
		case "address":
			query.address = value
		// case "port":
		// 	query.port = value
		case "kind":
			query.kind = value
		// case "tags":
		// 	query.tags = value
		case "namespace":
			query.namespace = value
		case "status":
			query.kind = value
		case "node-meta":
			if query.nodeMeta == nil {
				query.nodeMeta = make(map[string]string)
			}
			k, v, err := stringsSplit2(value, ":")
			if err != nil {
				return nil, fmt.Errorf(
					"intention.services: invalid format for query "+
						"parameter %q: %s", param, value)
			}
			query.nodeMeta[k] = v
		default:
			return nil, fmt.Errorf(
				"intentions.services: invalid query parameter: %q", opt)
		}
	}

	return &query, nil

}

// // IntentionSnippet is an Intention entry in Consul.
// type IntentionSnippet struct {
// 	SourceName      string
// 	DestinationName string
// 	Permissions     []*IntentionPermission
// 	Precedence      int
// }

func (d *intentionsQuery) Fetch(clients dep.Clients) (interface{}, *dep.ResponseMetadata, error) {
	select {
	case <-d.stopCh:
		return nil, nil, dep.ErrStopped
	default:
	}

	hcatOpts := d.opts.Merge(&hcat.QueryOptions{
		// ID:      d.id,
		// Name:    d.name,
		// Address: d.address,
		//port     	int
		// Kind: d.kind,
		//tags      ServiceTags
		Namespace: d.namespace,
		// Status:    d.status,
	})
	opts := hcatOpts.ToConsulOpts()
	if len(d.nodeMeta) != 0 {
		opts.NodeMeta = d.nodeMeta
	}

	entries, qm, err := clients.Consul().Connect().Intentions(opts)
	if err != nil {
		return nil, nil, errors.Wrap(err, d.String())
	}

	var intentionSnippets []*dep.IntentionSnippet
	for sourceName, destinationName := range entries {
		intentionSnippets = append(intentionSnippets, &dep.IntentionSnippet{
			// SourceName:      sourceName,
			// DestinationName: destinationName,
		})
	}
	// //return nil, nil, dep.ErrStopped

	// sort.Stable(ByName(intentionSnippets, reverse(less)))

	// rm := &dep.ResponseMetadata{
	// 	LastIndex:   qm.LastIndex,
	// 	LastContact: qm.LastContact,
	// }

	return nil, nil, dep.ErrStopped
	// return intentionSnippets, rm, nil
}

func (d *intentionsQuery) SetOptions(opts hcat.QueryOptions) {
	d.opts = opts
}

func (d *intentionsQuery) ID() string {
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
		return fmt.Sprintf("intentions.services(%s)",
			strings.Join(opts, "&"))
	}
	// return ""
	return "intentions.services"
}

func (d *intentionsQuery) String() string {
	return d.ID()
}
func (d *intentionsQuery) Stop() {
	close(d.stopCh)
}

func reverse(less func(i, j int) bool) func(i, j int) bool {
	return func(i, j int) bool {
		return !less(i, j)
	}
}
