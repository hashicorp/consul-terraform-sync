variable "services" {
  description = "Consul services monitored by Consul Terraform Sync"
  type = map(
    object({
      id      = string
      name    = string
      address = string
    })
  )
}
