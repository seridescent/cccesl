{
  description = "claude code cache expiry statusline";

  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [ "x86_64-linux" "aarch64-linux" "aarch64-darwin" "x86_64-darwin" ];
      perSystem = { lib, pkgs, ... }: {
        packages.default = pkgs.buildGo125Module {
          pname = "cccesl";
          version = "0.1.0";
          src = ./.;
          vendorHash = null;

          meta = {
            description = "claude code cache expiry statusline";
            homepage = "https://github.com/seridescent/cccesl";
            license = lib.licenses.mit;
          };
        };

        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.go_1_25
            pkgs.gopls
          ];
        };
      };
    };
}
