#!/usr/bin/env bash

set -euo pipefail

# Description: Lists Go tests
# Input: Command line arguments <build-tags> <package-path>
# Output: Standard output

# ----------------- Main Logic ------------------
if [ "$#" -ne 2 ]; then
    >&2 echo ">> ERROR: Incorrect number of parameters. Usage: $0 <build-tags> <package-path>"
    exit 1
fi

build_tags=$1
package_path=$2

go test -tags="${build_tags}" -list . "${package_path}" | grep "^Test"

# ----------------- Clean up ------------------
unset build_tags package_path
