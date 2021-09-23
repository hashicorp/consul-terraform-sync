package tftmpl

import (
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendRootTerraformBlock_backend(t *testing.T) {
	consulBackend, err := config.DefaultTerraformBackend(&config.ConsulConfig{
		Address: config.String("consul.example.com"),
		TLS: &config.TLSConfig{
			Enabled: config.Bool(true),
			CACert:  config.String("ca_cert"),
			Cert:    config.String("cert"),
			Key:     config.String("key"),
		},
	})
	require.NoError(t, err)

	testCases := []struct {
		name       string
		rawBackend map[string]interface{}
		expected   string
	}{
		{
			"nil",
			nil,
			`terraform {
  required_version = ">= 0.13.0, < 1.1.0"
}
`,
		}, {
			"empty",
			map[string]interface{}{"empty": map[string]interface{}{}},
			`terraform {
  required_version = ">= 0.13.0, < 1.1.0"
  backend "empty" {
  }
}
`,
		}, {
			"invalid structure",
			map[string]interface{}{"invalid": "unexpected type"},
			`terraform {
  required_version = ">= 0.13.0, < 1.1.0"
}
`,
		}, {
			"local",
			map[string]interface{}{"local": map[string]interface{}{
				"path": "relative/path/to/terraform.tfstate",
			}},
			`terraform {
  required_version = ">= 0.13.0, < 1.1.0"
  backend "local" {
    path = "relative/path/to/terraform.tfstate"
  }
}
`,
		}, {
			"consul",
			consulBackend,
			`terraform {
  required_version = ">= 0.13.0, < 1.1.0"
  backend "consul" {
    address   = "consul.example.com"
    ca_file   = "ca_cert"
    cert_file = "cert"
    gzip      = true
    key_file  = "key"
    path      = "consul-terraform-sync/terraform"
    scheme    = "https"
  }
}
`,
		}, {
			"postgres",
			map[string]interface{}{"pg": map[string]interface{}{
				"conn_str": "postgres://user:pass@db.example.com/terraform_backend",
			}},
			`terraform {
  required_version = ">= 0.13.0, < 1.1.0"
  backend "pg" {
    conn_str = "postgres://user:pass@db.example.com/terraform_backend"
  }
}
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hclFile := hclwrite.NewEmptyFile()
			body := hclFile.Body()

			var backend *hcltmpl.NamedBlock
			if tc.rawBackend != nil {
				b := hcltmpl.NewNamedBlock(tc.rawBackend)
				backend = &b
			}
			appendRootTerraformBlock(body, backend, nil)

			content := hclFile.Bytes()
			content = hclwrite.Format(content)
			assert.Equal(t, tc.expected, string(content))
		})
	}
}

func TestAppendRootProviderBlocks(t *testing.T) {
	testCases := []struct {
		name       string
		rawBackend map[string]interface{}
		expected   string
	}{
		{
			"nil",
			nil,
			`provider "" {
}
`,
		}, {
			"empty",
			map[string]interface{}{"empty": map[string]interface{}{}},
			`provider "empty" {
}
`,
		}, {
			"internal alias leak",
			map[string]interface{}{"foo": map[string]interface{}{
				"alias": "bar",
			}},
			`provider "foo" {
}
`,
		}, {
			"internal auto_commit leak",
			map[string]interface{}{"foo": map[string]interface{}{
				"auto_commit": "true",
			}},
			`provider "foo" {
}
`,
		}, {
			"invalid structure",
			map[string]interface{}{"invalid": "unexpected type"},
			`provider "" {
}
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hclFile := hclwrite.NewEmptyFile()
			body := hclFile.Body()

			backend := []hcltmpl.NamedBlock{hcltmpl.NewNamedBlock(tc.rawBackend)}
			appendRootProviderBlocks(body, backend)

			content := hclFile.Bytes()
			content = hclwrite.Format(content)
			assert.Equal(t, tc.expected, string(content))
		})
	}
}

