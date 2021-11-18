#!/usr/bin/env bash

# Check go fmt
echo "==> Checking that code complies with go fmt requirements..."
gofmt_files=$(go fmt ./...)
if [[ -n ${gofmt_files} ]]; then
    echo 'go fmt needs to be run on the following files:'
    echo "${gofmt_files}"
    echo "You can use the command: \`go fmt\` to reformat code."
    exit 1
fi

exit 0