# Consul-Terraform-Sync [![Build Status](https://github.com/hashicorp/consul-terraform-sync/actions/workflows/build.yml/badge.svg)](https://github.com/hashicorp/consul-terraform-sync/actions/workflows/build.yml) [![CircleCI](https://circleci.com/gh/hashicorp/consul-terraform-sync/tree/main.svg?style=svg&circle-token=a88491ffa8b02149fc483c29c6b8b91ed771f5a5)](https://circleci.com/gh/hashicorp/consul-terraform-sync/tree/main)


Consul-Terraform-Sync (just CTS from here on) is a service-oriented tool for managing network infrastructure near real-time. CTS runs as a daemon and integrates the network topology maintained by your Consul cluster with your network infrastructure to dynamically secure and connect services.

* Website: [consul.io/docs/nia](https://www.consul.io/docs/nia)

## Community Support
If you have questions about how `consul-terraform-sync` works, its capabilities or anything other than a bug or feature request (use GitHub's issue tracker for those), please see our community support resources.

Community portal: [discuss.hashicorp.com](https://discuss.hashicorp.com/tags/c/consul/29/consul-terraform-sync)

Other resources: [consul.io/community](https://www.consul.io/community)

Additionally, for issues and pull requests, we'll be using the 👍 reactions as a rough voting system to help gauge community priorities. So please add 👍 to any issue or pull request you'd like to see worked on. Thanks.

## Roadmap

Knowing about our upcoming features and priorities helps our users plan. This repository contains information about what we are working on and allows all our user to give direct feedback. Please visit the [Roadmap FAQs](#roadmap-faqs) for more information on the categorization of the roadmap.

[See the roadmap »](https://github.com/hashicorp/consul-terraform-sync/projects/1)



## Installation
CTS is a daemon that runs alongside [Consul](https://github.com/hashicorp/consul), similar to other Consul ecosystem tools like [Consul Template](https://github.com/hashicorp/consul-template). CTS is not included with the Consul binary and will need to be installed separately.

### Download
To install CTS, find the appropriate package for your system and download it as a zip archive. Unzip the package to extract the binary named `consul-terraform-sync`. Move the consul-terraform-sync binary to a location available on your `$PATH`.

  1. Download a pre-compiled, released version from the [CTS release page](https://releases.hashicorp.com/consul-terraform-sync/).
  1. Extract the binary using `unzip` or `tar`.
  1. Move the binary into `$PATH`.

```shell
$ wget https://releases.hashicorp.com/consul-terraform-sync/${VERSION}/consul-terraform-sync_${VERSION}_${OS}_${ARCH}.zip
$ unzip consul-terraform-sync_${VERSION}_${OS}_${ARCH}.zip
$ mv consul-terraform-sync /usr/local/bin/consul-terraform-sync
```

### Build from Source

You can also build CTS from source.

  1. Clone the repository to your local machine.
  1. Pick a [version](https://github.com/hashicorp/consul-terraform-sync/releases) or build from main.
  1. Build CTS using the [Makefile](Makefile).
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
Usage CLI: consul-terraform-sync <command> [-help] [options]

Commands:
    start
     task

Options:

    -autocomplete-install false
        Install the autocomplete

    -autocomplete-uninstall false
        Uninstall the autocomplete

    -config-dir 
        A directory to load files for configuring Consul-Terraform-Sync. 
        Configuration files require an .hcl or .json file extension in order 
        to specify their format. This option can be specified multiple times to 
        load different directories.

    -config-file 
        A file to load for configuring Consul-Terraform-Sync. Configuration 
        file requires an .hcl or .json extension in order to specify their format. 
        This option can be specified multiple times to load different 
        configuration files.

    -inspect false
        Run Consul-Terraform-Sync in Inspect mode to print the proposed state 
        changes for all tasks, and then exit. No changes are applied 
        in this mode.

    -inspect-task 
        Run Consul-Terraform-Sync in Inspect mode to print the proposed 
        state changes for the task, and then exit. No changes are applied
        in this mode.

    -once false
        Render templates and run tasks once. Does not run the process 
        as a daemon and disables buffer periods.
```

## Configuration

[Documentation to configure CTS](https://consul.io/docs/nia/configuration)

## Roadmap FAQs
**Why did you build this public facing roadmap?**

We know that our customers are making decisions and plans based on what we are developing, and we want to provide our customers the insights they need to plan.

**Why does your roadmap not have specific dates?**

Because the highest priority for us to put out a secure and operationally stable product, we can't provide specific target dates for features. The roadmap is subject to change at any time, and roadmap issues in this repository do not guarantee a feature will be launched as proposed.

**What are the roadmap categories?**
* *Recently Shipped* - These are the features, enhancements or bug fixes that we recently delivered.
* *Up Next* - Features, enhancements or bug fixes coming in the next couple of months.
* *Researching* - This might mean that we are still designing and thinking through how this feature or enhancement might work. We would love to hear from you on how you would like to see something implemented. Additionally, we would like to hear your use case or design ideas here.


**How can I request a feature be added to the roadmap?**

Please open an issue!  Guidelines on how to contribute can be found [here](/CONTRIBUTING.md). Community submitted issues will be tagged "Enhancement" and will be reviewed by the team.

**Will you accept a pull request?**

We want to create a strong community around CTS. We will take all PRs very seriously and review for inclusion. Please read about [contributing](/CONTRIBUTING.md).
