# How to develop

## NOTE

- <https://docs.github.com/en/actions/publishing-packages/publishing-docker-images>
- <https://gallery.ecr.aws/ubuntu/ubuntu>
- <https://gallery.ecr.aws/lts/ubuntu>
- <https://github.com/kachick/wait-other-jobs/blob/6a50464dd0f6a3cbde8ac50687ee246830f99075/.devcontainer/Dockerfile>
- <https://github.com/kachick/wait-other-jobs/pull/517>
- [podman-remote can't use --latest flag for ps](https://github.com/kachick/dotfiles/issues/448)

## Why?

- `pkgs.dockerTools` will not fit at here. Because this repo provides a ready-to-use Nix environment with systemd and multi-user setup, not just distribute a package.
