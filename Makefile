# Metadata about this makefile and position
MKFILE_PATH := $(lastword $(MAKEFILE_LIST))
CURRENT_DIR := $(patsubst %/,%,$(dir $(realpath $(MKFILE_PATH))))

# System information
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
GOPATH=$(shell go env GOPATH)
GOPATH := $(lastword $(subst :, ,${GOPATH}))# use last GOPATH entry

# Project information
PROJECT := $(shell go list -m)
NAME := $(notdir $(PROJECT))
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
GIT_DESCRIBE ?=
GIT_DIRTY ?= $(shell git diff --stat)
VERSION := $(shell awk -F\" '/Version =/ { print $2; exit }' "${CURRENT_DIR}/version/version.go")

# Tags specific for building
GOTAGS ?=

LD_FLAGS ?= \
	-s \
	-w \
	-X '${PROJECT}/version.Name=${NAME}' \
	-X '${PROJECT}/version.GitCommit=${GIT_COMMIT}' \
	-X '${PROJECT}/version.GitDescribe=${GIT_DESCRIBE}' \
	-X '${PROJECT}/version.GitDirty=${GIT_DIRTY}'

# dev builds and installs the project locally to $GOPATH/bin.
dev:
	@echo "==> Installing ${NAME} for ${GOOS}/${GOARCH}"
	@rm -f "${GOPATH}/pkg/${GOOS}_${GOARCH}/${PROJECT}/version.a"
	@go install -ldflags "$(LD_FLAGS)" -tags '$(GOTAGS)'
.PHONY: dev

# test runs the unit tests
test:
	@echo "==> Testing ${NAME}"
	@go test -count=1 -timeout=30s -cover ./... ${TESTARGS}
.PHONY: test

# test-unit-and-integration runs the unit and integration tests
test-unit-and-integration:
	@echo "==> Testing ${NAME} (unit & integration tests)"
	@mkdir -p .build/test-results
	@gotestsum --format testname --jsonfile .build/test-results.json -- -count=1 -timeout=2m -tags=integration -cover ./... ${TESTARGS}
.PHONY: test-unit-and-integration

# test-setup-e2e sets up the CTS binary and permissions to run in E2E tests
test-setup-e2e: dev
	sudo mv ${GOPATH}/bin/consul-terraform-sync /usr/local/bin/consul-terraform-sync
.PHONY: test-setup-e2e

# test-e2e-ci does e2e test setup and then runs the e2e tests in a CI environment
test-e2e-ci: test-setup-e2e
	@echo "==> Testing ${NAME} (e2e)"
	@echo "Tests regex: $(shell cat "${TESTS_REGEX_PATH}")"
	@gotestsum --format testname --jsonfile .build/test-results.json -- \
		./e2e -race -count=1 -timeout=600s -tags=e2e -run="$(shell cat "${TESTS_REGEX_PATH}")" ${TESTARGS}
.PHONY: test-e2e-ci

# test-e2e-local does e2e test setup and then runs the e2e tests
test-e2e-local: test-setup-e2e
	@echo "==> Testing ${NAME} (e2e)"
	@go test ./e2e -race -count=1 -v -timeout=45m -tags=e2e -local ./... ${TESTARGS}
.PHONY: test-e2e-local

# test-compat sets up the CTS binary and then runs the compatibility tests
test-compat: test-setup-e2e
	@echo "==> Testing ${NAME} compatibility with Consul"
	@go test -count=1 -timeout 30m -tags 'e2e' -v ./e2e/compatibility -run TestCompatibility_Consul
	@echo "==> Testing ${NAME} Terraform Downloads"
	@go test ./e2e/compatibility -timeout 5m -tags=e2e -v -run TestCompatibility_TFDownload
.PHONY: test-compat

# test-vault-integration sets up the CTS binary and then runs Vault integration tests
test-vault-integration: test-setup-e2e
	@echo "==> Testing ${NAME} compatibility with Consul"
	@go test -count=1 -timeout=80s -tags 'integration vault' -v ./... -run Vault
.PHONY: test-vault-integration

# test-benchmarks requires Terraform in the path of execution and Consul in $PATH.
test-benchmarks:
	@echo "==> Running benchmarks for ${NAME}"
	@go test -json ./e2e/benchmarks -timeout 2h -bench=. -tags e2e
.PHONY: test-benchmarks

# compile-weekly-tests is a check that our weekly-run tests can compile. this
# will be called on a more frequent cadence than weekly
compile-weekly-tests:
	@echo "==> Running compile check for weekly tests for ${NAME}"
	@go test -run TestCompatibility_Compile ./e2e/compatibility -timeout 5m -tags '$(GOTAGS) e2e'
	@go test -run TestBenchmarks_Compile ./e2e/benchmarks -timeout 5m -tags '$(GOTAGS) e2e'
	@go test -run TestVaultIntegration_Compile ./... -timeout=5m -tags '$(GOTAGS) integration vault'
.PHONY: compile-weekly-tests

# delete any cruft
clean:
	rm -f ./e2e/terraform
.PHONY: clean

# generate generates code for mockery annotations and open-api
# requires installing https://github.com/vektra/mockery &
# https://github.com/deepmap/oapi-codegen
generate:
	go generate ./...
.PHONY: generate

go-fmt-check:
	@$(CURDIR)/build-scripts/gofmtcheck.sh
.PHONY: go-fmt-check

terraform-fmt-check:
	@$(CURDIR)/build-scripts/terraformfmtcheck.sh
.PHONY: terraform-fmt-check

version:
	@$(CURDIR)/build-scripts/version.sh version/version.go
.PHONY: version
