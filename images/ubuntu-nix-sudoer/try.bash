#!/usr/bin/env bash

set -euxo pipefail

image=$1
user=$2

podman build --tag "$image" --build-arg username="$user" --file ./images/"$image"/Containerfile .
podman run --interactive --tty --rm --name "$image" "$image"
