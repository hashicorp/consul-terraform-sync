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

  # Suppressing Go standard library CVEs that are fixed in Go 1.24.8/1.24.9
  # but require staying on Go 1.23.10 for E2E test stability.
  # These CVEs are related to stdlib and will be addressed in a future release
  # when upgrading to a stable Go 1.24.x or 1.25.x with working E2E tests.
  triage {
    suppress {
      vulnerabilities = [
        "GO-2025-4013",  # crypto/x509: Panic with DSA public keys - Fixed in go1.24.8
        "GO-2025-4012",  # net/http: Cookie parsing memory exhaustion - Fixed in go1.24.8
        "GO-2025-4011",  # encoding/asn1: DER parsing memory exhaustion - Fixed in go1.24.8
        "GO-2025-4010",  # net/url: IPv6 hostname validation - Fixed in go1.24.8
        "GO-2025-4009",  # encoding/pem: Quadratic complexity - Fixed in go1.24.8
        "GO-2025-4008",  # crypto/tls: ALPN negotiation error - Fixed in go1.24.8
        "GO-2025-4007",  # crypto/x509: Name constraints quadratic complexity - Fixed in go1.24.9
        "GO-2025-3956",  # os/exec: LookPath unexpected paths - Fixed in go1.23.12
      ]
    }
  }
}
