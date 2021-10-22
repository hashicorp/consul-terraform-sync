#!/usr/bin/env bash

version_file=$1
version=$(awk '$1 == "Version" && $2 == "=" { gsub(/"/, "", $3); print $3 }' < "${version_file}")
prerelease=$(awk '$1 == "VersionPrerelease" && $2 == "=" { gsub(/"/, "", $3); print $3 }' < "${version_file}")
metadata=$(awk '$1 == "VersionMetadata" && $2 == "=" { gsub(/"/, "", $3); print $3 }' < "${version_file}")

if [ -n "$prerelease" ]; then
    version="${version}-${prerelease}"
else
    version="${version}"
fi

if [ -n "$metadata" ]; then
    version="${version}+${metadata}"
else
    version="${version}"
fi

echo "${version}"