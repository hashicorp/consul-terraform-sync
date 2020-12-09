## UNRELEASED

BREAKING CHANGES:
* Deprecate `provider` block name in this release for `terraform_provider` block name, and `provider` will be removed in the following release [[GH-140](https://github.com/hashicorp/consul-terraform-sync/pull/140)]

FEATURES:
* Add inspect mode to view proposed state changes for tasks [[GH-124](https://github.com/hashicorp/consul-terraform-sync/pull/124)]
* Expand usage of Terraform backends for state store [[GH-101](https://github.com/hashicorp/consul-terraform-sync/pull/101), [GH-129](https://github.com/hashicorp/consul-terraform-sync/pull/129)]
  * azurerm, cos, gcs, kubernetes, local, manta, pg, s3
* Add configuration option to select Terraform version to install and run [[GH-131](https://github.com/hashicorp/consul-terraform-sync/pull/131)]
  * Add support to run Terraform version 0.14
* Add status api to view status information about task execution
  * Task-status api for status of each task [[GH-138](https://github.com/hashicorp/consul-terraform-sync/pull/138)]
  * Overall-status api for the overall status across tasks [[GH-142](https://github.com/hashicorp/consul-terraform-sync/pull/142)]
  * Support configuring `port` on which the api is served [[GH-141](https://github.com/hashicorp/consul-terraform-sync/pull/141)]
  * Support `include=events` parameter for task-status api to include in the response payload the information of task execution events [[GH-145](https://github.com/hashicorp/consul-terraform-sync/pull/145)]
  * Support `status=<health-status>` parameter for task-status api to only return statuses of tasks of a specified health status [[GH-147](https://github.com/hashicorp/consul-terraform-sync/pull/147)]
* Add support to load arguments for `terraform_provider` blocks from env, Consul KV, and Vault using template syntax [[GH-143](https://github.com/hashicorp/consul-terraform-sync/pull/143)]

IMPROVEMENTS:
* Enable 3 retries on task execution errors [[GH-72](https://github.com/hashicorp/consul-terraform-sync/pull/72), [GH-121](https://github.com/hashicorp/consul-terraform-sync/pull/121)]
* Update out-of-band commits to execute only when a related task is successful [[GH-122](https://github.com/hashicorp/consul-terraform-sync/pull/122)]

BUG FIXES:
* Fix indefinite retries connecting to Consul on DNS errors [[GH-133](https://github.com/hashicorp/consul-terraform-sync/pull/133)]
* Fix Terraform workspace selection error [[GH-134](https://github.com/hashicorp/consul-terraform-sync/issues/134)]

## 0.1.0-techpreview1 (October 09, 2020)

* Initial release
