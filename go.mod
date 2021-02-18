module github.com/hashicorp/consul-terraform-sync

go 1.14

require (
	github.com/DataDog/zstd v1.4.5 // indirect
	github.com/PaloAltoNetworks/pango v0.4.1-0.20200904214627-5b4d88ba9b10
	github.com/bitly/go-hostpool v0.1.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/hashicorp/consul v1.8.0
	github.com/hashicorp/consul/sdk v0.5.0
	github.com/hashicorp/go-checkpoint v0.5.0
	github.com/hashicorp/go-syslog v1.0.0
	github.com/hashicorp/go-uuid v1.0.2
	github.com/hashicorp/go-version v1.2.1
	github.com/hashicorp/hcat v0.0.0-20210126174106-fbb1851663ad
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/hcl/v2 v2.6.0
	github.com/hashicorp/logutils v1.0.0
	github.com/hashicorp/terraform v0.12.29
	github.com/hashicorp/terraform-exec v0.9.0
	github.com/hashicorp/vault v1.4.2
	github.com/hashicorp/vault/api v1.0.5-0.20200630205458-1a16f3c699c6
	github.com/mitchellh/cli v1.1.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/go-wordwrap v1.0.0
	github.com/mitchellh/mapstructure v1.3.3
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.1
	github.com/tidwall/pretty v1.0.2 // indirect
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c // indirect
	github.com/xdg/stringprep v1.0.0 // indirect
	github.com/zclconf/go-cty v1.6.1
	golang.org/x/crypto v0.0.0-20200709230013-948cd5f35899 // indirect
	golang.org/x/text v0.3.3 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
)

replace (
	// Terraform imports a pre-go-mod version of Vault. These replace directives
	// resolves the ambiguous import between the package `vault/api` and
	// `vault/api` nested go module.
	github.com/hashicorp/vault => github.com/hashicorp/vault v1.5.5
	github.com/hashicorp/vault/api => github.com/hashicorp/vault/api v1.0.5-0.20200805123347-1ef507638af6
	github.com/hashicorp/vault/http => github.com/hashicorp/vault/http v1.5.5
	github.com/hashicorp/vault/vault => github.com/hashicorp/vault/vault v1.5.5
)
