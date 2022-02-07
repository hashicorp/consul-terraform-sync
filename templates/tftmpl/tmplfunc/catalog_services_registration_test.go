package tmplfunc

import (
	"io/ioutil"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/testutils"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/hashicorp/hcat/dep"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCatalogServicesRegistrationQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		opts []string
		exp  *catalogServicesRegistrationQuery
		err  bool
	}{
		{
			"no opts",
			[]string{},
			&catalogServicesRegistrationQuery{},
			false,
		},
		{
			"regexp",
			[]string{"regexp=.*"},
			&catalogServicesRegistrationQuery{
				regexp: regexp.MustCompile(".*"),
			},
			false,
		},
		{
			"dc",
			[]string{"dc=dc1"},
			&catalogServicesRegistrationQuery{
				dc: "dc1",
			},
			false,
		},
		{
			"ns",
			[]string{"ns=namespace"},
			&catalogServicesRegistrationQuery{
				ns: "namespace",
			},
			false,
		},
		{
			"node-meta",
			[]string{"node-meta=k:v", "node-meta=foo:bar"},
			&catalogServicesRegistrationQuery{
				nodeMeta: map[string]string{"k": "v", "foo": "bar"},
			},
			false,
		},
		{
			"multiple",
			[]string{"node-meta=k:v", "ns=namespace", "dc=dc1", "regexp=.*"},
			&catalogServicesRegistrationQuery{
				regexp:   regexp.MustCompile(".*"),
				dc:       "dc1",
				ns:       "namespace",
				nodeMeta: map[string]string{"k": "v"},
			},
			false,
		},
		{
			"invalid query",
			[]string{"invalid=true"},
			nil,
			true,
		},
		{
			"invalid query format",
			[]string{"dc1"},
			nil,
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			act, err := newCatalogServicesRegistrationQuery(tc.opts)
			if tc.err {
				assert.Error(t, err)
				return
			}

			if act != nil {
				act.stopCh = nil
			}

			assert.NoError(t, err, err)
			assert.Equal(t, tc.exp, act)
		})
	}
}

