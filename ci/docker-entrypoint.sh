#!/usr/bin/env bash

# make sure the host GID gets remaped to this container's docker GID
DOCKER_GID=$(ls -ln /var/run/docker.sock | awk '{print $4}')
groupmod -g "${DOCKER_GID}" docker

exec /usr/sbin/sshd -D -e
