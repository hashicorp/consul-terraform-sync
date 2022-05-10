resource "local_file" "bar" {
  content  = data.local_file.foo.content
  filename = "resources/bar.txt"
}

data "local_file" "foo" {
  filename = "foo.txt"
}

resource "local_file" "address" {
  for_each = var.services
  content  = each.value.address
  filename = "resources/${each.value.id}.txt"
}
