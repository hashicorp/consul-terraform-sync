# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

container {
  dependencies = true
  alpine_secdb = false
  secrets      = true
}

binary {
  go_modules = true
  osv        = true
  oss_index  = true
  nvd        = false
  secrets {
    matchers {
      known = ["tfc", "hcp", "tfe", "github", "artifactory", "slack", "aws", "google", "azure"]
    }
  }
  triage {
    suppress {
      vulnerabilities = [
          "GO-2026-5932", #supressing as there is no resolution for this CVE.
      ]
    }
  }
}
