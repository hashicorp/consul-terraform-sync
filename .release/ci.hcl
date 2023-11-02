# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

schema = "1"

project "consul-terraform-sync" {
  team = "consul api tooling"
  slack {
    # feed-consul-api-gh
    notification_channel = "C026W707YHJ"
  }
  github {
    organization = "hashicorp"
    repository = "consul-terraform-sync"
    release_branches = [
      "main",
      "release/0.2.x",
      "release/0.3.x",
      "release/0.4.x",
      "release/0.5.x",
      "release/0.6.x",
      "release/0.7.x"
    ]
  }
}

event "build" {
  depends = ["merge"]
  action "build" {
    organization = "hashicorp"
    repository = "crt-workflows-common"
    workflow = "build"
  }
}

event "prepare" {
  depends = ["build"]
  action "prepare" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "prepare"
    depends      = ["build"]
  }

  notification {
    on = "fail"
  }
}

## These are promotion and post-publish events
## they should be added to the end of the file after the verify event stanza.

event "trigger-staging" {
// This event is dispatched by the bob trigger-promotion command
// and is required - do not delete.
}

event "promote-staging" {
  depends = ["trigger-staging"]
  action "promote-staging" {
    organization = "hashicorp"
    repository = "crt-workflows-common"
    workflow = "promote-staging"
    config = "release-metadata.hcl"
  }

  notification {
    on = "always"
  }
}

event "promote-staging-docker" {
  depends = ["promote-staging"]
  action "promote-staging-docker" {
    organization = "hashicorp"
    repository = "crt-workflows-common"
    workflow = "promote-staging-docker"
  }

  notification {
    on = "always"
  }
}

event "trigger-production" {
// This event is dispatched by the bob trigger-promotion command
// and is required - do not delete.
}

event "promote-production" {
  depends = ["trigger-production"]
  action "promote-production" {
    organization = "hashicorp"
    repository = "crt-workflows-common"
    workflow = "promote-production"
  }

  notification {
    on = "always"
  }
}

event "promote-production-docker" {
  depends = ["promote-production"]
  action "promote-production-docker" {
    organization = "hashicorp"
    repository = "crt-workflows-common"
    workflow = "promote-production-docker"
  }

  notification {
    on = "always"
  }
}

event "promote-production-packaging" {
  depends = ["promote-production-docker"]
  action "promote-production-packaging" {
    organization = "hashicorp"
    repository = "crt-workflows-common"
    workflow = "promote-production-packaging"
  }

  notification {
    on = "always"
  }
}
