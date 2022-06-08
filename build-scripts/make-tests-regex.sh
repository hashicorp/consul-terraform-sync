#!/usr/bin/env bash

set -euo pipefail

# Description: Creates a regex for running Go tests
# Input: Standard input containing test names
# Output: Standard output

echo -n "^($(paste -sd '|' -))$"
