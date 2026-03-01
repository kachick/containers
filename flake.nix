{
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
          cntr = pkgs.rustPlatform.buildRustPackage {
            pname = "cntr";
            version = "0.1.0";
            src = ./cntr;
            cargoLock.lockFile = ./cntr/Cargo.lock;
          };
          default = self.packages.${system}.cntr;
        }
      );
      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
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
