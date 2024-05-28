# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

FROM alpine:3 as default

# NAME and PRODUCT_VERSION are the name of the software in releases.hashicorp.com
# and the version to download. Example: NAME=consul PRODUCT_VERSION=1.2.3.
ARG NAME=consul-terraform-sync
ARG PRODUCT_VERSION

LABEL maintainer="Consul Team <consul@hashicorp.com>"
LABEL version=$PRODUCT_VERSION
LABEL licenses="MPL-2.0"

# Set ARGs as ENV so that they can be used in ENTRYPOINT/CMD
ENV NAME=$NAME
ENV VERSION=$PRODUCT_VERSION
ENV PRODUCT_NAME=$NAME

# TARGETARCH and TARGETOS are set automatically when --platform is provided.
ARG TARGETOS TARGETARCH

RUN apk add --no-cache dumb-init git bash openssh

# Create a non-root user to run the software.
RUN addgroup ${NAME} && adduser -S -G ${NAME} ${NAME}

COPY dist/$TARGETOS/$TARGETARCH/consul-terraform-sync /bin/consul-terraform-sync
COPY LICENSE /usr/share/doc/$PRODUCT_NAME/LICENSE.txt

### Added for CTS
RUN mkdir -p /consul-terraform-sync/config \
	&& chown -R ${NAME}:${NAME} /consul-terraform-sync
VOLUME /consul-terraform-sync/config
COPY .release/docker/docker-entrypoint.sh /bin/docker-entrypoint.sh
WORKDIR /consul-terraform-sync
ENTRYPOINT ["/bin/docker-entrypoint.sh"]
###

USER ${NAME}
CMD /bin/${NAME}
