module github.com/hashicorp/consul-nia

go 1.14

require (
	github.com/hashicorp/consul v1.8.0
	github.com/hashicorp/consul/sdk v0.5.0
	github.com/hashicorp/go-syslog v1.0.0
	github.com/hashicorp/go-version v1.2.1 // indirect
	github.com/hashicorp/hcat v0.0.0-20200819213737-8df9f16d7129
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/hcl/v2 v2.6.0
	github.com/hashicorp/logutils v1.0.0
	github.com/hashicorp/terraform v0.12.29
	github.com/hashicorp/terraform-exec v0.3.0
	github.com/mitchellh/mapstructure v1.3.3
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.5.1
	github.com/zclconf/go-cty v1.5.1
	golang.org/x/crypto v0.0.0-20200709230013-948cd5f35899
	golang.org/x/text v0.3.3 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
)

replace (
	// Terraform imports a pre-go-mod version of Vault. These replace directives
	// resolves the ambiguous import between the package `vault/api` and
	// `vault/api` nested go module.
	github.com/hashicorp/vault => github.com/hashicorp/vault v1.5.0
	github.com/hashicorp/vault/api => github.com/hashicorp/vault/api v1.0.5-0.20190730042357-746c0b111519
)
