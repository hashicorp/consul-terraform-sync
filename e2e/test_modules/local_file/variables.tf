# Copyright IBM Corp. 2020, 2025
# SPDX-License-Identifier: MPL-2.0

variable "services" {
  description = "Consul services monitored by Consul-Terraform-Sync"
  type = map(
    object({
      id      = string
      name    = string
      address = string
    })
  )
}
