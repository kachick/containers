# Containerfile(s)

[![CI - Nix Status](https://github.com/kachick/containers/actions/workflows/ci-nix.yml/badge.svg?branch=main)](https://github.com/kachick/containers/actions/workflows/ci-nix.yml?query=branch%3Amain+)
[![CI - Container Status](https://github.com/kachick/containers/actions/workflows/containers.yml/badge.svg?branch=main)](https://github.com/kachick/containers/actions/workflows/containers.yml?query=branch%3Amain+)

Usage is written in README.md in each images directory.

## Container Images

| Name                                            | Build in loccal              | Examples                                                                                                    |
| ----------------------------------------------- | ---------------------------- | ----------------------------------------------------------------------------------------------------------- |
| [ubuntu-nix-systemd](images/ubuntu-nix-systemd) | `task try:systemd USER=user` | [dotfiles](https://github.com/kachick/dotfiles/blob/f81983534aeb9f0db80932f5ae73bd59793b8af4/README.md#L47) |
| [ubuntu-nix-sudoer](images/ubuntu-nix-sudoer)   | `task try:sudoer USER=user`  | `podman run --user=user --rm -it ghcr.io/kachick/ubuntu-nix-sudoer:latest`                                  |
