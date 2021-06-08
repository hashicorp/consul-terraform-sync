package tmplfunc

import (
	"testing"

	"github.com/hashicorp/hcat/dep"
	"github.com/stretchr/testify/assert"
)

func TestHCLServiceFunc(t *testing.T) {
	testCases := []struct {
		name     string
		content  *dep.HealthService
		expected string
	}{
		{
			"nil",
			nil,
			"",
		}, {
			"empty",
			&dep.HealthService{},
			`id                    = ""
name                  = ""
kind                  = ""
address               = ""
port                  = 0
meta                  = {}
tags                  = []
namespace             = ""
status                = ""
node                  = ""
node_id               = ""
node_address          = ""
node_datacenter       = ""
node_tagged_addresses = {}
node_meta             = {}
cts_user_defined_meta = {}`,
		}, {
			"basic",
			&dep.HealthService{
				ID:             "api",
				Name:           "api",
				Address:        "1.2.3.4",
				Port:           8080,
				ServiceMeta:    map[string]string{"key": "value"},
				Tags:           []string{"tag"},
				Status:         "passing",
				Node:           "worker-01",
				NodeID:         "39e5a7f5-2834-e16d-6925-78167c9f50d8",
				NodeAddress:    "127.0.0.1",
				NodeDatacenter: "dc1",
				NodeTaggedAddresses: map[string]string{
					"lan":      "127.0.0.1",
					"lan_ipv4": "127.0.0.1",
					"wan":      "127.0.0.1",
					"wan_ipv4": "127.0.0.1",
				},
				NodeMeta: map[string]string{
					"consul-network-segment": "",
				},
			},
			`id      = "api"
name    = "api"
kind    = ""
address = "1.2.3.4"
port    = 8080
meta = {
  key = "value"
}
tags            = ["tag"]
namespace       = ""
status          = "passing"
node            = "worker-01"
node_id         = "39e5a7f5-2834-e16d-6925-78167c9f50d8"
node_address    = "127.0.0.1"
node_datacenter = "dc1"
node_tagged_addresses = {
  lan      = "127.0.0.1"
  lan_ipv4 = "127.0.0.1"
  wan      = "127.0.0.1"
  wan_ipv4 = "127.0.0.1"
}
node_meta = {
  consul-network-segment = ""
}
cts_user_defined_meta = {}`,
		}, {
			"namespace-n-kind",
			&dep.HealthService{
				Namespace: "namespace",
				Kind:      "mykind",
			},
			`id                    = ""
name                  = ""
kind                  = "mykind"
address               = ""
port                  = 0
meta                  = {}
tags                  = []
namespace             = "namespace"
status                = ""
node                  = ""
node_id               = ""
node_address          = ""
node_datacenter       = ""
node_tagged_addresses = {}
node_meta             = {}
cts_user_defined_meta = {}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := hclServiceFunc(nil)(tc.content)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestHCLServiceFunc_ctsUserDefinedMeta(t *testing.T) {
	meta := map[string]map[string]string{
		"api": {
			"key":        "value",
			"foo_bar":    "baz",
			"spaced key": "spaced value",
		},
	}
	content := &dep.HealthService{
		ID:      "api",
		Name:    "api",
		Address: "1.2.3.4",
		Port:    8080,
	}
	expected := `id                    = "api"
name                  = "api"
kind                  = ""
address               = "1.2.3.4"
port                  = 8080
meta                  = {}
tags                  = []
namespace             = ""
status                = ""
node                  = ""
node_id               = ""
node_address          = ""
node_datacenter       = ""
node_tagged_addresses = {}
node_meta             = {}
cts_user_defined_meta = {
  foo_bar      = "baz"
  key          = "value"
  "spaced key" = "spaced value"
}`

	actual := hclServiceFunc(meta)(content)
	assert.Equal(t, expected, actual)
}
