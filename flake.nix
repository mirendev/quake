{
  description = "Quake - A Make-like build tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        packages.default = pkgs.buildGoModule {
          pname = "quake";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-jBji7VbfndfYRHPYx/dIGCqAu4nttJ0EOuhrBPfwseU=";
          subPackages = ["."];

          ldflags = [
            "-s"
            "-w"
            "-X main.commit=${self.rev or "dirty"}"
          ];

          preBuild = ''
            export CGO_ENABLED=0
          '';
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            golangci-lint
          ];
        };

        formatter = pkgs.nixfmt-rfc-style;
      }
    );
}
