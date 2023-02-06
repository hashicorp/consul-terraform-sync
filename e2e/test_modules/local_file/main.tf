# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "local_file" "address" {
  content  = join("\n", [for s in var.services : join("\t", [s.name, s.address])])
  filename = "services.txt"
}
