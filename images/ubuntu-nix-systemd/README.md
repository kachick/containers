# Nix package manager on Ubuntu - systemd

## Usage

Keep systemd in background and use it with another user

```bash
podman run --rm ghcr.io/kachick/ubuntu-nix-systemd:latest &
podman exec --user=user -it "$(podman ps --sort=created --format {{.Names}} | tail -1)" bash
```

Make sure non root and non sudoers can run nix features

```console
user@541fad9ac911:/$ nix --version
nix (Nix) 2.20.3
user@541fad9ac911:/$ nix run nixpkgs#hello
Hello, world!
user@541fad9ac911:/$ ps -ef | grep systemd
root           1       0  0 10:53 ?        00:00:00 /bin/systemd --system
root          15       1  0 10:53 ?        00:00:00 /lib/systemd/systemd-journald
systemd+      22       1  0 10:53 ?        00:00:00 /lib/systemd/systemd-resolved
user         770      26  0 10:55 pts/0    00:00:00 grep --color=auto systemd
user@541fad9ac911:/$ sudo
bash: sudo: command not found
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
