# Docker container image - Nix package manager on Ubuntu

## Usage

```bash
git clone git@github.com:kachick/dockerfile-ubuntu-nix.git
cd dockerfile-ubuntu-nix
docker build -t dockerfile-ubuntu-nix - < Dockerfile
docker run -it dockerfile-ubuntu-nix
```

```console
root@b130fdb85b72:/# nix --version
nix (Nix) 2.19.3
root@b130fdb85b72:/# nix run nixpkgs#hello
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
