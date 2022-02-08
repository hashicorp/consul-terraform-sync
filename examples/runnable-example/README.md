
# Runnable Example
This is an example of how to run Consul-Terraform-Sync with Consul. It uses the Terraform provider [local](https://registry.terraform.io/providers/hashicorp/local/latest), which manages local resources. In this case, it will be used to create local files.

This example is composed of:
- Consul-Terraform-Sync configuration
- Terraform module that is Consul-Terraform-Sync compatible
- Consul configuration

## Requirements
- Consul-Terraform-Sync
- Consul

This example assumes that the `consul-terraform-sync` and `consul` binaries are installed on the user's path. See the installation steps for [Consul-Terraform-Sync](https://github.com/hashicorp/consul-terraform-sync#installation) and [Consul](https://learn.hashicorp.com/tutorials/consul/get-started-install) for more details.

Terraform is not a requirement, as it is [installed by Consul-Terraform-Sync](https://www.consul.io/docs/nia/network-drivers#understanding-terraform-automation).

## Demo

Start Consul agent in dev mode, using the provided Consul config directory. This will register multiple services to Consul.

```
$ consul agent -dev -config-dir=./consul.d
```

Start Consul-Terraform-Sync using the provided config file.
```
$ consul-terraform-sync -config-file=config.hcl
```

Consul-Terraform-Sync will create a `sync-tasks/example-task` directory for the one example task specified in the config file and run the task.

The module executed by the task will create a file that has information about the services. It will be created as `test.txt` by default. Note that one of the services has an additional metadata value, as specified in the `config.hcl`, which is also printed out by the module.
```
$ cat sync-tasks/example-task/test.txt
api api 127.0.0.1 api_meta
web web 127.0.0.1 web_meta
```

Deregister one of the services in Consul. This can be done using the Consul CLI or the Consul API.
```
$ consul services deregister -id=api
```

Consul-Terraform-Sync will automatically detect the change and rerun the task for that service, which will in turn update the `test.txt` file with the latest information about the Consul services.
```
$ cat sync-tasks/example-task/test.txt
web web 127.0.0.1 web_meta
```
