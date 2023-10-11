// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tmplfunc

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/testutils"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/hashicorp/hcat/dep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServicesRegexQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		opts []string
		exp  *servicesRegexQuery
		err  bool
	}{
		{
			"no opts",
			[]string{},
			&servicesRegexQuery{},
			true,
		},
		{
			"regexp empty string",
			[]string{"regexp="},
			&servicesRegexQuery{
				regexp: regexp.MustCompile(""),
			},
			false,
		},
		{
			"regexp",
			[]string{"regexp=.*"},
			&servicesRegexQuery{
				regexp: regexp.MustCompile(".*"),
			},
			false,
		},
		{
			"multiple",
			[]string{"regexp=.*", "\"my-tag\" in Service.Tags", "node-meta=k:v", "ns=namespace", "dc=dc1"},
			&servicesRegexQuery{
				regexp:   regexp.MustCompile(".*"),
				dc:       "dc1",
				ns:       "namespace",
				nodeMeta: map[string]string{"k": "v"},
				filter:   "\"my-tag\" in Service.Tags",
			},
			false,
		},
		{
			"invalid query",
			[]string{"regexp=.*", "invalid=true"},
			nil,
			true,
		},
		{
			"invalid query format",
			[]string{"regexp"},
			nil,
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			act, err := newServicesRegexQuery(tc.opts)
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

func TestServicesRegexQuery_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    []string
		exp  string
	}{
		{
			"regexp",
			[]string{"regexp=.*"},
			"service.regex(regexp=.*)",
		},
		{
			"multiple",
			[]string{"node-meta=k:v", "dc=dc1", "ns=namespace", "regexp=web", "\"my-tag\" in Service.Tags"},
			`service.regex(dc=dc1&filter="my-tag" in Service.Tags&node-meta=k:v&ns=namespace&regexp=web)`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := newServicesRegexQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}

func TestServicesRegexQuery_Fetch(t *testing.T) {
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
	//      - server instance: api-db-1

	// set up nodes
	srv1 := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{})
	defer srv1.Stop()
	consulSrv1 := &dep.HealthService{
		Name:   "consul",
		ID:     "consul",
		NodeID: srv1.Config.NodeID,
		Node:   srv1.Config.NodeName,
	}

	nodeMeta := map[string]string{"k": "v"}
	tb := &testutils.TestingTB{}
	srv2, err := testutil.NewTestServerConfigT(tb,
		func(c *testutil.TestServerConfig) {
			c.Bootstrap = false
			c.LogLevel = "warn"
			c.Stdout = io.Discard
			c.Stderr = io.Discard
			c.NodeMeta = nodeMeta
		})
	consulSrv2 := &dep.HealthService{
		Name:     "consul",
		ID:       "consul",
		NodeMeta: nodeMeta,
		NodeID:   srv2.Config.NodeID,
		Node:     srv2.Config.NodeName,
	}

	require.NoError(t, err, "failed to start consul server 2")
	defer srv2.Stop()

	srv3, err := testutil.NewTestServerConfigT(tb,
		func(c *testutil.TestServerConfig) {
			c.Datacenter = "dc2"
			c.Bootstrap = true
			c.LogLevel = "warn"
			c.Stdout = io.Discard
			c.Stderr = io.Discard
		})
	require.NoError(t, err, "failed to start consul server 3")
	defer srv3.Stop()
	consulSrv3 := &dep.HealthService{
		Name:     "consul",
		ID:       "consul",
		NodeMeta: nodeMeta,
		NodeID:   srv3.Config.NodeID,
		Node:     srv3.Config.NodeName,
	}

	// join nodes
	srv1.JoinLAN(t, srv2.LANAddr) // dc1: srv1, srv2
	srv1.JoinWAN(t, srv3.WANAddr) // dc2: srv3

	// register services
	apiSrv := &dep.HealthService{ID: "api-1", Name: "api", Tags: []string{"tag1"}, Node: consulSrv1.Node, NodeID: consulSrv1.NodeID}
	service := testutil.TestService{ID: apiSrv.ID, Name: apiSrv.Name, Tags: apiSrv.Tags}
	testutils.RegisterConsulServiceHealth(t, srv1, service, 8*time.Second, testutil.HealthPassing)

	apiWebSrv := &dep.HealthService{ID: "api-web-1", Name: "api-web", Node: consulSrv2.Node, NodeID: consulSrv2.NodeID}
	service = testutil.TestService{ID: apiWebSrv.ID, Name: apiWebSrv.Name}
	testutils.RegisterConsulServiceHealth(t, srv2, service, 8*time.Second, testutil.HealthPassing)

	webSrv := &dep.HealthService{ID: "web-1", Name: "web", Node: consulSrv2.Node, NodeID: consulSrv2.NodeID}
	service = testutil.TestService{ID: webSrv.ID, Name: webSrv.Name}
	testutils.RegisterConsulServiceHealth(t, srv2, service, 8*time.Second, testutil.HealthPassing)

	dbSrv := &dep.HealthService{ID: "db-1", Name: "api-db", Node: consulSrv3.Node, NodeID: consulSrv3.NodeID}
	service = testutil.TestService{ID: dbSrv.ID, Name: dbSrv.Name}
	testutils.RegisterConsulServiceHealth(t, srv3, service, 8*time.Second, testutil.HealthPassing)

	// set up consul client for srv1
	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr
	client, err := consulapi.NewClient(consulConfig)
	require.NoError(t, err, "failed to make consul client")

	cases := []struct {
		name     string
		i        []string
		expected []*dep.HealthService
	}{
		{
			"regexp only",
			[]string{"regexp=api.*"},
			[]*dep.HealthService{
				apiSrv,
				apiWebSrv,
			},
		},
		{
			"node-meta",
			[]string{"regexp=web", "node-meta=k:v"},
			[]*dep.HealthService{
				webSrv,
				apiWebSrv,
			},
		},
		{
			"dc",
			[]string{"regexp=api", "dc=dc2"},
			[]*dep.HealthService{
				dbSrv,
			},
		},
		{
			"filter",
			[]string{"regexp=.*", "\"tag1\" in Service.Tags"},
			[]*dep.HealthService{
				apiSrv,
			},
		},
		{
			"none matching",
			[]string{"regexp=noop"},
			[]*dep.HealthService{},
		},
		{
			"all matching",
			[]string{"regexp=.*"},
			[]*dep.HealthService{
				// defaults to dc1
				apiSrv,
				apiWebSrv,
				webSrv,
				consulSrv1,
				consulSrv2,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := newServicesRegexQuery(tc.i)
			require.NoError(t, err)

			a, _, err := d.Fetch(&testClient{consul: client})
			require.NoError(t, err)
			actual, ok := a.([]*dep.HealthService)
			require.True(t, ok)
			require.Equal(t, len(tc.expected), len(actual))

			sort.Stable(ByNodeThenID(tc.expected))
			sort.Stable(ByNodeThenID(actual))
			for i, actualNode := range actual {
				assert.Equal(t, tc.expected[i].Name, actualNode.Name)
				assert.Equal(t, tc.expected[i].ID, actualNode.ID)
				assert.Equal(t, tc.expected[i].Node, actualNode.Node,
					fmt.Sprintf("unexpected node name for service %s", actualNode.Node))
				assert.Equal(t, tc.expected[i].NodeID, actualNode.NodeID,
					fmt.Sprintf("unexpected node id for service %s", actualNode.ID))
			}
		})
	}
}
