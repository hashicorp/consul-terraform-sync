resource "local_file" "address" {
  for_each = var.services
  content  = each.value.address
  filename = "resources/${each.value.id}.txt"
}
