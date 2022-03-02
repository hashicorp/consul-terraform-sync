resource "local_file" "address" {
  depends_on = [local_file.greeting_services]
  for_each   = var.services
  content    = each.value.address
  filename   = "resources/${each.value.id}.txt"
}

resource "local_file" "greeting_services" {
  content = "${join("\n", [
    for _, service in var.services : "Hello, ${service.name}!"
  ])}\n"
  filename = "services_greetings.txt"
  provisioner "local-exec" {
    command = "sleep ${var.delay}"
  }
}
