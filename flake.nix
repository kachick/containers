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
          systemd-container-shell = pkgs.buildGoModule {
            pname = "systemd-container-shell";
            version = "0.2.0";
            src = ./systemd-container-shell;
            vendorHash = null;

            env.CGO_ENABLED = 0;

            nativeBuildInputs = [ pkgs.makeWrapper ];

            # Bundle fzf in PATH so `nix run` works without fzf installed on the host.
            postInstall = ''
              wrapProgram "$out/bin/systemd-container-shell" \
                --suffix PATH : "${lib.makeBinPath [ pkgs.fzf ]}"
            '';

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

                go
                gopls
                fzf
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
