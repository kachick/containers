FROM ubuntu:22.04

# hadolint global ignore=DL4006
# Because of SHELL is not supported in OCI format
# SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# If you faced any error from dprint: https://github.com/dprint/dprint-plugin-dockerfile/issues

# Available versions in apt: https://packages.ubuntu.com/jammy/curl
# --no-install-recommends is recommended by hadolint, but it omits ca-certificates
RUN apt-get update && apt-get install --no-install-recommends -y curl=7.81.0-1ubuntu1.15 ca-certificates=20230311ubuntu0.22.04.1 \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/*

RUN curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install linux \
  --extra-conf "sandbox = false" \
  --init none \
  --no-confirm

ENV PATH="${PATH}:/nix/var/nix/profiles/default/bin"
