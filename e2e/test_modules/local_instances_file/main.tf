# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "local_file" "address" {
  for_each = var.services
  content  = each.value.address
  filename = "resources/${each.value.id}.txt"
}
