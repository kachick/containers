# Containerfile - Nix package manager on Ubuntu

[![CI - Nix Status](https://github.com/kachick/containers-ubuntu-nix/actions/workflows/ci-nix.yml/badge.svg?branch=main)](https://github.com/kachick/containers-ubuntu-nix/actions/workflows/ci-nix.yml?query=branch%3Amain+)

## Usage

```bash
git clone git@github.com:kachick/container-ubuntu-nix.git
cd container-ubuntu-nix
podman build --tag container-ubuntu-nix --file Containerfile .
podman run -it container-ubuntu-nix
```

```console
root@a1cdfec2dca5:/# nix --version
nix (Nix) 2.20.3
root@a1cdfec2dca5:/# nix run nixpkgs#hello
Hello, world!
```

## Motivation

- [nixos/nix](https://hub.docker.com/r/nixos/nix) is not NixOS, but it does not have some basic tools as `groupadd`\
  ref: <https://stackoverflow.com/questions/75653182/why-do-some-official-nix-docker-containers-not-have-the-nixos-rebuild-command>
- I want flake by default
- I prefer ubuntu rather than [alpine](https://hub.docker.com/r/nixos/nix) for daily use

## Note

- Devcontainer
  - https://github.com/kachick/wait-other-jobs/pull/517
  - https://github.com/kachick/devcontainer-ubuntu-nix
- [Installer](https://github.com/DeterminateSystems/nix-installer)
