# Example Consul Terraform Sync Compatible Module

This is an example of a module that is compatible with Consul Terraform Sync, which means it contains a root module (`main.tf`) and a `services` input variable, as specified in `variables.tf`.

## Features

This module writes to a file the name, id, and IP address for all the Consul services. It also writes a metadata value if provided for the service.

## Requirements
### Terraform Providers

| Name | Version |
|------|---------|
| local | 2.1.0 |


## Usage
| User-defined service meta | Required | Description |
|-------------------|----------|-------------|
| test_key | false | Test metadata that is printed out for the service |

**User Config for Consul Terraform Sync**

example.hcl
```hcl
task {
  name = "example-task"
  description = "Writes the service name, id, and IP address to a file"
  module = "../../example-module"
  providers = ["local"]
  condition "services" {
    names = ["web", "api"]
    cts_user_defined_meta = {
      "api" = "api_meta"
      "web" = "web_meta"
    }
  }
  variable_files = [/path/to/task-example.tfvars]
}

driver "terraform" {
  required_providers {
    local = {
      source = "hashicorp/local"
      version = "2.1.0"
    }
  }
}

terraform_provider "local" {
}

```

**Variable file**

Optional input variable file defined by a user for the task above.

| Input Variables | Default Value | Description |
|-------------------|----------|-------------|
| filename | test.txt | Name of file that is created with the service information |

```hcl
# task-example.tfvars

filename = "consul_services.txt"
```
