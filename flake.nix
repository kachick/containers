{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    selfup = {
      url = "github:kachick/selfup/v1.2.1";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      selfup,
    }:
    let
      inherit (nixpkgs) lib;
      forAllSystems = lib.genAttrs lib.systems.flakeExposed;
    in
    {
      formatter = forAllSystems (system: nixpkgs.legacyPackages.${system}.nixfmt-tree);
      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShellNoCC {
            buildInputs =
              (with pkgs; [
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
              ])
              ++ [ selfup.packages.${system}.default ];
          };
        }
      );
    };
}
