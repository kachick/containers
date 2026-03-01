# How to develop

## NOTE

- <https://docs.github.com/en/actions/publishing-packages/publishing-docker-images>
- <https://gallery.ecr.aws/ubuntu/ubuntu>
- <https://gallery.ecr.aws/lts/ubuntu>
- <https://github.com/kachick/wait-other-jobs/blob/6a50464dd0f6a3cbde8ac50687ee246830f99075/.devcontainer/Dockerfile>
- <https://github.com/kachick/wait-other-jobs/pull/517>
- [podman-remote can't use --latest flag for ps](https://github.com/kachick/dotfiles/issues/448)

## Why?

- `pkgs.dockerTools` does not fit here because this repo provides a full Nix environment with systemd and multi-user setup, not just a package.
