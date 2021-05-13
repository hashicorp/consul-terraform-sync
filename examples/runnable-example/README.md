
# Runnable Example
This is an eample of how to run Consul-Terraform-Sync with Consul. It uses the Terraform provider [local](https://registry.terraform.io/providers/hashicorp/local/latest), which manages local resources. In this case, it will be used to create local files.

This example is composed of:
- Consul-Terraform-Sync configuration
- Terraform module that is Consul-Terraform-Sync compatible
- Consul configuration

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

The module executed by the task will create a file the task that has information about the services. It will be created as `test.txt` by default.
```
$ cat sync-tasks/example-task/test.txt
api api 127.0.0.1
web web 127.0.0.1
```

Deregister the one of the services in Consul. This can be done using the Consul CLI or the Consul API.
```
$ consul services deregister -id=api
```

Consul-Terraform-Sync will automatically detect the change and rerun the task for that service, which will in turn update the `test.txt` file with the latest information about the Consul services.
```
$ cat sync-tasks/example-task/test.txt
web web 127.0.0.1 test_value
```
