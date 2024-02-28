# Nix package manager on Ubuntu

## Usage

```bash
git clone git@github.com:kachick/containers.git
cd containers
podman build --tag ubuntu-nix --file ./images/ubuntu-nix/Containerfile .
podman run -it ubuntu-nix
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
