{
  description = "A Nix-flake-based Go development environment";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = import nixpkgs {
          inherit system;
        };
        isAarch64Linux = system == "aarch64-linux";
      in {
        packages = {
          notify-macos = pkgs.writeShellScriptBin "notify-macos" ''
            # Send notification to macOS host from NixOS VM
            
            # Get host IP from environment or use default
            HOST_IP="''${MACOS_HOST_IP:-192.168.178.124}"
            PORT=9876
            
            # Check arguments
            if [ $# -lt 1 ]; then
                echo "Usage: notify-macos <message> [title]" >&2
                exit 1
            fi
            
            MESSAGE="$1"
            TITLE="''${2:-NixOS Notification}"
            
            # Create JSON payload
            JSON=$(${pkgs.jq}/bin/jq -n \
              --arg title "$TITLE" \
              --arg message "$MESSAGE" \
              '{title: $title, message: $message}')
            
            # Send notification using netcat
            echo "$JSON" | ${pkgs.netcat}/bin/nc -w 2 "$HOST_IP" "$PORT"
          '';

          default = self.packages.${system}.notify-macos;
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs;
            [
              go
              golangci-lint
              goreleaser
              gnumake
              gotools  # includes goimports, godoc, etc.
            ];
        };
      }
    );
}
