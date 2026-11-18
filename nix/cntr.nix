{
  pkgs,
  cntr,
}:
let
  # Pinning cntr from the flake input
  # Upstream nixpkgs PR https://github.com/NixOS/nixpkgs/pull/548453 is stuck due to test failures.
  # Integration tests require writing to /proc/self/setgroups and unsharing user namespaces,
  # which fails in sandboxed builds (e.g. GitHub Actions runners), so disable checks.
  #
  # Patches:
  # - cntr-fork-into-pidns.patch: setns(CLONE_NEWPID) does not change the caller's PID namespace,
  #   causing setpgid EPERM in interactive bash. Fork before execve to enter the container PID namespace.
  # Call package directly from source to avoid deprecated stdenv.isDarwin evaluation warning in upstream flake.nix
  cntrPkg =
    (pkgs.callPackage "${cntr}/default.nix" {
      self = cntr;
    }).overrideAttrs
      (old: {
        patches = (old.patches or [ ]) ++ [
          ./patches/cntr-fork-into-pidns.patch
        ];
        doCheck = false;
      });

  # cntr (https://github.com/Mic92/cntr):
  # A debugging tool that brings host tools into running container namespaces.
  #
  # Why use cntr:
  # - Ideal for debugging minimal or shell-less images (e.g., distroless, scratch)
  #   where 'podman exec' fails with "executable file not found in $PATH".
  # - Note on filesystem layout: '/' remains the host filesystem so host binaries
  #   run without library mismatch. The container root is mounted at the initial
  #   working directory (CNTR_BASE_DIR). Use relative paths (e.g., 'cat etc/os-release')
  #   to inspect container files.
  #
  # Example usage with distroless:
  #   podman run -d --name test -v $(nix build --no-link --print-out-paths nixpkgs#pkgsStatic.busybox)/bin/busybox:/sleep:ro gcr.io/distroless/base-debian13 /sleep 3600
  #   podman-cntr attach test
  #
  # Why not need a NixOS module (security.wrappers setcap) for cntr:
  # Upstream docs suggest root or setcap ('cap_sys_admin,cap_sys_chroot=ep') for Docker/rootful setups.
  # However, rootless Podman runs in a separate user namespace. Root or setcap binaries cannot see or enter
  # user-owned namespaces cleanly (see https://github.com/Mic92/cntr/issues/172).
  # Running cntr via 'podman unshare' enters the container's user namespace with full capabilities,
  # and setting CNTR_BASE_DIR to a user-writable path avoids needing host root or setcap permissions entirely.
  #
  # Why supply --effective-user by default on 'attach':
  # Without --effective-user, cntr inherits container environment (HOME=/root). In rootless Podman,
  # unprivileged users cannot read host /root (0700), triggering permission errors on .bash_profile/.bash_logout.
  # Supplying the host user sets HOME to user's home directory, fixing permission errors and loading Home Manager shell configs.
  podmanCntr = pkgs.writeShellApplication {
    name = "podman-cntr";
    runtimeInputs = [
      pkgs.podman
      cntrPkg
    ];
    text = ''
      CNTR_BASE_DIR="''${CNTR_BASE_DIR:-''${XDG_RUNTIME_DIR:-/tmp}/cntr}"
      export CNTR_BASE_DIR
      mkdir -p "$CNTR_BASE_DIR"

      if [ "''${1:-}" = "attach" ]; then
        has_effective_user=0
        for arg in "$@"; do
          case "$arg" in
            --effective-user|--effective-user=*)
              has_effective_user=1
              break
              ;;
          esac
        done
        if [ "$has_effective_user" -eq 0 ]; then
          shift
          exec podman unshare env CNTR_BASE_DIR="$CNTR_BASE_DIR" cntr attach --effective-user "''${USER:-$(id -un)}" "$@"
        fi
      fi

      exec podman unshare env CNTR_BASE_DIR="$CNTR_BASE_DIR" cntr "$@"
    '';
    meta.description = "Attach to rootless podman containers using cntr";
  };
in
[
  cntrPkg
  podmanCntr
]
