#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


set -euo pipefail

# Description: Creates a regex for running Go tests
# Input: Standard input containing test names
# Output: Standard output

echo -n "^($(paste -sd '|' -))$"
