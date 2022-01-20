# Required var.services declaration
variable "services" {
  description = "Consul services monitored by Consul Terraform Sync"
  # Optional attributes of services
  type = map(
    object({
      id        = string
      name      = string
      kind      = string
      address   = string
      port      = number
      meta      = map(string)
      tags      = list(string)
      namespace = string
      status    = string

      node                  = string
      node_id               = string
      node_address          = string
      node_datacenter       = string
      node_tagged_addresses = map(string)
      node_meta             = map(string)

      cts_user_defined_meta = map(string)
    })
  )
}

variable "filename" {
  type    = string
  default = "test.txt"
}

variable "sample" {
  type = object({
    user_meta = map(string)
    tags      = list(string)
  })


  default = {
    user_meta = {
      name            = "unknown"
      age             = "unknown"
      day_of_the_week = "unknown"
    }
    tags = ["happy", "sad"]
  }
}
