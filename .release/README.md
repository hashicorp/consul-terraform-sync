# How to Release Using Common Release Tooling (CRT)

## Background

- This repo has been configured to use CRT.

- Releasable artifacts will be built on every push to `main` and `release/**` branches (as configured in [build.yaml](../.github/workflows/build.yml)).

- Please ensure that the version in [version/version.go](../version/version.go) or [version/version_ent.go](../version/version_ent.go) is set to the next targeted release version for that branch.

- The `release_branches` key in the [ci.hcl](ci.hcl) file should be set to all branches that release artifacts should be built for (commonly this should be any active release branches and for some products main/master). For any build workflow that successfully completes for commits pushed to these branches, artifacts will be processed and made ready for release.

- The crt-orchestrator GitHub app is responsible for processing the artifacts and making them 'release' ready. This includes running jobs like signing and notarization and will include running quality tests and security scans in the future. These GitHub workflows are located in the hashicorp/crt-workflows-common repo and are run in the context of this repo by the crt-orchestrator app. The workflows are mapped to identically named events in the `ci.hcl` file.

## How to Promote Artifacts

Artifacts can be promoted to `staging` and then on to `production` when it's time to do a release.

Promotion approvals are gaited by GitHub environments. These environments should have been [set up](https://docs.google.com/document/d/14v-KQKhwSsfduJQPaD13GZtHwxIUx0nRDFspKtZAslQ/edit#bookmark=id.ow4jgmydlwsa) when the repo was being onboarded to CRT and will contain a small subset of approvers per product.

For this repo the environment is - `cts-oss`.

### When you are ready to promote artifacts to staging:

- Navigate to the [hashicorp-crt-stable-local](https://artifactory.hashicorp.engineering/ui/repos/tree/General/hashicorp-crt-stable-local) repo in Artifactory and check that the artifacts for the commit (sha) you want to release are there. You will find the product's stable release artifacts stored there under the path: `<product-name>/<version>/<git-sha>`.

- Follow [these](https://docs.google.com/document/d/14v-KQKhwSsfduJQPaD13GZtHwxIUx0nRDFspKtZAslQ/edit#bookmark=id.jxvv98nkufyi) steps to promote the artifacts to `staging`.

### When you are ready to promote artifacts to production:

- Navigate to the [hashicorp-crt-staging-local](https://artifactory.hashicorp.engineering/ui/repos/tree/General/hashicorp-crt-staging-local) repo in Artifactory and check that the artifacts for the commit (sha) you want to release are there. You will find the product's staged release artifacts stored there under the path: `<product-name>/<version>/<git-sha>`.

- Follow [these](https://docs.google.com/document/d/14v-KQKhwSsfduJQPaD13GZtHwxIUx0nRDFspKtZAslQ/edit#bookmark=id.dia5v7srf30s) steps to promote the artifacts to `production`. As mentioned in the [doc](https://docs.google.com/document/d/14v-KQKhwSsfduJQPaD13GZtHwxIUx0nRDFspKtZAslQ/edit#bookmark=id.4hmskuc5vs8s), this will also carry out other release related steps like pushing a tag back to the product repo and publishing linux packages.

## Things to note

Currently, docker image publishing is not available in CRT. Please reach out to #team-rel-eng when you are doing a release and we will assist you with doing this manually. This will be available in CRT very soon!

Keep in mind that this is a *high-level* overview of how to release artifacts and binaries using Common Release Tooling (CRT). We have a self-serve onboarding guide that covers much more in depth info. [Self-Serve Onboarding Guide](https://docs.google.com/document/d/14v-KQKhwSsfduJQPaD13GZtHwxIUx0nRDFspKtZAslQ/edit?usp=sharing) 

As always, if you have any questions beyond the guide, feel free to reach out to #team-rel-eng in Slack and we will be happy to assist! :heart:
