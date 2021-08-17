package tmplfunc

import (
	"testing"

	"github.com/hashicorp/consul-terraform-sync/testutils"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/hcat/dep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNodesQuery(t *testing.T) {
	cases := []struct {
		name string
		opts []string
		exp  *nodesQuery
		err  bool
	}{
		{
			"no opts",
			[]string{},
			&nodesQuery{},
			false,
		},
		{
			"datacenter",
			[]string{"datacenter=dc2"},
			&nodesQuery{
				dc: "dc2",
			},
			false,
		},
		{
			"invalid option",
			[]string{"hello=world"},
			nil,
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			act, err := newNodesQuery(tc.opts)
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

func TestNodesQuery_String(t *testing.T) {
	cases := []struct {
		name string
		i    []string
		exp  string
	}{
		{

			"datacenter",
			[]string{"datacenter=dc2"},
			"node(dc=dc2)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := newNodesQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}

func TestNodesQuery_Fetch(t *testing.T) {
	srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{})
	defer srv.Stop()

	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = srv.HTTPAddr
	client, err := consulapi.NewClient(consulConfig)
	require.NoError(t, err, "failed to make consul client")

	srv.WaitForLeader(t)

	cases := []struct {
		name     string
		i        []string
		expected []*dep.Node
	}{
		{
			"empty",
			[]string{},
			[]*dep.Node{
				{
					Address:    "127.0.0.1",
					Datacenter: "dc1",
					TaggedAddresses: map[string]string{
						"lan":      "127.0.0.1",
						"lan_ipv4": "127.0.0.1",
						"wan":      "127.0.0.1",
						"wan_ipv4": "127.0.0.1",
					},
					Meta: map[string]string{
						"consul-network-segment": "",
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := newNodesQuery(tc.i)
			require.NoError(t, err)

			a, _, err := d.Fetch(&testClient{consul: client})
			require.NoError(t, err)
			actual, ok := a.([]*dep.Node)
			require.True(t, ok)
			require.Equal(t, len(tc.expected), len(actual))

			for i, actualNode := range actual {
				assert.Equal(t, tc.expected[i].Address, actualNode.Address)
				assert.Equal(t, tc.expected[i].Datacenter, actualNode.Datacenter)
				assert.Equal(t, tc.expected[i].TaggedAddresses, actualNode.TaggedAddresses)
				assert.Equal(t, tc.expected[i].Meta, actualNode.Meta)
			}
		})
	}
}
