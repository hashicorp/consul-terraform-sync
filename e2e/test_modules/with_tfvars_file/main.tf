terraform {
  required_providers {
    # Provider source is used for Terraform discovery and installation of
    # providers. Declare source for all providers required by the module.
    local = {
      source  = "hashicorp/local"
      version = ">= 2.1.0"
    }
  }
}

resource "local_file" "obj" {
  content  = join(",", var.sample.tags, [for key, value in var.sample.user_meta : "${key}=${value}"])
  filename = var.filename
}