resource "local_file" "address" {
    for_each = local.flattened_services
    content = each.value.address.address
    filename = "../resources/consul_service_${each.value.name}.txt"
}

locals {
  # List of services to each of its known IP addresses
  flattened_services = {
    for s in flatten([
      for name, service in var.services : [
        for i in range(length(service.addresses)) : {
          name        = "${service.name}.${i}"
          description = service.description
          address     = service.addresses[i]
        }
      ]
    ]) : s.name => s
  }
}
