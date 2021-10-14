#!/usr/bin/env bash

# Check terraform fmt
echo "==> Checking that code complies with terraform fmt requirements..."
tffmt_files=$(terraform fmt -check -recursive)
if [[ -n ${tffmt_files} ]]; then
    echo 'terraform fmt needs to be run on the following files:'
    echo "${tffmt_files}"
    echo "You can use the command: \`terraform fmt\` to reformat code."
    exit 1
fi

exit 0