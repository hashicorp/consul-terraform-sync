module github.com/hashicorp/consul-terraform-sync

go 1.16

require (
	cloud.google.com/go v0.78.0 // indirect
	cloud.google.com/go/storage v1.13.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/PaloAltoNetworks/pango v0.5.1
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/aws/aws-sdk-go v1.37.19 // indirect
	github.com/fatih/color v1.10.0 // indirect
	github.com/go-test/deep v1.0.7 // indirect
	github.com/golang/snappy v0.0.2 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/hashicorp/consul v1.9.3
	github.com/hashicorp/consul/api v1.8.1
	github.com/hashicorp/consul/sdk v0.7.0
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-checkpoint v0.5.0
	github.com/hashicorp/go-getter v1.5.3
	github.com/hashicorp/go-hclog v0.15.0 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.8 // indirect
	github.com/hashicorp/go-syslog v1.0.0
	github.com/hashicorp/go-tfe v0.12.0
	github.com/hashicorp/go-uuid v1.0.2
	github.com/hashicorp/go-version v1.3.0
	github.com/hashicorp/hcat v0.0.0-20210430144333-5970348d8f49
	github.com/hashicorp/hcl v1.0.1-vault
	github.com/hashicorp/hcl/v2 v2.8.2
	github.com/hashicorp/logutils v1.0.0
	github.com/hashicorp/terraform v0.14.7
	github.com/hashicorp/terraform-exec v0.13.3
	github.com/hashicorp/vault/api v1.0.5-0.20210210214158-405eced08457
	github.com/hashicorp/vault/sdk v0.1.14-0.20210322210658-b52b8b8c1264 // indirect
	github.com/klauspost/compress v1.11.7 // indirect
	github.com/miekg/dns v1.1.40 // indirect
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
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/zclconf/go-cty v1.8.2
	go.opencensus.io v0.22.6 // indirect
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
	github.com/hashicorp/vault/api => github.com/hashicorp/vault/api v1.0.5-0.20200805123347-1ef507638af6

	// pin this version to avoid later versions that depend on a v1alpha1 branch that's no longer available
	// this is a transitive dependency through vault, which pins v0.18.2, we use the same version here
	k8s.io/api => k8s.io/api v0.18.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.2
	k8s.io/client-go => k8s.io/client-go v0.18.2
)
