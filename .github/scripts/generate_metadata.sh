#!/usr/bin/env bash

set -euo pipefail

declare -r org="hashicorp"
declare -r product="consul-terraform-sync"

#if ! product_version="$(make version)"; then
#  echo "Error running 'make version'"
#  echo "${product_version}"
#  exit 1
#fi

if ! git_sha="$(git rev-parse HEAD)"; then
  echo "Unable to determine git sha for this commit"
  exit 1
fi

cat <<-EOM
{
  "org": "${org}",
  "product": "${product}",
  "sha": "${git_sha}",
  "version": "v0.2.1",
  "buildworkflowid" : "${GITHUB_RUN_ID}"
}
EOM