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

resource "local_file" "consul_services" {
  content = "${join("\n", [
    for _, service in var.services : "${service.name} ${service.id} ${service.node_address} ${lookup(service.cts_user_defined_meta, service.name, "")}"
  ])}\n"
  filename = var.filename
}