func TestCatalogServicesRegistrationQuery_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    []string
		exp  string
	}{
		{
			"empty",
			[]string{},
			"catalog.services.registration",
		},
		{
			"regexp",
			[]string{"regexp=.*"},
			"catalog.services.registration(regexp=.*)",
		},
		{
			"datacenter",
			[]string{"dc=dc1"},
			"catalog.services.registration(dc=dc1)",
		},
		{
			"namespace",
			[]string{"ns=namespace"},
			"catalog.services.registration(ns=namespace)",
		},
		{
			"node-meta",
			[]string{"node-meta=k:v", "node-meta=foo:bar"},
			"catalog.services.registration(node-meta=foo:bar&node-meta=k:v)",
		},
		{
			"multiple",
			[]string{"node-meta=k:v", "dc=dc1", "ns=namespace", "regexp=.*"},
			"catalog.services.registration(dc=dc1&node-meta=k:v&ns=namespace&regexp=.*)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := newCatalogServicesRegistrationQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}

func TestCatalogServicesRegistrationQuery_Fetch(t *testing.T) {
	t.Parallel()

	// Test is fetching services from a Consul cluster set-up as:
	// dc1: (wan joined with dc2)
	//   - node: srv1 (no node-meta) (lan joined with srv2)
	//      - server instance: api-1
	//   - node: srv2 (with node-meta)
	//      - server instance: api-web-1
	//      - server instance: web-1
	// dc2:
	//   - node: srv3 (no node-meta)
	//      - server instance: db-1

	// set up nodes
	srv1 := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{})
	defer srv1.Stop()

	tb := &testutils.TestingTB{}
	srv2, err := testutil.NewTestServerConfigT(tb,
		func(c *testutil.TestServerConfig) {
			c.Bootstrap = false
			c.LogLevel = "warn"
			c.Stdout = ioutil.Discard
			c.Stderr = ioutil.Discard
			c.NodeMeta = map[string]string{"k": "v"}
		})

	require.NoError(t, err, "failed to start consul server 2")
	defer srv2.Stop()

	srv3, err := testutil.NewTestServerConfigT(tb,
		func(c *testutil.TestServerConfig) {
			c.Datacenter = "dc2"
			c.Bootstrap = true
			c.LogLevel = "warn"
			c.Stdout = ioutil.Discard
			c.Stderr = ioutil.Discard
		})
	require.NoError(t, err, "failed to start consul server 3")
	defer srv3.Stop()

	// join nodes
	srv1.JoinLAN(t, srv2.LANAddr) // dc1: srv1, srv2
	srv1.JoinWAN(t, srv3.WANAddr) // dc2: srv3

	// register services
	service := testutil.TestService{ID: "api-1", Name: "api", Tags: []string{"tag1"}}
	testutils.RegisterConsulService(t, srv1, service, 8*time.Second)

	service = testutil.TestService{ID: "api-web-1", Name: "api-web"}
	testutils.RegisterConsulService(t, srv2, service, 8*time.Second)

	service = testutil.TestService{ID: "web-1", Name: "web"}
	testutils.RegisterConsulService(t, srv2, service, 8*time.Second)

	service = testutil.TestService{ID: "db-1", Name: "db"}
	testutils.RegisterConsulService(t, srv3, service, 8*time.Second)

	// set up consul client for srv1
	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr
	client, err := consulapi.NewClient(consulConfig)
	require.NoError(t, err, "failed to make consul client")

	// wait for consul service to be registered in dc2
	WaitForCatalogRegistration(t, client, &consulapi.QueryOptions{Datacenter: "dc2"},
		"consul", 8*time.Second)

	cases := []struct {
		name     string
		i        []string
		expected []*dep.CatalogSnippet
	}{
		{
			"no filtering (in dc1)",
			[]string{},
			[]*dep.CatalogSnippet{
				{Name: "api", Tags: dep.ServiceTags([]string{"tag1"})},
				{Name: "api-web", Tags: dep.ServiceTags([]string{})},
				{Name: "consul", Tags: dep.ServiceTags([]string{})},
				{Name: "web", Tags: dep.ServiceTags([]string{})},
			},
		},
		{
			"node-meta filter (in dc1)",
			[]string{"node-meta=k:v"},
			[]*dep.CatalogSnippet{
				{Name: "api-web", Tags: dep.ServiceTags([]string{})},
				{Name: "consul", Tags: dep.ServiceTags([]string{})},
				{Name: "web", Tags: dep.ServiceTags([]string{})},
			},
		},
		{
			"regexp filter (in dc1)",
			[]string{"regexp=api"},
			[]*dep.CatalogSnippet{
				{Name: "api", Tags: dep.ServiceTags([]string{"tag1"})},
				{Name: "api-web", Tags: dep.ServiceTags([]string{})},
			},
		},
		{
			"dc filter",
			[]string{"dc=dc2"},
			[]*dep.CatalogSnippet{
				{Name: "consul", Tags: dep.ServiceTags([]string{})},
				{Name: "db", Tags: dep.ServiceTags([]string{})},
			},
		},
		{
			"all filters (besides namespace)",
			[]string{"dc=dc1", "regexp=api", "node-meta=k:v"},
			[]*dep.CatalogSnippet{
				{Name: "api-web", Tags: dep.ServiceTags([]string{})},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := newCatalogServicesRegistrationQuery(tc.i)
			assert.NoError(t, err)

			actual, _, err := d.Fetch(&testClient{consul: client})
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

// testClient implements hcat's dep.Client interface used to make requests
// to Consul and Vault
type testClient struct {
	consul *consulapi.Client
}

// Consul returns the Consul client
func (c *testClient) Consul() *consulapi.Client {
	if c == nil {
		return nil
	}
	return c.consul
}

// Vault returns the Vault client
func (c *testClient) Vault() *vaultapi.Client {
	// no-op: currently no need to support Vault
	return nil
}

// WaitForCatalogRegistration polls and waits for a service to be registered in the Consul catalog
func WaitForCatalogRegistration(tb testing.TB, client *consulapi.Client, queryOpts *consulapi.QueryOptions,
	serviceID string, wait time.Duration) {
	polling := make(chan struct{})
	stopPolling := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopPolling:
				return
			default:
				resp, _, err := client.Catalog().Services(queryOpts)
				require.NoError(tb, err)
				if _, ok := resp[serviceID]; ok {
					polling <- struct{}{}
					return
				}
			}
		}
	}()

	select {
	case <-polling:
		return
	case <-time.After(wait):
		close(stopPolling)
		tb.Fatalf("timed out after waiting for %v for service %q to register "+
			"with in the Consul catalog", wait, serviceID)
	}
}
