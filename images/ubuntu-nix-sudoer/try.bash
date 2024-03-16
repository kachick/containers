#!/usr/bin/env bash

set -euxo pipefail

image_tag=ubuntu-nix-sudoer
user="$1"

podman build --tag "$image_tag" --build-arg username="$user" --file "./images/${image_tag}/Containerfile" .
podman run --interactive --tty --rm "$image_tag"
