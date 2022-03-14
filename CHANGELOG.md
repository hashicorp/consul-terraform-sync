## 0.5.2 (March 15, 2022)

BUG FIXES:

* Fix issue where triggering a task while task is running is ignored and can cause locked state [[GH-732](https://github.com/hashicorp/consul-terraform-sync/issues/732)]
* Fix issue where re-creating a task using the task creation API/CLI after Consul restart can result in an error [[GH-701](https://github.com/hashicorp/consul-terraform-sync/issues/701)]
* Fix issue where creating a task with the same name as a deleted task, where the deleted task includes a `condition "schedule"`, can leave CTS in a bad state [[GH-715](https://github.com/hashicorp/consul-terraform-sync/issues/715)]

## 0.5.1 (February 24, 2022)

BUG FIXES:
* **(Enterprise Only)** Regression with Terraform Cloud driver's `required_providers` configuration which leads the driver to potentially attempt to retrieve an incorrect provider source and version [[GH-728](https://github.com/hashicorp/consul-terraform-sync/issues/728)]

## 0.5.0 (February 23, 2022)

KNOWN ISSUES: _See the linked GitHub issues for details on any workarounds_
* Re-creating a task using the task creation API/CLI after Consul restart can result in an error. [[GH-701](https://github.com/hashicorp/consul-terraform-sync/issues/701)]
* Creating a task with the same name as a deleted task, where the deleted task includes a `condition "schedule"`, can leave CTS in a bad state [[GH-715](https://github.com/hashicorp/consul-terraform-sync/issues/715)]
* Creating a task with multiple module inputs using JSON requires specific JSON format. [[GH-714](https://github.com/hashicorp/consul-terraform-sync/issues/714)]
* **(Enterprise Only)** Regression with Terraform Cloud driver's `required_providers` configuration which leads the driver to potentially attempt to retrieve an incorrect provider source and version [[GH-728](https://github.com/hashicorp/consul-terraform-sync/issues/728)]

BREAKING CHANGES:
* Stop monitoring and including non-passing service instances in `terraform.tfvars` by default. CTS should only monitor passing service instances unless configured otherwise. [[GH-430](https://github.com/hashicorp/consul-terraform-sync/issues/430)]
* Removed `driver.terraform.working_dir` configuration option that was deprecated in v0.3.0. Use top-level `working_dir` to configure parent directory for all tasks or `task.working_dir` to configure per task. [[GH-548](https://github.com/hashicorp/consul-terraform-sync/pull/548)]
* Require that `condition "catalog-services"` block's `regexp` field be configured instead of relying on previous default behavior. [[GH-574](https://github.com/hashicorp/consul-terraform-sync/pull/574)]
* Change default value of `source_includes_var` field from false to true for all types of `condition` blocks. [[GH-578](https://github.com/hashicorp/consul-terraform-sync/pull/578)]

FEATURES:
* Support for deleting an existing task through the API and CLI. [[GH-522](https://github.com/hashicorp/consul-terraform-sync/issues/522)]
* Support for creating a task through the API and CLI. [[GH-522](https://github.com/hashicorp/consul-terraform-sync/issues/522)]
* Support for retrieving information about an existing task through the API. [[GH-681](https://github.com/hashicorp/consul-terraform-sync/pull/681)]
* Support for `-auto-approve` option to skip interactive approval prompts in the CLI. [[GH-576](https://github.com/hashicorp/consul-terraform-sync/pull/576)]

IMPROVEMENTS:
* Tasks will now trigger only after the `min` time of the `buffer_period` configuration has transpired. Previously tasks would trigger immediately, once, before honoring the minimum buffer period time. [[GH-642](https://github.com/hashicorp/consul-terraform-sync/pull/642)]
* Support configuring a task's `condition "services"` and `source_input "services"` blocks with query parameters: `datacenter`, `namespace`, `filter`, and `cts_user_defined_meta`. [[GH-357](https://github.com/hashicorp/consul-terraform-sync/issues/357)]
* Support new `names` field for configuring a task's `condition "services"` and `source_input "services"` blocks as an optional alternative to `regexp` to list monitored services by name. [[GH-561](https://github.com/hashicorp/consul-terraform-sync/issues/561)]
* Support `source_includes_var` field for a task's `condition "services"` block. [[GH-584](https://github.com/hashicorp/consul-terraform-sync/pull/584)]
* Expand `source_input` block usage to dynamic tasks and support multiple `source_input` blocks per task. [[GH-607](https://github.com/hashicorp/consul-terraform-sync/issues/607)]
* Initialize disabled tasks, which allows for earlier validation of a task. [[GH-625](https://github.com/hashicorp/consul-terraform-sync/pull/625)]

DEPRECATIONS:
* **(Enterprise Only)** Deprecate `workspace_prefix` for the Terraform Cloud driver that adds unexpected `-` character added between the prefix and task name. Use the new `workspaces.prefix` option instead. [[GH-442](https://github.com/hashicorp/consul-terraform-sync/issues/442)]
  * If the `workspace_prefix` option is in use by CTS v0.3.x or v0.4.x for the Terraform Cloud driver, visit [GH-442](https://github.com/hashicorp/consul-terraform-sync/issues/442) for upgrade guidelines.
* Deprecate `port` CLI option. Use `http-addr` option instead. [[GH-617](https://github.com/hashicorp/consul-terraform-sync/pull/617)]
* Deprecate `services` field in task configuration. Use `condition "services"` or `source_input "services"` instead. [[GH-669](https://github.com/hashicorp/consul-terraform-sync/issues/669)]
* Deprecate `service` block in configuration. To replace usage, first upgrade the associated task config `services` field to a `condition "services"` or `source_input "services"`. Then move the `service` block fields into the new `condition` or `source_input` block. [[GH-670](https://github.com/hashicorp/consul-terraform-sync/issues/670)]
* Deprecate `source` field in task configuration. Rename `source` to `module` in configuration. [[GH-566](https://github.com/hashicorp/consul-terraform-sync/issues/566)]
* Deprecate `source_input` block in task configuration. Rename `source_input` to `module_input` in configuration. [[GH-567](https://github.com/hashicorp/consul-terraform-sync/issues/567)]
* Deprecate `source_includes_var` field in task configuration's `condition` block. Rename `source_includes_var` to `use_as_module_input` in configuration. [[GH-568](https://github.com/hashicorp/consul-terraform-sync/issues/568)]
* Deprecate non-status, config-related information from the Task Status API's response payload. Use the Get Task API instead to retrieve the deprecated information. [[GH-569](https://github.com/hashicorp/consul-terraform-sync/issues/569)]

BUG FIXES:
* Fix CLI client enable task command timing out when buffer period is enabled [[GH-516](https://github.com/hashicorp/consul-terraform-sync/issues/516)]
* Fix Services condition when configured with regex and Catalog Services condition to use cached indexes for Consul API blocking queries. [[GH-529](https://github.com/hashicorp/consul-terraform-sync/issues/529)]
* **(Enterprise Only)** Fix task not triggering after re-enabled when running with the TFC driver.

## 0.4.3 (January 14, 2022)
SECURITY:
* Upgrade Go to address [CVE-2021-44716](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-44716) and [CVE-2021-44717](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-44717)

FEATURES:
* Support for Terraform v1.1 [[GH-588](https://github.com/hashicorp/consul-terraform-sync/pull/588)]

BUG FIXES:
* Fix issue where enabling a task can display an EOF error in terminal even though task is enabled. [[GH-516](https://github.com/hashicorp/consul-terraform-sync/issues/516)]

## 0.4.2 (November 22, 2021)
KNOWN ISSUES:
* Enabling a task can display an EOF error in terminal even though task is enabled. [[GH-516](https://github.com/hashicorp/consul-terraform-sync/issues/516)]

FEATURES:
* Support TLS and mutual TLS for the CTS API and CLI. [[GH-466](https://github.com/hashicorp/consul-terraform-sync/issues/466)]
* **(Enterprise Only)** Add Terraform Cloud workspace tagging support to add, require, and restrict tags with new `driver.terraform-cloud.workspaces` options.

IMPROVEMENTS:
* **(Enterprise Only)** Add default address for the Terraform Cloud driver to https://app.terraform.io.

BUG FIXES:
* Fix Services condition when configured with regex, Consul KV condition, and Catalog Services condition to use Consul API blocking queries for more efficient monitoring of changes on Consul. [[GH-460](https://github.com/hashicorp/consul-terraform-sync/pull/460), [GH-467](https://github.com/hashicorp/consul-terraform-sync/pull/467)]
* Fix issue where choosing to cancel when using the Enable CLI still enabled the task. [[GH-451](https://github.com/hashicorp/consul-terraform-sync/issues/451)]
* Fix issue where Update Task API unexpectedly updated the task when running with inspect mode. [[GH-465](https://github.com/hashicorp/consul-terraform-sync/issues/465)]
* Fix panic when there is a Terraform validation warning related to provider blocks when CTS runs Terraform CLI v0.15+. [[GH-473](https://github.com/hashicorp/consul-terraform-sync/issues/473)]
* **(Enterprise Only)** Fix issue where configured service block filter was not being used to filter monitored service instances when using the Terraform Cloud driver. [[GH-454](https://github.com/hashicorp/consul-terraform-sync/issues/454)]
* **(Enterprise Only)** Fix issue where using the Terraform Cloud driver with a scheduled task configured with a consul-kv source input did not run at the scheduled time when only the monitored services had changed. [[GH-502](https://github.com/hashicorp/consul-terraform-sync/issues/502)]

## 0.4.1 (November 03, 2021)

BUG FIXES:
* Compile 0.4.0 binaries with statically linked C bindings. [[GH-475](https://github.com/hashicorp/consul-terraform-sync/pull/475)]
* Fix 0.4.0 docker image to use static binaries. [[GH-474](https://github.com/hashicorp/consul-terraform-sync/issues/474)]

## 0.4.0 (October 13, 2021)
KNOWN ISSUES:
* The formatting of the Terraform Plan outputted in the terminal by the Enable CLI and Inspect Mode is difficult to read when used with the TFC driver for certain Terraform versions. See issue for workaround. [[GH-425](https://github.com/hashicorp/consul-terraform-sync/issues/425)]

BREAKING CHANGES:
* Remove deprecated `tag` filtering option from `service` configuration, which has been replaced by the more general `filter` option. [[GH-312](https://github.com/hashicorp/consul-terraform-sync/issues/312)]
* The logging timestamps are now reported using the timezone of the system CTS is running on, instead of defaulting to UTC time. [[GH-332](https://github.com/hashicorp/consul-terraform-sync/issues/332)]

FEATURES:
* Add support for triggering tasks based on Consul KV changes. [[GH-150](https://github.com/hashicorp/consul-terraform-sync/issues/150)]
* **(Enterprise Only)** Add integration with Terraform Cloud remote operations through the Terraform Cloud driver. [[GH-328](https://github.com/hashicorp/consul-terraform-sync/issues/328)]
* Add support for triggering a task on schedule for a task module requiring only services input. Supports a new schedule condition that is configured in conjunction with `task.services`. [[GH-308](https://github.com/hashicorp/consul-terraform-sync/issues/308)]
* Add support for new services source input block which can be used in conjunction with the scheduled task trigger `task.source_input "services"`. This allows for service regex to be defined in lieu of `task.services`. [[GH-382](https://github.com/hashicorp/consul-terraform-sync/issues/382)]
* Add support for new Consul KV source input block which can be used in conjunction with the scheduled task trigger `task.source_input "consul-kv"`. This allows for Consul key-values to be used as input to the Terraform Module. [[GH-389](https://github.com/hashicorp/consul-terraform-sync/issues/389)]

IMPROVEMENTS:
* **(Enterprise Only)** Add TLS configuration for the Terraform Cloud driver when connecting with Terraform Enterprise.
* Enhanced http and structured logging. [[GH-332](https://github.com/hashicorp/consul-terraform-sync/issues/332)]

BUG FIXES:
* Enforce GET request method for Overall Status API (`/v1/status`) so that other methods return 405 Method Not Allowed. [[GH-427](https://github.com/hashicorp/consul-terraform-sync/issues/427)]
* Enforce GET request method for Task Status API (`/v1/status/tasks`) so that other methods return 405 Method Not Allowed. [[GH-360](https://github.com/hashicorp/consul-terraform-sync/issues/360)]

## 0.3.1 (January 14, 2022)
SECURITY:
* Upgrade Go to address [CVE-2021-44716](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-44716) and [CVE-2021-44717](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-44717)

FEATURES:
* Support for Terraform v1.1 [[GH-588](https://github.com/hashicorp/consul-terraform-sync/pull/588)]

## 0.3.0 (September 01, 2021)
BREAKING CHANGES:
* `INFO` log level is now the default, changed from `WARN`. [[GH-23](https://github.com/hashicorp/consul-terraform-sync/issues/23)]

FEATURES:
* (**Beta Feature**) Add regex support for service triggers. This feature currently does not support any query parameters [[GH-357](https://github.com/hashicorp/consul-terraform-sync/issues/357)], which includes any query parameters set in a service block. [[GH-299](https://github.com/hashicorp/consul-terraform-sync/issues/299), [GH-357](https://github.com/hashicorp/consul-terraform-sync/pull/353)]
* **(Enterprise Only)** Add integration with Terraform Enterprise remote operations by using the new Terraform Cloud driver. [[GH-327](https://github.com/hashicorp/consul-terraform-sync/issues/327)]
* **(Enterprise Only)** Add `task.terraform_version` configuration option to set the Terraform version used for the task's workspace on Terraform Enterprise.
* **(Enterprise Only)** Add `license_path` configuration option and `CONSUL_LICENSE` and `CONSUL_LICENSE_PATH` environment variables to check for valid, unexpired Consul license on start up and provide logging notification for expiration and termination events.

IMPROVEMENTS:
* Deprecate `driver.working_dir` configuration option to be removed in v0.5.0. Add new options to configure working directory for managing CTS generated artifacts. Top-level `working_dir` to configure parent directory for all tasks or `task.working_dir` to configure per task. [[GH-314](https://github.com/hashicorp/consul-terraform-sync/issues/314)]

BUG FIXES:
- Fix loading the `CONSUL_HTTP_ADDR` environment variable. [[GH-351](https://github.com/hashicorp/consul-terraform-sync/pull/351)]
- Fix issue where the task-level `buffer_period` configuration did not override the global-level `buffer_period` configuration when the task-level `buffer_period` was disabled. [[GH-359](https://github.com/hashicorp/consul-terraform-sync/pull/359)]

## 0.2.2 (January 13, 2022)
SECURITY:
* Upgrade Go to address [CVE-2021-44716](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-44716) and [CVE-2021-44717](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-44717)

## 0.2.1 (July 14, 2021)
FEATURES:
* Add support for Terraform v1.0 [[GH-333](https://github.com/hashicorp/consul-terraform-sync/pull/333)]

BUG FIXES:
- Fix missing event when task was enabled and executed using the CLI enable sub command [[GH-318](https://github.com/hashicorp/consul-terraform-sync/issues/318), [GH-319](https://github.com/hashicorp/consul-terraform-sync/issues/319)]
- Fix disabled tasks to trigger after re-enabling [[GH-320](https://github.com/hashicorp/consul-terraform-sync/issues/320)]

## 0.2.0 (June 22, 2021)
BREAKING CHANGES:
* Change task source for local modules to expect path based on directory where CTS is run instead of task directory. [[GH-264](https://github.com/hashicorp/consul-terraform-sync/issues/264),  [GH-283](https://github.com/hashicorp/consul-terraform-sync/pull/283)]
* Change the empty `namespace` value for `var.services` from `null` to empty string `""`. This effects CTS when used with Consul OSS, and no changes when used with Consul Enterprise where the default namespace value is `"default"`. [[GH-303](https://github.com/hashicorp/consul-terraform-sync/pull/303)]

FEATURES:
* Add support for Terraform v0.15 [[GH-277](https://github.com/hashicorp/consul-terraform-sync/pull/277)]
* Add support to only trigger a task on service registration (on first instance of a service registering) or on service deregistration (on last instance of a service deregistering) [[GH-307](https://github.com/hashicorp/consul-terraform-sync/issues/307)]
* Add support for filtering service nodes using a filter expression. Deprecate `tag` in favor of `filter`, where `tag` will be removed in CTS v0.4.0. [[GH-295](https://github.com/hashicorp/consul-terraform-sync/pull/295)]
* Execute Terraform validate after tasks are initialized [[GH-306](https://github.com/hashicorp/consul-terraform-sync/pull/306)]

BUG FIXES:
- Add support for relative paths for task variable files [[GH-279](https://github.com/hashicorp/consul-terraform-sync/issues/279), [GH-288](https://github.com/hashicorp/consul-terraform-sync/pull/288)]
- Fix Terraform installation issue when path is set to an empty string [[GH-212](https://github.com/hashicorp/consul-terraform-sync/issues/212), [GH-297](https://github.com/hashicorp/consul-terraform-sync/pull/297)]
- Fix missing event when task was enabled and executed using the CLI enable sub command [[GH-318](https://github.com/hashicorp/consul-terraform-sync/issues/318), [GH-319](https://github.com/hashicorp/consul-terraform-sync/issues/319)]

## 0.1.3 (July 14, 2021)
BUG FIXES:
- Fix missing event when task was enabled and executed using the CLI enable sub command [[GH-318](https://github.com/hashicorp/consul-terraform-sync/issues/318), [GH-319](https://github.com/hashicorp/consul-terraform-sync/issues/319)]
- Fix disabled tasks to trigger after re-enabling [[GH-320](https://github.com/hashicorp/consul-terraform-sync/issues/320)]

## 0.1.2 (April 28, 2021)

SECURITY:
* Update `tfinstall` to verify downloaded versions of Terraform with the rotated HashiCorp PGP signing key ([HCSEC-2021-12](https://discuss.hashicorp.com/t/hcsec-2021-12-codecov-security-event-and-hashicorp-gpg-key-exposure/23512)) [[GH-263](https://github.com/hashicorp/consul-terraform-sync/pull/263)]
* Update Docker release process with rotated HashiCorp signing key ([HCSEC-2021-12](https://discuss.hashicorp.com/t/hcsec-2021-12-codecov-security-event-and-hashicorp-gpg-key-exposure/23512)) [[GH-270](https://github.com/hashicorp/consul-terraform-sync/pull/270)]
* Update the fallback version of Terraform to download to v0.13.7 which was released with the rotated HashiCorp signing key ([HCSEC-2021-12](https://discuss.hashicorp.com/t/hcsec-2021-12-codecov-security-event-and-hashicorp-gpg-key-exposure/23512)) [[GH-271](https://github.com/hashicorp/consul-terraform-sync/pull/271)]

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
