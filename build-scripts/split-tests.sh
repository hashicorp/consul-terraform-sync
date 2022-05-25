#!/usr/bin/env bash

set -euo pipefail

# ----------------- Functions ------------------

function list_tests() {
    local build_tags=$1
    local pkg=$2

    go test -tags="${build_tags}" -list . "${pkg}" | grep "^Test" | sort
}

function split_list() {
    local chunks_count=$1
    local input_file=$2
    local output_dir=$3

    local total_lines
    local lines_per_file

    total_lines=$(wc -l < "${input_file}" | xargs)
    ((lines_per_file = (total_lines + chunks_count - 1) / chunks_count))

    echo ">> Tests total count: ${total_lines}"
    mkdir -p "${output_dir}"
    split -d -a 1 -l "${lines_per_file}" "${input_file}" "${output_dir}/chunk."
    echo ">> Chunks files line counts:"
    wc -l "${output_dir}"/*
}

function join_lists() {
  local chunks_dir=$1

  local joined

  for f in "${chunks_dir}"/*; do
    joined=$(paste -sd '|' "${f}")
    printf "^(%s)$" "${joined}" > "${f}"
  done
}

function validate() {
  local chunks_count=$1

  if [ "${chunks_count}" -gt 10 ]; then
    >&2 echo ">> ERROR: Chunks count (${chunks_count}) exceeded max value (10)"
    exit 1
  fi
}

# ----------------- Main Logic ------------------

if [ "$#" -ne 4 ]; then
    >&2 echo ">> ERROR: Incorrect number of parameters. Usage: $0 <build-tags> <package> <chunks-count> <chunks-output-dir>"
    exit 1
fi

build_tags=$1
package=$2
chunks_count=$3
chunks_dir=$4

all_tests_file=$(mktemp /tmp/split.XXXXX)

validate "${chunks_count}"
list_tests "${build_tags}" "${package}" > "${all_tests_file}"
split_list "${chunks_count}" "${all_tests_file}" "${chunks_dir}"
join_lists "${chunks_dir}"

# ----------------- Clean up ------------------

rm "${all_tests_file}"
unset build_tags package chunks_count chunks_dir
