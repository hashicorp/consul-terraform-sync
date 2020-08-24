# Metadata about this makefile and position
MKFILE_PATH := $(lastword $(MAKEFILE_LIST))
CURRENT_DIR := $(patsubst %/,%,$(dir $(realpath $(MKFILE_PATH))))

# System information
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
GOPATH=$(shell go env GOPATH)
GOPATH := $(lastword $(subst :, ,${GOPATH}))# use last GOPATH entry

# Project information
GOVERSION := 1.14
PROJECT := $(shell go list -m)
NAME := $(notdir $(PROJECT))
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
GIT_DESCRIBE ?= $(shell git describe --tags --always)
VERSION := $(shell awk -F\" '/Version/ { print $$2; exit }' "${CURRENT_DIR}/version/version.go")

# Tags specific for building
GOTAGS ?=

LD_FLAGS ?= \
	-s \
	-w \
	-X '${PROJECT}/version.Name=${NAME}' \
	-X '${PROJECT}/version.GitCommit=${GIT_COMMIT}' \
	-X '${PROJECT}/version.GitDescribe=${GIT_DESCRIBE}'

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
	@go test -count=1 -timeout=60s -tags=integration -cover ./... ${TESTARGS}
.PHONY: test-all

# test-e2e runs e2e tests
test-e2e:
	@echo "==> Testing ${NAME} (e2e)"
	@go test ./e2e -count=1 -timeout=60s -tags=e2e ./... ${TESTARGS}
.PHONY: test-e2e

# test-setup-e2e sets up the nia binary and permissions to run consul-nia
# cli in circle
test-setup-e2e: dev
	sudo mv ${GOPATH}/bin/consul-nia /usr/local/bin/consul-nia
.PHONY: test-setup-e2e

# test-e2e-cirecleci does circleci setup and then runs the e2e tests
test-e2e-cirecleci: test-setup-e2e test-e2e
.PHONY: test-e2e-cirecleci
