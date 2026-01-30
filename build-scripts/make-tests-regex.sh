#!/usr/bin/env bash
# Copyright IBM Corp. 2020, 2025
# SPDX-License-Identifier: MPL-2.0


set -euo pipefail

# Description: Creates a regex for running Go tests
# Input: Standard input containing test names
# Output: Standard output

echo -n "^($(paste -sd '|' -))$"
