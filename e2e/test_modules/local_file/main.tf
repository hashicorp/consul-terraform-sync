# Copyright IBM Corp. 2020, 2025
# SPDX-License-Identifier: MPL-2.0

resource "local_file" "address" {
  content  = join("\n", [for s in var.services : join("\t", [s.name, s.address])])
  filename = "services.txt"
}
