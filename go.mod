module github.com/hashicorp/consul-terraform-sync

go 1.16

require (
	cloud.google.com/go v0.78.0 // indirect
	cloud.google.com/go/storage v1.13.0 // indirect
	github.com/DataDog/zstd v1.4.5 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/PaloAltoNetworks/pango v0.5.1
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/aws/aws-sdk-go v1.37.16 // indirect
	github.com/bitly/go-hostpool v0.1.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/fatih/color v1.10.0 // indirect
	github.com/go-git/go-git/v5 v5.2.0 // indirect
	github.com/golang/snappy v0.0.2 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/hashicorp/consul v1.9.3
	github.com/hashicorp/consul/sdk v0.7.0
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-checkpoint v0.5.0
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-getter v1.5.2 // indirect
	github.com/hashicorp/go-hclog v0.15.0 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.8 // indirect
	github.com/hashicorp/go-syslog v1.0.0
	github.com/hashicorp/go-uuid v1.0.2
	github.com/hashicorp/go-version v1.2.1
	github.com/hashicorp/hcat v0.0.0-20210401143330-8f813cb572a8
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/hcl/v2 v2.8.2
	github.com/hashicorp/logutils v1.0.0
	github.com/hashicorp/terraform v0.14.7
	github.com/hashicorp/terraform-exec v0.13.0
	github.com/hashicorp/vault v1.4.2
	github.com/hashicorp/vault/api v1.0.5-0.20200717191844-f687267c8086
	github.com/kevinburke/ssh_config v0.0.0-20201106050909-4977a11b4351 // indirect
	github.com/klauspost/compress v1.11.7 // indirect
	github.com/mitchellh/cli v1.1.2
	github.com/mitchellh/copystructure v1.1.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/mitchellh/mapstructure v1.4.1
	github.com/pierrec/lz4 v2.6.0+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/tidwall/pretty v1.0.2 // indirect
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/xanzy/ssh-agent v0.3.0 // indirect
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c // indirect
	github.com/xdg/stringprep v1.0.0 // indirect
	github.com/zclconf/go-cty v1.8.0
	go.opencensus.io v0.22.6 // indirect
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83 // indirect
	golang.org/x/net v0.0.0-20210222171744-9060382bd457 // indirect
	golang.org/x/oauth2 v0.0.0-20210220000619-9bb904979d93 // indirect
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba // indirect
	google.golang.org/genproto v0.0.0-20210222212404-3e1e516060db // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
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
