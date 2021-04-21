## 0.1.1 (April 21, 2021)

BUG FIXES:
* Fix issue where CTS does not reconnect with Consul if it stops and restarts by adding retries for up to 8-12 minutes and then exiting if retries are unsuccessful. [[GH-233](https://github.com/hashicorp/consul-terraform-sync/issues/233), [GH-242](https://github.com/hashicorp/consul-terraform-sync/pull/242)]
* Fix issue with services template being generated before data on all services is ready. [[GH-239](https://github.com/hashicorp/consul-terraform-sync/issues/239), [GH-257](https://github.com/hashicorp/consul-terraform-sync/pull/257)]

## 0.1.0 (March 29, 2021)

BUG FIXES:
* Fix Task Status API response which was incorrectly returning empty providers and services information when requesting a task with no event data. [[GH-219](https://github.com/hashicorp/consul-terraform-sync/pull/219)]
* Fix service filtering with tag containing `=`. [[GH-222](https://github.com/hashicorp/consul-terraform-sync/pull/222)]
* Fix Docker image to pass in configuration when running in daemon-mode. [[GH-221](https://github.com/hashicorp/consul-terraform-sync/pull/221)]
* Mitigate task execution on partial data when monitoring a large number of services. [[GH-232](https://github.com/hashicorp/consul-terraform-sync/pull/232)]
* Fix tasks that are watching the same services from going stale after a couple executions. [[GH-234](https://github.com/hashicorp/consul-terraform-sync/issues/234), [GH-237](https://github.com/hashicorp/consul-terraform-sync/pull/237)]
* Fix exponential backoff retry, which was incorrectly implementing x^2 instead of 2^x. Used to retry PANOS commit and Terraform. [[GH-235](https://github.com/hashicorp/consul-terraform-sync/pull/235)]
* Fix `-version` flag output to include the binary name. [[GH-238](https://github.com/hashicorp/consul-terraform-sync/pull/238)]

## 0.1.0-beta (February 25, 2021)

BREAKING CHANGES:
* Remove support for `provider` block name (deprecated v0.1.0-techpreview2). Use `terraform_provider` block name instead. [[GH-169](https://github.com/hashicorp/consul-terraform-sync/pull/169)]
* Change version output from stderr to stdout. [[GH-199](https://github.com/hashicorp/consul-terraform-sync/pull/199)]
* Change API error structure from string to object for future flexibility. [[GH-201](https://github.com/hashicorp/consul-terraform-sync/pull/201)]
* Change Overall Status API response payload's `task_summary` from a map of status values to counts to a map of objects in order to allow returning other types of summary information. [[GH-203](https://github.com/hashicorp/consul-terraform-sync/pull/203)]

FEATURES:
* Add `cts_user_defined_meta` option to the `service` configuration block for appending user-defined metadata grouped by services to be used by Terraform modules. [[GH-166](https://github.com/hashicorp/consul-terraform-sync/pull/166)]
* Add support for querying service by namespace for Consul Enterprise. [[GH-175](https://github.com/hashicorp/consul-terraform-sync/pull/175)]
* Add `enabled` boolean field to task configuration which configures a task to run or not. [[GH-188](https://github.com/hashicorp/consul-terraform-sync/pull/188), [GH-189](https://github.com/hashicorp/consul-terraform-sync/pull/189)]
* Add a Disable Task CLI which will stop a task from running and updating resources until re-enabled. [[GH-194](https://github.com/hashicorp/consul-terraform-sync/pull/194)]
* Add an Enable Task CLI which will start a task so that it runs and updates resources. [[GH-198](https://github.com/hashicorp/consul-terraform-sync/pull/198)]
* Add support for a CLI `-port` flag to set the API port that the CLI should use if not default port 8558. [[GH-197](https://github.com/hashicorp/consul-terraform-sync/pull/197)]
* Add an Update Task API to support patch updating a task's enabled state. [[GH-191](https://github.com/hashicorp/consul-terraform-sync/pull/191), [GH-214](https://github.com/hashicorp/consul-terraform-sync/pull/214)]
* Add a run parameter to Update Task API which can dry-run a task with updates and return an inspect plan (?run=inspect) or update a task run it immediately as opposed to run at the natural CTS cadence (?run=now). [[GH-196](https://github.com/hashicorp/consul-terraform-sync/pull/196)]
* Configurable PAN-OS out-of-band commits [[GH-170](https://github.com/hashicorp/consul-terraform-sync/pull/170)]
* PAN-OS commit retry with exponential backoff [[GH-178](https://github.com/hashicorp/consul-terraform-sync/pull/178)]
* Add support for CTS to communicate with the local Consul agent over HTTP/2 to improve the efficiency of TCP connections for monitoring the Consul catalog [[GH-146](https://github.com/hashicorp/consul-terraform-sync/issues/146), [GH-207](https://github.com/hashicorp/consul-terraform-sync/pull/207)].
* Official docker image [[GH-215](https://github.com/hashicorp/consul-terraform-sync/pull/215)]

IMPROVEMENTS:
* Changed default `consul.transport` options used for the Consul client to improve TCP connection reuse. [[GH-164](https://github.com/hashicorp/consul-terraform-sync/pull/164)]
* Mark generated provider variables as [sensitive for Terraform 0.14+](https://www.hashicorp.com/blog/terraform-0-14-adds-the-ability-to-redact-sensitive-values-in-console-output) [[GH-181](https://github.com/hashicorp/consul-terraform-sync/pull/181)]
* Separate provider-related variables into a different file from services [[GH-182](https://github.com/hashicorp/consul-terraform-sync/pull/182), [GH-183](https://github.com/hashicorp/consul-terraform-sync/pull/183)]
* Update the Overall Status API response to return count of enabled and disabled tasks and to return count of tasks with no event data as status value 'unknown'. [[GH-203](https://github.com/hashicorp/consul-terraform-sync/pull/203)]
* Update the Task Status API response to include a new 'enabled' boolean field to indicate if task is enabled or disabled. [[GH-202](https://github.com/hashicorp/consul-terraform-sync/pull/202)]
* Include service kind in module input [[GH-168](https://github.com/hashicorp/consul-terraform-sync/issues/168), [GH-174](https://github.com/hashicorp/consul-terraform-sync/pull/174)]

BUG FIXES:
* Avoid appending duplicate `terraform` suffix to the KV path for Consul backend. [[GH-165](https://github.com/hashicorp/consul-terraform-sync/pull/165)]
* Fix edge case where multiple tasks have identical `terraform.tfvars.tmpl` files causing Consul Terraform Sync to indefinitely hang. [[GH-167](https://github.com/hashicorp/consul-terraform-sync/pull/167)]
* Handle case where provider configuration used nested blocks, which was causing an unsupported argument error. [[GH-173](https://github.com/hashicorp/consul-terraform-sync/pull/173)]
* Fix `task_env` config validation causing the feature to be unusable. [[GH-184](https://github.com/hashicorp/consul-terraform-sync/pull/184)]
* Fix how CTS configures the Consul KV backend for Terraform remote state store to default with configuration from the Consul block. [[GH-213](https://github.com/hashicorp/consul-terraform-sync/pull/213)]

## 0.1.0-techpreview2 (December 16, 2020)

BREAKING CHANGES:
* Deprecate `provider` block name in this release for `terraform_provider` block name, and `provider` will be removed in the following release [[GH-140](https://github.com/hashicorp/consul-terraform-sync/pull/140)]
* Fix PAN-OS out-of-band commits to use partial commits based on the configured admin user (required when using the PAN-OS provider) instead of committing all queued changes from any user [[GH-137](https://github.com/hashicorp/consul-terraform-sync/pull/137)].

FEATURES:
* Add inspect mode to view proposed state changes for tasks [[GH-124](https://github.com/hashicorp/consul-terraform-sync/pull/124)]
* Expand usage of Terraform backends for state store [[GH-101](https://github.com/hashicorp/consul-terraform-sync/pull/101), [GH-129](https://github.com/hashicorp/consul-terraform-sync/pull/129)]
  * azurerm, cos, gcs, kubernetes, local, manta, pg, s3
* Add configuration option to select Terraform version to install and run [[GH-131](https://github.com/hashicorp/consul-terraform-sync/pull/131)]
  * Add support to run Terraform version 0.14
* Add status api to view status information about task execution. Served by default at port 8558 [[GH-158](https://github.com/hashicorp/consul-terraform-sync/pull/158)]
  * Task-status api for status of each task [[GH-138](https://github.com/hashicorp/consul-terraform-sync/pull/138), [GH-144](https://github.com/hashicorp/consul-terraform-sync/pull/144), [GH-148](https://github.com/hashicorp/consul-terraform-sync/pull/148), [GH-159](https://github.com/hashicorp/consul-terraform-sync/pull/159), [GH-160](https://github.com/hashicorp/consul-terraform-sync/pull/160)]
  * Overall-status api for the overall status across tasks [[GH-142](https://github.com/hashicorp/consul-terraform-sync/pull/142), [GH-161](https://github.com/hashicorp/consul-terraform-sync/pull/161)]
  * Support configuring `port` on which the api is served [[GH-141](https://github.com/hashicorp/consul-terraform-sync/pull/141)]
  * Support `include=events` parameter for task-status api to include in the response payload the information of task execution events [[GH-145](https://github.com/hashicorp/consul-terraform-sync/pull/145)]
  * Support `status=<health-status>` parameter for task-status api to only return statuses of tasks of a specified health status [[GH-147](https://github.com/hashicorp/consul-terraform-sync/pull/147)]
* Add support to dynamically load Terraform provider arguments within the `terraform_provider` blocks from env, Consul KV, and Vault using template syntax [[GH-143](https://github.com/hashicorp/consul-terraform-sync/pull/143)]
  * Add Vault config option [[GH-139](https://github.com/hashicorp/consul-terraform-sync/pull/139)]
* Add support to set Terraform provider environment variables using the meta-argument `task_env` block to avoid rendering sensitive arguments in plain-text or to re-map environment variable names [[GH-157](https://github.com/hashicorp/consul-terraform-sync/pull/157)]

IMPROVEMENTS:
* Enable 2 retries on task execution errors when running in daemon mode [[GH-72](https://github.com/hashicorp/consul-terraform-sync/pull/72), [GH-121](https://github.com/hashicorp/consul-terraform-sync/pull/121), [GH-155](https://github.com/hashicorp/consul-terraform-sync/pull/155)]
* Update out-of-band commits to execute only when a related task is successful [[GH-122](https://github.com/hashicorp/consul-terraform-sync/pull/122)]

BUG FIXES:
* Fix indefinite retries connecting to Consul on DNS errors [[GH-133](https://github.com/hashicorp/consul-terraform-sync/pull/133)]
* Fix Terraform workspace selection error [[GH-134](https://github.com/hashicorp/consul-terraform-sync/issues/134)]

## 0.1.0-techpreview1 (October 09, 2020)

* Initial release
