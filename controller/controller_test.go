package controller

import (
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/stretchr/testify/assert"
)

func TestNewDriverTasks(t *testing.T) {
	// newDriverTasks function reorganizes various user-defined configuration
	// blocks into a task object with all the information for the driver to
	// execute on.
	testCases := []struct {
		name  string
		conf  *config.Config
		tasks []driver.Task
	}{
		{
			"no config",
			nil,
			[]driver.Task{},
		}, {
			"no tasks",
			&config.Config{Tasks: &config.TaskConfigs{}},
			[]driver.Task{},
		}, {
			// Fetches correct provider and required_providers blocks from config
			"providers",
			&config.Config{
				Tasks: &config.TaskConfigs{
					{
						Name:      config.String("name"),
						Providers: []string{"providerA", "providerB"},
						Source:    config.String("source"),
					},
				},
				Driver: &config.DriverConfig{
					Terraform: &config.TerraformConfig{
						RequiredProviders: map[string]interface{}{
							"providerA": map[string]string{
								"source": "source/providerA",
							},
						},
					},
				},
				Providers: &config.ProviderConfigs{
					{"providerB": map[string]interface{}{
						"var": "val",
					}},
				},
			},
			[]driver.Task{{
				Name: "name",
				Providers: []map[string]interface{}{
					{"providerA": map[string]interface{}{}},
					{"providerB": map[string]interface{}{
						"var": "val",
					}},
				},
				ProviderInfo: map[string]interface{}{
					"providerA": map[string]string{
						"source": "source/providerA",
					},
				},
				Services: []driver.Service{},
				Source:   "source",
				VarFiles: []string{},
			}},
		}, {
			// Fetches correct provider and required_providers blocks from config
			// with context of alias
			"provider instance",
			&config.Config{
				Tasks: &config.TaskConfigs{
					{
						Name:      config.String("name"),
						Providers: []string{"providerA.alias1", "providerB"},
						Source:    config.String("source"),
					},
				},
				Driver: &config.DriverConfig{
					Terraform: &config.TerraformConfig{
						RequiredProviders: map[string]interface{}{
							"providerA": map[string]string{
								"source": "source/providerA",
							},
						},
					},
				},
				Providers: &config.ProviderConfigs{
					{"providerA": map[string]interface{}{
						"alias": "alias1",
						"foo":   "bar",
					}},
					{"providerA": map[string]interface{}{
						"alias": "alias2",
						"baz":   "baz",
					}},
					{"providerB": map[string]interface{}{
						"var": "val",
					}},
				},
			},
			[]driver.Task{{
				Name: "name",
				Providers: []map[string]interface{}{
					{"providerA": map[string]interface{}{
						"alias": "alias1",
						"foo":   "bar",
					}},
					{"providerB": map[string]interface{}{
						"var": "val",
					}},
				},
				ProviderInfo: map[string]interface{}{
					"providerA": map[string]string{
						"source": "source/providerA",
					},
				},
				Services: []driver.Service{},
				Source:   "source",
				VarFiles: []string{},
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.conf.Finalize()
			tasks := newDriverTasks(tc.conf)
			assert.Equal(t, tc.tasks, tasks)
		})
	}
}
