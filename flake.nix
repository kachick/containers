{
  nixConfig = {
    extra-substituters = [
      "https://cache.garnix.io"
    ];
    extra-trusted-public-keys = [
      "cache.garnix.io:CTFPyKSLcx5RMJKfLo5EEPUObbA78b0YQ2DTCJXqr9g="
    ];
  };

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs =
    {
      self,
      nixpkgs,
    }:
    let
      inherit (nixpkgs) lib;
      forAllSystems = lib.genAttrs lib.systems.flakeExposed;
    in
    {
      formatter = forAllSystems (system: nixpkgs.legacyPackages.${system}.nixfmt-tree);
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          container-try = pkgs.rustPlatform.buildRustPackage {
            pname = "container-try";
            version = "0.1.0";
            src = ./container-try;
            cargoLock.lockFile = ./container-try/Cargo.lock;

            doInstallCheck = true;
            installCheckPhase = ''
              runHook preInstallCheck
              "$out/bin/container-try" --help
              runHook postInstallCheck
            '';

            meta = {
              mainProgram = "container-try";
            };
          };
          default = self.packages.${system}.container-try;
        }
      );
      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            env = {
              # Fix nixd pkgs versions in the inlay hints
              NIX_PATH = "nixpkgs=${pkgs.path}";

              # Workaround for rust-analyzer error: "ERROR can't load standard library, try installing `rust-src`"
              RUST_SRC_PATH = "${pkgs.rustPlatform.rustLibSrc}";
            };

            buildInputs = with pkgs; [
              bashInteractive
              findutils # xargs
              nixfmt
              nixfmt-tree
              nixd
              go-task

              shfmt
              shellcheck

              buildah
              trivy

              dprint
              typos

              rustc
              cargo
              rust-analyzer
              rustfmt
              clippy
            ];
          };
        }
      );
    };
}
