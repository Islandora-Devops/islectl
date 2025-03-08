#!/usr/bin/env bash

set -eou pipefail

go build

GITHUB_ACTIONS="${GITHUB_ACTIONS:=false}"
if [ "$GITHUB_ACTIONS" = "false" ]; then
  docker build -t ghcr.io/islandora-devops/islectl:ci ./ci
fi

if [ ! -f ./ci/host-keys/ssh_host_ed25519_key ]; then
  ssh-keygen -t ed25519 -f ./ci/host-keys/ssh_host_ed25519_key -N ""
fi

if [ ! -f ./ci/ssh/id_rsa ]; then
  chmod 700 ./ci/ssh
  ssh-keygen -t rsa -b 4096 -f ./ci/ssh/id_rsa -q -N ""
fi

docker compose --file ./ci/docker-compose.yml up -d

sleep 5

ssh-keyscan -p 1234 localhost > ./ci/ssh/known_hosts

./islectl create \
  --context-name foo \
  --type remote \
  --profile dev \
  --project-dir /home/foo/isle \
  --project-name isle \
  --ssh-hostname localhost \
  --ssh-port 1234 \
  --ssh-user foo \
  --ssh-key $(pwd)/ci/ssh/id_rsa \
  --yes

./islectl build --context foo
./islectl down --context foo

docker compose --file ./ci/docker-compose.yml down
