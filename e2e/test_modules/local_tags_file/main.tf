# Copyright IBM Corp. 2020, 2025
# SPDX-License-Identifier: MPL-2.0

# Creates a local file for each service containing a list of the tags
resource "local_file" "tags" {
  for_each = var.catalog_services
  content  = join(",", each.value)
  filename = "resources/${each.key}_tags.txt"
}
