# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform_provider "tf_providerC" { }

task {
  name = "taskB"
  condition "services" {
    names = ["serviceC", "serviceD"]
  }
}
