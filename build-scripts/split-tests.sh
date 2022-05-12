#!/usr/bin/env bash

function list_tests() {
    local build_tags=$1
    local pkg=$2

    go test -build_tags="${build_tags}" -list . "${pkg}" | grep "^Test"
}

function split_list() {
    local chunks_count=$1
    local input_file=$2
    local output_dir=$3

    local total_lines
    local lines_per_file

    total_lines=$(wc -l < "${input_file}")
    ((lines_per_file = (total_lines + chunks_count - 1) / chunks_count))

    split -d -a 1 -l "${lines_per_file}" "${input_file}" "${output_dir}/chunk."
}

function join_lists() {
  local chunks_dir=$1
  local joined

  for f in "${chunks_dir}"/*; do
    joined=$(paste -sd '|' "${f}")
    printf "^(%s)$" "${joined}" > "${f}"
  done
}

# -----------------------------------

build_tags=$1
package=$2
chunks_count=$3
chunks_dir=$4

all_tests_file=$(mktemp /tmp/split.XXXXX)

if [ "${chunks_count}" -gt 10 ]; then
  echo "ERROR: Max chunks_count (10) exceeded"
  exit 1
fi

list_tests "${build_tags}" "${package}" > "${all_tests_file}"
split_list "${chunks_count}" "${all_tests_file}" "${chunks_dir}"
join_lists "${chunks_dir}"

rm "${all_tests_file}"
