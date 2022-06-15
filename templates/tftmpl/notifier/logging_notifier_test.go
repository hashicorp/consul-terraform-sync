package notifier

import (
	"bytes"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/hcat/dep"
	"github.com/stretchr/testify/assert"
)

func TestNotifier_logDependency(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		dep          interface{}
		logSubString string
	}{
		{
			"services",
			[]*dep.HealthService{
				{Name: "api", ID: "api-1"},
				{Name: "api", ID: "api-2"},
				{Name: "web", ID: "web-1"},
			},
			`received dependency: variable=services ids=["api-1", "api-2", "web-1"]`,
		},
		{
			"catalog-services",
			[]*dep.CatalogSnippet{
				{Name: "api"},
				{Name: "web"},
			},
			`received dependency: variable=catalog_services names=["api", "web"]`,
		},
		{
			"key-pair",
			&dep.KeyPair{Key: "key_a", Value: "value_a"},
			`received dependency: variable=consul_kv recurse=false key=key_a`,
		},
		{
			"key-pairs",
			[]*dep.KeyPair{
				{Key: "key_a", Value: "value_a"},
				{Key: "key_b", Value: "value_b"},
			},
			`received dependency: variable=consul_kv recurse=true keys=["key_a", "key_b"]`,
		},
		{
			"unknown",
			[]string{"data_a", "data_b"},
			`received unknown dependency: variable=[]string`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := logging.Setup(&logging.Config{
				Level:  "DEBUG",
				Writer: &buf,
			})
			assert.NoError(t, err)

			logDependency(logging.Global(), tc.dep)
			assert.Contains(t, buf.String(), tc.logSubString)
		})
	}
}
