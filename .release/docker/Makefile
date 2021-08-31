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

# GITHUB_SHA is an environment variable from GitHub Actions
# save images tagged as $GITHUB_SHA-$VERSION-$ARCH since we will need to combine distinct tags
# together with docker manifest
docker: export VERSION=$$(./../../build-scripts/version.sh ../../version/version.go)
docker:
	echo "Building image for $(PROJECT_NAME):$(GITHUB_SHA)-$(VERSION)-$(ARCH)"
	docker buildx create --use
	docker buildx build --load --build-arg VERSION=$(VERSION) --build-arg NAME=$(PROJECT_NAME) --platform=linux/$(ARCH)$(GOARM_DOCKER) -t $(PROJECT_NAME):$(GITHUB_SHA)-$(VERSION)-$(ARCH) .
	docker save --output $(PROJECT_NAME)_$(VERSION)_docker_linux_$(ARCH).tar $(PROJECT_NAME):$(GITHUB_SHA)-$(VERSION)-$(ARCH)