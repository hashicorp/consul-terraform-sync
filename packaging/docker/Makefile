SHELL := /usr/bin/env bash -euo pipefail -c
export PROJECT_NAME=consul-terraform-sync

.PHONY: build download docker

build: download docker

download: export RELEASE_DIST=https://releases.hashicorp.com
download: export VERSION=$$(./../../build-scripts/version.sh ../../version/version.go)
download: 
	curl -o $(PROJECT_NAME).zip $(RELEASE_DIST)/$(PROJECT_NAME)/$(VERSION)/$(PROJECT_NAME)_$(VERSION)_linux_$(ARCH).zip
	unzip $(PROJECT_NAME).zip
	rm $(PROJECT_NAME).zip

docker: export VERSION=$$(./../../build-scripts/version.sh ../../version/version.go)
docker:
	echo $(VERSION)
	docker buildx build --load --build-arg VERSION=$(VERSION) --build-arg ARCH=$(ARCH) --platform=linux/$(ARCH)$(GOARM_DOCKER) --no-cache -t $(PROJECT_NAME):$(VERSION) .
	docker save --output $(PROJECT_NAME)_$(VERSION)_docker_linux_$(ARCH).tar $(PROJECT_NAME):$(VERSION)
