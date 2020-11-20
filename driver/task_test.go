package driver

import (
	"reflect"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/client"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/client"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		clientType  string
		expectError bool
		expect      client.Client
	}{
		{
			"happy path with development client",
			developmentClient,
			false,
			&client.Printer{},
		},
		{
			"happy path with mock client",
			testClient,
			false,
			&mocks.Client{},
		},
		{
			"error when creating Terraform CLI client",
			"",
			true,
			&client.TerraformCLI{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := newClient(&clientConfig{
				task:       Task{},
				clientType: tc.clientType,
			})
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, reflect.TypeOf(tc.expect), reflect.TypeOf(actual))
			}
		})
	}
}

func TestTask_ProviderNames(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		task     Task
		expected []string
	}{
		{
			"no provider",
			Task{},
			[]string{},
		},
		{
			"happy path",
			Task{
				Providers: hcltmpl.NewNamedBlocksTest([]map[string]interface{}{
					{"local": map[string]interface{}{
						"configs": "stuff",
					}},
					{"null": map[string]interface{}{}},
				}),
			},
			[]string{"local", "null"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.task.ProviderNames()
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestTask_ServiceNames(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		task     Task
		expected []string
	}{
		{
			"no services",
			Task{},
			[]string{},
		},
		{
			"happy path",
			Task{
				Services: []Service{
					Service{Name: "web"},
					Service{Name: "api"},
				},
			},
			[]string{"web", "api"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.task.ServiceNames()
			assert.Equal(t, tc.expected, actual)
		})
	}
}
