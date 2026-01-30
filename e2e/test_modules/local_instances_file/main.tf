# Copyright IBM Corp. 2020, 2025
# SPDX-License-Identifier: MPL-2.0

resource "local_file" "address" {
  for_each = var.services
  content  = each.value.address
  filename = "resources/${each.value.id}.txt"
}
