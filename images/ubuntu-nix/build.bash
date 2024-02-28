#!/usr/bin/env bash

set -euxo pipefail

podman build --tag ubuntu-nix \
	--build-arg container_user_uid="$(stat -c '%u' .)" \
	--build-arg container_user_gid="$(stat -c '%g' .)" \
	--build-arg username='user' \
	--build-arg groupname='ubuntu-nix' \
	--file ./images/ubuntu-nix/Containerfile \
	.
