#
# WARNING: This configuration file should only be used as an example for
# reference. Values in this configuration are nonsensical with the purpose of
# exemplifying various configuration blocks and how they are translated into
# Terraform configuration files.
#

log_level = "info"
port = 8555

consul {
  address = "consul.example.com"
}

task {
  name = "my-task"
  description = "automate services for website X"
  module = "namespace/example/module"
  version = "1.1.0"
  providers = ["myprovider"]
  condition "services" {
    names = ["web", "api"]
  }
  variable_files = ["example.module.tfvars", "/path/to/example.module.tfvars"]
}

driver "terraform" {
  required_providers {
    myprovider = {
      source = "namespace/myprovider"
      version = "1.3.0"
    }
  }
}

terraform_provider "myprovider" {
  address = "myprovider.example.com"
  username = "admin"
  attr = "foobar"
}
