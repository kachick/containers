# Ubuntu - systemd

Base Ubuntu 26.04 image with `systemd` and a non-root `visitor` account, without Nix pre-installed.
Used as the base for [ubuntu-nix-systemd](../ubuntu-nix-systemd) and layered Nix images that provide their own Nix store and daemon config.

## Usage

```bash
# Build from local
nix run github:kachick/containers#systemd-container-shell -- build ubuntu-systemd

# Pull from registry
nix run github:kachick/containers#systemd-container-shell -- pull --image ghcr.io/kachick/ubuntu-26.04-systemd
```
