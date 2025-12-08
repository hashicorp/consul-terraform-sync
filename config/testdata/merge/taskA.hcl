# Copyright IBM Corp. 2020, 2025
# SPDX-License-Identifier: MPL-2.0

terraform_provider "tf_providerA" { }

terraform_provider "tf_providerB" { }

task {
  name = "taskA"
  condition "services" {
    names = ["serviceA", "serviceB"]
  }
}
