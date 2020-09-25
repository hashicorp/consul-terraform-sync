# Configure Consul NIA

## CLI

There is not a default configuration that can be defined to run the Consul NIA daemon in a useful manner. One of the flags to set the config path is required.

```
$ consul-nia -h
Usage of consul-nia:
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
  -once
      Render templates and run tasks once. Does not run the process as a daemon
      and disables wait timers.
  -version
      Print the version of this daemon.
```

## Configuration

The Consul NIA daemon is configured using a file. Consul NIA supports [HashiCorp Configuration Language](https://github.com/hashicorp/hcl) (HCL) and JSON configuration file formats. An example HCL configuration is shown below to automate a task to execute a Terraform module for 2 services.

```hcl
log_level = "info"

consul {
  address = "consul.example.com"
}

buffer_period {
  min = "5s"
  max = "20s"
}

service {
  name = "api"
  datacenter = "dc1"
}

task {
  name = "website-x"
  description = "automate services for website-x"
  source = "namespace/example/module"
  version = "1.0.0"
  providers = ["myprovider"]
  services = ["web", "api"]
  wait {
    min = "10s"
  }
}

driver "terraform" {
  required_providers {
    myprovider = {
      source = "namespace/myprovider"
      version = "1.3.0"
    }
  }
}

provider "myprovider" {
  address = "myprovider.example.com"
}
```

### Consul NIA
Top level options are reserved for configuring Consul NIA.

```hcl
log_level = "INFO"
port = null
inspect_mode = false
syslog {}
buffer_period {
  enabled = true
  min = "5s"
  max = "20s"
}
```

* `log_level` - `(string: "WARN")` The log level to use for Consul NIA logging.
* `port` - `(int: <none>)` Optional port that the daemon binds on to serve its API.
* `inspect_mode` - `(bool: false)` Run Consul NIA in Inspect mode to print the current and proposed state change, and then exits. No changes are applied in this mode. This option is interchangeable with the `--inspect` CLI flag.
* `syslog` - Specifies the syslog server for logging.
  * `enabled` - `(bool: false)` Enable syslog logging. Specifying other option also enables syslog logging.
  * `facility` - `(string: <none>)` Name of the syslog facility to log to.
  * `name` - `(string: "consul-nia")` Name to use for the Consul NIA process when logging to syslog.
* `buffer_period` - Configures the default buffer period for all [tasks](#task) to dampen the affects of flapping services to downstream network devices. It defines the minimum and maximum amount of time to wait for the cluster to reach a consistent state and accumulate changes before triggering task executions. The default is enabled to reduce the number of times downstream infrastructure is updated within a short period of time. This is useful to enable in systems that have a lot of flapping.
  * `enabled` - `(bool: true)` Enable or disable buffer periods globally. Specifying `min` will also enable it.
  * `min` - `(string: 5s)` The minimum period of time to wait after changes are detected before triggering related tasks.
  * `max` - `(string: 20s)` The maximum period of time to wait after changes are detected before triggering related tasks.

### Consul

The `consul` block is used to connect with Consul to query the Consul Catalog and Consul KV.

```hcl
consul {
  address = "consul.example.com"
  auth {}
  tls {}
  token = null
  transport {}
}
```

* `address` - `(string: "localhost:8500")` Address is the address of the Consul server. It may be an IP or FQDN.
* `auth` - Auth is the HTTP basic authentication for communicating with Consul.
  * `enabled` - `(bool: false)`
  * `username` - `(string: <none>)`
  * `password` - `(string: <none>)`
* `tls` - Configure TLS to use a secure client connection with Consul. This requires Consul to be configured to serve HTTPS.
  * `enabled` - `(bool: false)` Enable TLS. Specifying any option for TLS will also enable it.
  * `verify` - `(bool: true)` Enables TLS peer verification. The default is enabled, which will check the global CA chain to make sure the given certificates are valid. If you are using a self-signed certificate that you have not added to the CA chain, you may want to disable SSL verification. However, please understand this is a potential security vulnerability.
  * `key` - `(string: <none>)` The client key file to use for talking to Consul over TLS. The key also be provided through the `CONSUL_CLIENT_KEY` environment variable.
  * `ca_cert` - `(string: <none>)` The CA file to use for talking to Consul over TLS. Can also be provided though the `CONSUL_CACERT` environment variable.
  * `ca_path` - `(string: <none>)` The path to a directory of CA certs to use for talking to Consul over TLS. Can also be provided through the `CONSUL_CAPATH` environment variable.
  * `cert` - `(string: <none>)` The client cert file to use for talking to Consul over TLS. Can also be provided through the `CONSUL_CLIENT_CERT` environment variable.
  * `server_name` - `(string: <none>)` The server name to use as the SNI host when connecting via TLS. Can also be provided through the `CONSUL_TLS_SERVER_NAME` environment variable.
* `token` - `(string: <none>)` The ACL token to use for client communication with the local Consul agent. The token can also be provided through the `CONSUL_TOKEN` or `CONSUL_HTTP_TOKEN` environment variables.
* `transport` - Transport configures the low-level network connection details.
  * `dial_keep_alive` - `(string: "30s")` The amount of time for keep-alives.
  * `dial_timeout` - `(string: "30s")` The amount of time to wait to establish a connection.
  * `disable_keep_alives` - `(bool: false)` Determines if keep-alives should be used. Disabling this significantly decreases performance.
  * `idle_conn_timeout` - `(string: "90s")` The timeout for idle connections.
  * `max_idle_conns` - `(int: 100)` The maximum number of total idle connections.
  * `max_idle_conns_per_host` - `(int: 1)` The maximum number of idle connections per remote host.
  * `tls_handshake_timeout` - `(string: "10s")` amount of time to wait to complete the TLS handshake.

### Service
A `service` block defines the explicit configuration for Consul NIA to monitor a service. This block may be specified multiple times to configure multiple services. For a service to be included in task automation, the service name or ID must be included in the `task.services` field of a [`task` block](#task). A service can be implicitly declared with default values by omitting the `service` block for that service and referenced by its name in the `task.services` field.

```hcl
service {
  id = null
  name = "web"
  datacenter = null
  description = "all instances of the service web"
  namespace = null
}
```

* `datacenter` - `(string: <none>)` The datacenter the service is deployed in.
* `description` - `(string: <none>)` The human readable text to describe the service.
* `id` - `(string: <none>)` ID identifies the service for Consul NIA. This is used to explicitly identify the service config for a task to use. If no ID is provided, the service is identified by the service name within a [task definition](#task).
* `name` - `(string: <required>)` The Consul logical name of the service (required).
* `namespace` <EnterpriseAlert inline /> - `(string: "default")` The namespace of the service. If not provided, the namespace will be inferred from the Consul NIA ACL token, or default to the `default` namespace.
* `tag` - `(string: <none>)` Tag is used to filter nodes based on the tag for the service.

### Task
A `task` block configures which task to run in automation for the selected services. The list of services can include services explicitly defined by a `service` block or implicitly declared by the service name. The `task` block may be specified multiple times to configure multiple tasks.

```hcl
task {
  name = "taskA"
  description = ""
  providers = []
  services = ["web", "api"]
  source = "org/example/module"
  version = "1.0.0"
  variable_files = []
}
```

* `description` - `(string: <none>)` The human readable text to describe the service.
* `name` - `(string: <required>)` Name is the unique name of the task (required).
* `providers` - `(list(string): [])` Providers is the list of provider names the task is dependent on. This is used to map [provider configuration](#provider) to the task.
* `services` - `(list(string): [])` Services is the list of service IDs or logical service names the task executes on. Consul NIA monitors the Consul Catalog for changes to these services and triggers the task to run. Any service value not explicitly defined by a `service` block with a matching ID is assumed to be a logical service name in the default namespace (enterprise).
* `source` - `(string: <required>)` Source is the location the driver uses to fetch dependencies. The source format is dependent on the driver. For the [Terraform driver](#terraform-driver), the source is the module path (local or remote). Read more on [Terraform module source here](https://www.terraform.io/docs/modules/sources.html).
* `variable_files` - `(list(string): [])` A list of paths to files containing variables for the task. For the [Terraform driver](#terraform-driver), these are files with `.tfvars` file extensions and are used as Terraform [input variables](https://www.terraform.io/docs/configuration/variables.html) to pass as arguments to the Terraform module. Variables are loaded in the same order as they appear in the order of the files. Duplicate variables are overwritten with the later value.
* `version` - `(string: <none>)` The version of the provided source the task will use. For the [Terraform driver](#terraform-driver), this is the module version. The latest version will be used as the default if omitted.
* `buffer_period` - Configures the buffer period for the task to dampen the affects of flapping services to downstream network devices. It defines the minimum and maximum amount of time to wait for the cluster to reach a consistent state and accumulate changes before triggering task execution. The default is inherited from the top level [`buffer_period` block](#consul-nia). If configured, these values will take precedence over the global buffer period. This is useful to enable for a task that is dependent on services that have a lot of flapping.
  * `enabled` - `(bool: false)` Enable or disable buffer periods for this task. Specifying `min` will also enable it.
  * `min` - `(string: 5s)` The minimum period of time to wait after changes are detected before triggering related tasks.
  * `max` - `(string: 20s)` The maximum period of time to wait after changes are detected before triggering related tasks.

### Terraform Driver
The `driver` block configures the subprocess for Consul NIA to propagate infrastructure change. The default Terraform driver does not need to be explicitly configured.

```hcl
driver "terraform {
  log = false
  persist_log = false
  path = ""
  working_dir = ""
  
  backend "consul" {
    scheme = "https"
  }

  required_providers {
    myprovider = {
      source = "namespace/myprovider"
      version = "1.3.0"
    }
  }
}
```

* `backend` - `(obj: optional)` The backend stores [Terraform state files](https://www.terraform.io/docs/state/index.html) for each task. This option is similar to the [Terraform backend configuration](https://www.terraform.io/docs/backends/types/consul.html). Consul backend is the only supported backend at this time. If omitted, Consul NIA will use default values and use configuration from the [`consul` block](#consul). The Consul KV path is the base path to store state files for Consul NIA tasks. The full path of each state file will have the task identifer appended to the end of the path, e.g. `consul-nia/terraform-env:task-name`.
* `log` - `(bool: false)` Enable all Terraform output (stderr and stdout) to be included in the Consul NIA log. This is useful for debugging and development purposes. It may be difficult to work with log aggregators that expect uniform log format.
* `path` - `(string: optional)` The file path to install Terraform or discover an existing Terraform binary. If omitted, Consul NIA will install Terraform in the same directory as the daemon.
* `persist_log` - `(bool: false)` Enable trace logging for each Terraform client to disk per task. This is equivalent to setting `TF_LOG_PATH=<work_dir>/terraform.log`. Trace log level results in verbose logging and may be useful for debugging and development purposes. We do not recommend enabling this for production. There is no log rotation and may quickly result in large files.
* `required_providers` - `(obj)` Declare each Terraform providers used across all tasks. This is similar to the [Terraform `terraform.required_providers`](https://www.terraform.io/docs/configuration/provider-requirements.html#requiring-providers) field to specify the source and version for each provider. Consul NIA will process these requirements when preparing each task that uses the provider.
* `working_dir` - `(string: "nia-tasks")` The base working directory to manage Terraform configurations all tasks. The full path of each working directory will have the task identifier appended to the end of the path, e.g. `./nia-tasks/task-name`.

### Provider
A `provider` block configures the options to interface with network infrastructure. Define a block for each provider required by the set of Terraform modules across all tasks. This block resembles [provider blocks for Terraform configuration](https://www.terraform.io/docs/configuration/providers.html). To find details on how to configure a provider, refer to the corresponding documentation for the Terraform provider. The main directory of publicly available providers are hosted on the [Terraform Registry](https://registry.terraform.io/browse/providers).

The below configuration captures the general design of defining a provider using the [Vault Terraform provider](https://registry.terraform.io/providers/hashicorp/vault/latest/docs) as an example.

```hcl
driver "terraform" {
  required_providers {
    vault = {
      source = "hashicorp/vault"
      version = "2.13.0"
    }
  }
}

provider "vault" {
  address = "vault.example.com"
}

task {
  source = "some/source"
  providers = ["vault"]
  services = ["web", "api"]
}
```
