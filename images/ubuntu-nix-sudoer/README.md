# Nix package manager on Ubuntu - sudoer

## Usage

```bash
podman run --rm -it ghcr.io/kachick/ubuntu-nix-sudoer:latest
```

```console
user@1fa1d02b83b7:/$ nix --version
nix (Nix) 2.20.3
user@1fa1d02b83b7:/$ nix run nixpkgs#hello
Hello, world!
user@1fa1d02b83b7:/$ ps -ef | grep systemd
user          26       1  0 17:23 pts/0    00:00:00 grep --color=auto systemd
user@1fa1d02b83b7:/$ sudo --version
Sudo version 1.9.9
Sudoers policy plugin version 1.9.9
Sudoers file grammar version 48
Sudoers I/O plugin version 1.9.9
Sudoers audit plugin version 1.9.9
```

## Motivation

- [nixos/nix](https://hub.docker.com/r/nixos/nix) is not NixOS, but it does not have some basic tools as `groupadd`\
  ref: <https://stackoverflow.com/questions/75653182/why-do-some-official-nix-docker-containers-not-have-the-nixos-rebuild-command>
- I want flake by default
- I prefer ubuntu rather than [alpine](https://hub.docker.com/r/nixos/nix) for daily use

## Note

- systemd pattern didn't work for running home-manager in non root user
