#!/usr/bin/dumb-init /bin/sh
set -e

# Note above that we run dumb-init as PID 1 in order to reap zombie processes
# as well as forward signals to all processes in its session. Normally, sh
# wouldn't do either of these functions so we'd leak zombies as well as do
# unclean termination of all our sub-processes.

# If the user is trying to run consul-terraform-sync directly with some arguments,
# then pass them to consul-terraform-sync.
# On alpine /bin/sh is busybox which supports the bashism below.
if [ "${1:0:1}" = '-' ]; then
	set -- /bin/consul-terraform-sync "$@"
fi

# If user is trying to run consul-terraform-sync with no arguments (daemon-mode),
# docker will run '/bin/sh -c /bin/${NAME}'. Check for the full command since
# running 'bin/sh' is a common pattern
if [ "$*" = '/bin/sh -c /bin/${NAME}' ]; then
	set -- /bin/consul-terraform-sync
fi

# Matches VOLUME in the Dockerfile, for importing config files into image
CTS_CONFIG_DIR=/consul-terraform-sync/config

# Set the configuration directory
if [ "$1" = '/bin/consul-terraform-sync' ]; then
	shift
	set -- /bin/consul-terraform-sync -config-dir="$CTS_CONFIG_DIR" "$@"
fi

exec "$@"
