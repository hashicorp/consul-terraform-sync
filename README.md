[![CircleCI](https://circleci.com/gh/hashicorp/consul-nia/tree/master.svg?style=svg&circle-token=a88491ffa8b02149fc483c29c6b8b91ed771f5a5)](https://circleci.com/gh/hashicorp/consul-nia/tree/master)

# Consul NIA

NIA stands for Network Infrastructure Automation. Consul NIA is a service-oriented tool for managing network infrastructure near real-time. Consul NIA runs as a daemon and integrates the network topology maintained by your Consul cluster with your network infrastructure to dynamically secure and connect services.

## Community Support
If you have questions about how `consul-nia` works, its capabilities or anything other than a bug or feature request (use github's issue tracker for those), please see our community support resources.

Community portal: https://discuss.hashicorp.com/c/consul

Other resources: https://www.consul.io/community.html

Additionally, for issues and pull requests, we'll be using the 👍 reactions as a rough voting system to help gauge community priorities. So please add 👍 to any issue or pull request you'd like to see worked on. Thanks.

## Installation
Consul NIA is a daemon that runs alongside [Consul](https://github.com/hashicorp/consul), similar to other Consul ecosystem tools like [Consul Template](https://github.com/hashicorp/consul-template). Consul NIA is not included with the Consul binary and will need to be installed separately.

### Download
To install Consul NIA, find the appropriate package for your system and download it as a zip archive. Unzip the package to extract the binary named `consul-nia`. Move the consul-nia binary to a location available on your `$PATH`.

  1. Download a pre-compiled, released version from the [Consul NIA release page](https://releases.hashicorp.com/consul-nia/).
  1. Extract the binary using `unzip` or `tar`.
  1. Move the binary into `$PATH`.

```shell
$ wget https://releases.hashicorp.com/consul-nia/${VERSION}/consul-esm_${VERSION}_${OS}_${ARCH}.zip
$ unzip consul_nia_${VERSION}_${OS}_${ARCH}.zip
$ mv consul-nia /usr/local/bin/consul-nia
```

### Build from Source

You can also build Consul NIA from source.

  1. Clone the repository to your local machine.
  1. Pick a [version](https://github.com/hashicorp/consul-nia/releases) or build from master.
  1. Build Consul NIA using the [Makefile](Makefile).
  1. The `consul-nia` binary is now installed to `$GOPATH/bin`.

```shell
$ git clone https://github.com/hashicorp/consul-nia.git
$ cd consul-nia
$ git fetch --all
$ git checkout tags/vX.Y.Z
$ make dev
$ which consul-nia
```

### Verify

Once installed, verify the installation works by prompting the help option.

```shell
$ consul-nia -help
Usage of consul-nia:
  -config-dir value
      A directory to look for .hcl or .json config files in. Can be specified multiple times.
  -config-file value
      A config file to use. Can be either .hcl or .json format. Can be specified multiple times.
  -inspect
      Print the current and proposed state change, and then exits.
  -version
      Print the version of this daemon.
```
