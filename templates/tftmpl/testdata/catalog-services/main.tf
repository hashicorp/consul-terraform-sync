# This file is generated by Consul-Terraform-Sync.
#
# The HCL blocks, arguments, variables, and values are derived from the
# operator configuration for Consul-Terraform-Sync. Any manual changes to
# this file may not be preserved and could be overwritten by a subsequent
# update.
#
# Task: test
# Description: user description for task named 'test'

terraform {
  required_version = ">= 0.13.0, < 1.6.0"
}


# user description for task named 'test'
module "test" {
  source           = "namespace/consul-terraform-sync/consul//modules/test"
  version          = "0.0.0"
  services         = var.services
  catalog_services = var.catalog_services
}
