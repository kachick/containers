{
  inputs = {
    nixpkgs.url = "https://channels.nixos.org/nixos-unstable/nixexprs.tar.zst";
    cntr = {
      url = "github:Mic92/cntr/2.1.0";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.treefmt-nix.follows = "";
      inputs.fenix.follows = "";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      cntr,
    }:
    let
      inherit (nixpkgs) lib;
      forAllSystems = lib.genAttrs (
        lib.intersectLists [
          "x86_64-linux"
          "aarch64-linux"
        ] lib.systems.flakeExposed
      );
    in
    {
      formatter = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        pkgs.writeShellScriptBin "dprint-fmt" ''
          exec "${lib.getExe pkgs.dprint}" fmt "$@"
        ''
      );
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          cntrPackages = import ./nix/cntr.nix {
            inherit pkgs cntr;
          };
        in
        {
          systemd-container-shell = pkgs.rustPlatform.buildRustPackage {
            pname = "systemd-container-shell";
            version = with builtins; (fromTOML (readFile ./systemd-container-shell/Cargo.toml)).package.version;
            src = ./systemd-container-shell;
            cargoLock.lockFile = ./systemd-container-shell/Cargo.lock;

            doInstallCheck = true;
            installCheckPhase = ''
              runHook preInstallCheck
              "$out/bin/systemd-container-shell" --help
              runHook postInstallCheck
            '';

            meta = {
              mainProgram = "systemd-container-shell";
            };
          };
          inherit (cntrPackages) cntr podman-cntr;
          default = self.packages.${system}.systemd-container-shell;
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

            buildInputs =
              (with pkgs; [
                bashInteractive
                nixd
                go-task

                shfmt
                shellcheck

                buildah
                trivy

                zizmor

                dprint
                typos

                rustc
                cargo
                rust-analyzer
                rustfmt
                clippy
              ])
              ++ [
                self.packages.${system}.cntr
                self.packages.${system}.podman-cntr
              ];
          };
        }
      );
    };
}
