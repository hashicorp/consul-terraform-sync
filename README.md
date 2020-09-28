[![CircleCI](https://circleci.com/gh/hashicorp/consul-terraform-sync/tree/master.svg?style=svg&circle-token=a88491ffa8b02149fc483c29c6b8b91ed771f5a5)](https://circleci.com/gh/hashicorp/consul-terraform-sync/tree/master)

# Consul NIA

NIA stands for Network Infrastructure Automation. Consul NIA is a service-oriented tool for managing network infrastructure near real-time. Consul NIA runs as a daemon and integrates the network topology maintained by your Consul cluster with your network infrastructure to dynamically secure and connect services.

## Community Support
If you have questions about how `consul-terraform-sync` works, its capabilities or anything other than a bug or feature request (use github's issue tracker for those), please see our community support resources.

Community portal: https://discuss.hashicorp.com/c/consul

Other resources: https://www.consul.io/community.html

Additionally, for issues and pull requests, we'll be using the 👍 reactions as a rough voting system to help gauge community priorities. So please add 👍 to any issue or pull request you'd like to see worked on. Thanks.

## Installation
Consul NIA is a daemon that runs alongside [Consul](https://github.com/hashicorp/consul), similar to other Consul ecosystem tools like [Consul Template](https://github.com/hashicorp/consul-template). Consul NIA is not included with the Consul binary and will need to be installed separately.

### Download
To install Consul NIA, find the appropriate package for your system and download it as a zip archive. Unzip the package to extract the binary named `consul-terraform-sync`. Move the consul-terraform-sync binary to a location available on your `$PATH`.

  1. Download a pre-compiled, released version from the [Consul NIA release page](https://releases.hashicorp.com/consul-terraform-sync/).
  1. Extract the binary using `unzip` or `tar`.
  1. Move the binary into `$PATH`.

```shell
$ wget https://releases.hashicorp.com/consul-terraform-sync/${VERSION}/consul-esm_${VERSION}_${OS}_${ARCH}.zip
$ unzip consul_nia_${VERSION}_${OS}_${ARCH}.zip
$ mv consul-terraform-sync /usr/local/bin/consul-terraform-sync
```

### Build from Source

You can also build Consul NIA from source.

  1. Clone the repository to your local machine.
  1. Pick a [version](https://github.com/hashicorp/consul-terraform-sync/releases) or build from master.
  1. Build Consul NIA using the [Makefile](Makefile).
  1. The `consul-terraform-sync` binary is now installed to `$GOPATH/bin`.

```shell
$ git clone https://github.com/hashicorp/consul-terraform-sync.git
$ cd consul-terraform-sync
$ git fetch --all
$ git checkout tags/vX.Y.Z
$ make dev
$ which consul-terraform-sync
```

### Verify

Once installed, verify the installation works by prompting the help option.

```shell
$ consul-terraform-sync -h
Usage of consul-terraform-sync:
  -config-dir value
      A directory to load files for configuring Consul NIA. Configuration files
      require an .hcl or .json file extention in order to specify their format.
      This option can be specified multiple times to load different directories.
  -config-file value
      A file to load for configuring Consul NIA. Configuration file requires an
      .hcl or .json extension in order to specify their format. This option can
      be specified multiple times to load different configuration files.
  -inspect
      Run Consul NIA in Inspect mode to print the current and proposed state
      change, and then exits. No changes are applied in this mode.
  -version
      Print the version of this daemon.
```

## Configuration

[Documentation to configure Consul NIA](docs/config.md)
