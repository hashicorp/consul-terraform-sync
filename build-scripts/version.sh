#!/usr/bin/env bash
# Copyright IBM Corp. 2020, 2025
# SPDX-License-Identifier: MPL-2.0


version_file=$1
version=$(awk '$1 == "Version" && $2 == "=" { gsub(/"/, "", $3); print $3 }' < "${version_file}")
prerelease=$(awk '$1 == "VersionPrerelease" && $2 == "=" { gsub(/"/, "", $3); print $3 }' < "${version_file}")
metadata=$(awk '$1 == "VersionMetadata" && $2 == "=" { gsub(/"/, "", $3); print $3 }' < "${version_file}")

if [[ -n "$base" ]]; then
    echo "${version}"
    exit
fi

if [ -n "$prerelease" ]; then
    version="${version}-${prerelease}"
fi

if [ -n "$metadata" ]; then
    version="${version}+${metadata}"
fi

echo "${version}"
