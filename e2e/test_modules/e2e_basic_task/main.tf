resource "local_file" "address" {
    for_each = var.services
    content = each.value.address
    filename = "../resources/consul_service_${each.value.id}.txt"
}
