#!/usr/bin/env bash

set -euxo pipefail

image_tag=ubuntu-nix-systemd
user="$1"

podman build --tag "$image_tag" --build-arg username="$user" --file "./images/${image_tag}/Containerfile" .
container_id="$(podman run --rm --detach "$image_tag")"
sleep 1 # Wait systemd to be ready
podman exec --user "$user" --interactive --tty "$container_id" bash
podman stop "$container_id"
