# Metadata about this makefile and position
MKFILE_PATH := $(lastword $(MAKEFILE_LIST))
CURRENT_DIR := $(patsubst %/,%,$(dir $(realpath $(MKFILE_PATH))))

# System information
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
GOPATH=$(shell go env GOPATH)
GOPATH := $(lastword $(subst :, ,${GOPATH}))# use last GOPATH entry

# Project information
GOVERSION := 1.16
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

# test runs the test suite
test:
	@echo "==> Testing ${NAME}"
	@go test -count=1 -timeout=30s -cover ./... ${TESTARGS}
.PHONY: test

# test-integration runs the test suite and integration tests
test-integration:
	@echo "==> Testing ${NAME} (test suite & integration)"
	@go test -count=1 -timeout=80s -tags=integration -cover ./... ${TESTARGS}
.PHONY: test-all

# test-setup-e2e sets up the sync binary and permissions to run in circle
test-setup-e2e: dev
	sudo mv ${GOPATH}/bin/consul-terraform-sync /usr/local/bin/consul-terraform-sync
.PHONY: test-setup-e2e

# test-e2e-cirecleci does e2e test setup and then runs the e2e tests
test-e2e-cirecleci: test-setup-e2e
	@echo "==> Testing ${NAME} (e2e)"
	@go test ./e2e -count=1 -timeout=900s -tags=e2e ./... ${TESTARGS}
.PHONY: test-e2e-cirecleci

# test-e2e-local does e2e test setup and then runs the e2e tests
test-e2e-local: test-setup-e2e
	@echo "==> Testing ${NAME} (e2e)"
	@go test ./e2e -count=1 -v -timeout=45m -tags=e2e -local ./... ${TESTARGS}
.PHONY: test-e2e-local

# test-compat sets up the CTS binary and then runs the compatibility tests
test-compat: test-setup-e2e
	@echo "==> Testing ${NAME} compatibility with Consul"
	@go test ./e2e/compatibility -timeout 30m -tags=e2e -v -run TestCompatibility_Consul
.PHONY: test-compat

# test-benchmarks requires Terraform in the path of execution and Consul in $PATH.
test-benchmarks:
	@echo "==> Running benchmarks for ${NAME}"
	@go test -json ./e2e/benchmarks -timeout 2h -bench=. -tags e2e
.PHONY: test-benchmarks

# compile-weekly-tests is a check that our weekly-runned tests can compile. this
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
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

terraform-fmt-check:
	@sh -c "'$(CURDIR)/scripts/terraformfmtcheck.sh'"

# temp noop command to get build pipeline working
dev-tree:
	@true
.PHONY: dev-tree

version:
ifneq (,$(wildcard version/version_ent.go))
	@$(CURDIR)/build-scripts/version.sh version/version_ent.go
else
	@$(CURDIR)/build-scripts/version.sh version/version.go
endif

.PHONY: version