func TestAppendRootModuleBlocks(t *testing.T) {
	testCases := []struct {
		name     string
		task     Task
		cond     Condition
		si       SourceInput
		varNames []string
		expected string
	}{
		{
			name: "module without conditions or variables",
			task: Task{
				Description: "user description for task named 'test'",
				Name:        "test",
				Source:      "namespace/example/test-module",
				Version:     "1.0.0",
			},
			cond:     nil,
			varNames: nil,
			expected: `# user description for task named 'test'
module "test" {
  source   = "namespace/example/test-module"
  version  = "1.0.0"
  services = var.services
}
`},
		{
			name: "module with catalog service conditions",
			task: Task{
				Description: "user description for task named 'test'",
				Name:        "test",
				Source:      "namespace/example/test-module",
				Version:     "1.0.0",
			},
			cond: &CatalogServicesCondition{
				CatalogServicesMonitor: CatalogServicesMonitor{
					Regexp: ".*",
				},
				SourceIncludesVar: true,
			},
			varNames: nil,
			expected: `# user description for task named 'test'
module "test" {
  source           = "namespace/example/test-module"
  version          = "1.0.0"
  services         = var.services
  catalog_services = var.catalog_services
}
`},
		{
			name: "module with variables",
			task: Task{
				Description: "user description for task named 'test'",
				Name:        "test",
				Source:      "namespace/example/test-module",
				Version:     "1.0.0",
			},
			cond:     nil,
			varNames: []string{"test1", "test2"},
			expected: `# user description for task named 'test'
module "test" {
  source   = "namespace/example/test-module"
  version  = "1.0.0"
  services = var.services

  test1 = var.test1
  test2 = var.test2
}
`},
		{
			name: "module with catalog service conditions",
			task: Task{
				Description: "user description for task named 'test'",
				Name:        "test",
				Source:      "namespace/example/test-module",
				Version:     "1.0.0",
			},
			cond: &CatalogServicesCondition{
				CatalogServicesMonitor: CatalogServicesMonitor{
					Regexp: ".*",
				},
				SourceIncludesVar: true,
			},
			varNames: nil,
			expected: `# user description for task named 'test'
module "test" {
  source           = "namespace/example/test-module"
  version          = "1.0.0"
  services         = var.services
  catalog_services = var.catalog_services
}
`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hclFile := hclwrite.NewEmptyFile()
			body := hclFile.Body()
			appendRootModuleBlock(body, tc.task, tc.varNames, tc.cond, tc.si)

			content := hclFile.Bytes()
			content = hclwrite.Format(content)
			assert.Equal(t, tc.expected, string(content))
		})
	}
}

func TestService_hcatQuery(t *testing.T) {
	testCases := []struct {
		name     string
		service  Service
		expected string
	}{
		{
			"empty",
			Service{},
			`""`,
		}, {
			"base",
			Service{Name: "app"},
			`"app"`,
		}, {
			"datacenter",
			Service{
				Name:       "app",
				Datacenter: "dc1",
			},
			`"app" "dc=dc1"`,
		}, {
			"namespace",
			Service{
				Name:      "app",
				Namespace: "namespace",
			},
			`"app" "ns=namespace"`,
		}, {
			"filter",
			Service{
				Name:   "filtered-app",
				Filter: `"test" in Service.Tags or Service.Tags is empty`,
			},
			`"filtered-app" "\"test\" in Service.Tags or Service.Tags is empty"`,
		}, {
			"all",
			Service{
				Name:       "app",
				Datacenter: "dc1",
				Namespace:  "namespace",
				Filter:     `Service.Meta["meta-key"] contains "test"`,
			},
			`"app" "dc=dc1" "ns=namespace" "Service.Meta[\"meta-key\"] contains \"test\""`,
		},
	}
	for _, tc := range testCases {
		actual := tc.service.hcatQuery()
		assert.Equal(t, tc.expected, actual)
	}
}
