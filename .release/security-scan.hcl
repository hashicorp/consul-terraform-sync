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
        "GHSA-vvm8-gf78-28g4", // go-jose/v3: no fixed v3 release (fixed only in v4); pulled in transitively via vault/api.
        "GHSA-4w3g-jfj2-cw2q", // go-jose/v3: no fixed v3 release (fixed only in v4); pulled in transitively via vault/api.
        "GHSA-mgcc-cww5-fp2q", // go-jose/v3: no fixed v3 release (fixed only in v4); pulled in transitively via vault/api.
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
        "GHSA-vvm8-gf78-28g4", // go-jose/v3: no fixed v3 release (fixed only in v4); pulled in transitively via vault/api.
        "GHSA-4w3g-jfj2-cw2q", // go-jose/v3: no fixed v3 release (fixed only in v4); pulled in transitively via vault/api.
        "GHSA-mgcc-cww5-fp2q", // go-jose/v3: no fixed v3 release (fixed only in v4); pulled in transitively via vault/api.
      ]
    }
  }
  secrets {
    matchers {
      known = ["tfc", "hcp", "tfe", "github", "artifactory", "slack", "aws", "google", "azure"]
    }
  }
}
