# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

container {
  dependencies = true
  alpine_secdb = false
  secrets      = true

  # Triage items that are _safe_ to ignore here. Note that this list should be
  # periodically cleaned up to remove items that are no longer found by the scanner.
  triage {
    suppress {
      vulnerabilities = [
        "GHSA-v6v8-xj6m-xwqh", # github.com/hashicorp/go-retryablehttp@v0.7.0
        "GO-2024-2947", # github.com/hashicorp/go-retryablehttp@v0.7.0
        "GO-2025-3533" # github.com/getkin/kin-openapi@v0.94.0
      ]
    }
  }
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

  # Triage items that are _safe_ to ignore here. Note that this list should be
  # periodically cleaned up to remove items that are no longer found by the scanner.
  triage {
    suppress {
      vulnerabilities = [
        "GHSA-v6v8-xj6m-xwqh", # github.com/hashicorp/go-retryablehttp@v0.7.0
        "GO-2024-2947", # github.com/hashicorp/go-retryablehttp@v0.7.0
        "GO-2025-3533" # github.com/getkin/kin-openapi@v0.94.0
      ]
    }
  }
}
