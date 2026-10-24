# Copyright IBM Corp. 2020, 2026
# SPDX-License-Identifier: MPL-2.0

container {
  dependencies = true
  alpine_secdb = false
  secrets      = true
  triage {
    suppress {
      vulnerabilities = [
        "GO-2026-5932", // x/crypto/openpgp: no fixed version exists upstream; CTS imports no openpgp package.
      ]
    }
  }
}

binary {
  go_modules = true
  osv        = true
  oss_index  = true
  nvd        = false
  triage {
    suppress {
      vulnerabilities = [
        "GO-2026-5932", // x/crypto/openpgp: no fixed version exists upstream; CTS imports no openpgp package.
      ]
    }
  }
  secrets {
    matchers {
      known = ["tfc", "hcp", "tfe", "github", "artifactory", "slack", "aws", "google", "azure"]
    }
  }
}
