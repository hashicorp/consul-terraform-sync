FROM alpine:3.13 as default

# NAME and VERSION are the name of the software in releases.hashicorp.com
# and the version to download. Example: NAME=consul VERSION=1.2.3.
ARG NAME=consul-terraform-sync
ARG VERSION

LABEL maintainer="Consul Team <consul@hashicorp.com>"
LABEL version=$VERSION

# Set ARGs as ENV so that they can be used in ENTRYPOINT/CMD
ENV NAME=$NAME
ENV VERSION=$VERSION

RUN apk add --no-cache dumb-init git

# Create a non-root user to run the software.
RUN addgroup ${NAME} && adduser -S -G ${NAME} ${NAME}

COPY consul-terraform-sync /bin/consul-terraform-sync

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
