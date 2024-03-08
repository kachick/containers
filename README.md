# Containerfile(s)

[![CI - Nix Status](https://github.com/kachick/containers/actions/workflows/ci-nix.yml/badge.svg?branch=main)](https://github.com/kachick/containers/actions/workflows/ci-nix.yml?query=branch%3Amain+)
[![CI - Container Status](https://github.com/kachick/containers/actions/workflows/containers.yml/badge.svg?branch=main)](https://github.com/kachick/containers/actions/workflows/containers.yml?query=branch%3Amain+)

Usage is written in README.md in each images directory.

## Container Images

| Name                                            | Build in loccal                               | Pull from container registry                           |
| ----------------------------------------------- | --------------------------------------------- | ------------------------------------------------------ |
| [ubuntu-nix-sudoer](images/ubuntu-nix-sudoer)   | `task try IMAGE=ubuntu-nix-sudoer USER=user`  | `podman pull ghcr.io/kachick/ubuntu-nix-sudoer:latest` |
| [ubuntu-nix-systemd](images/ubuntu-nix-systemd) | `task try IMAGE=ubuntu-nix-systemd USER=user` |                                                        |
