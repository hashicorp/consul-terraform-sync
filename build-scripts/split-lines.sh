#!/usr/bin/env bash
# Copyright IBM Corp. 2020, 2025
# SPDX-License-Identifier: MPL-2.0


set -euo pipefail

# Description: Splits lines into a number of files and stores the output in a given directory
# Input: Standard input containing lines, and command line arguments <parts-count> <parts-output-dir>
# Output: Files containing a subset of the input lines, stored in the output directory

# ----------------- Functions ------------------
function split_list() {
    local parts_count=$1
    local collected_input=$2
    local output_dir=$3

    local total_lines
    local lines_per_file

    total_lines=$(wc -l < "${collected_input}" | xargs)
    ((lines_per_file = (total_lines + parts_count - 1) / parts_count))

    mkdir -p "${output_dir}"
    echo ">> Tests total count: ${total_lines}"
    split -d -a 1 -l "${lines_per_file}" "${collected_input}" "${output_dir}/part."
    echo ">> Parts files line counts:"
    wc -l "${output_dir}"/*
}

function read_stdin() {
  local collected=$1

  while read -r line; do
    echo "${line}" >> "${collected}"
  done < /dev/stdin
}

function validate() {
  local parts_count=$1

  if [ "${parts_count}" -gt 10 ]; then
    >&2 echo ">> ERROR: Parts count (${parts_count}) exceeded max value (10)"
    exit 1
  fi
}

# ----------------- Main Logic ------------------
if [ "$#" -ne 2 ]; then
    >&2 echo ">> ERROR: Incorrect number of parameters. Usage: $0 <parts-count> <parts-output-dir>"
    exit 1
fi

parts_count=$1
output_dir=$2

validate "${parts_count}"
collected_input=$(mktemp /tmp/split.XXXXX)
read_stdin "${collected_input}"
split_list "${parts_count}" "${collected_input}" "${output_dir}"

# ----------------- Clean up ------------------
rm "${collected_input}"
unset parts_count output_dir collected_input
