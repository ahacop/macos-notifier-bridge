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

            # Parse arguments
            MESSAGE=""
            TITLE="NixOS Notification"
            SOUND=""

            # Simple argument parsing
            while [[ $# -gt 0 ]]; do
              case $1 in
                --sound)
                  SOUND="$2"
                  shift 2
                  ;;
                *)
                  if [ -z "$MESSAGE" ]; then
                    MESSAGE="$1"
                  elif [ "$TITLE" = "NixOS Notification" ]; then
                    TITLE="$1"
                  fi
                  shift
                  ;;
              esac
            done

            # Check required arguments
            if [ -z "$MESSAGE" ]; then
                echo "Usage: notify-macos <message> [title] [--sound <sound>]" >&2
                echo "Available sounds: Basso, Blow, Bottle, Frog, Funk, Glass, Hero, Morse, Ping, Pop, Purr, Sosumi, Submarine, Tink" >&2
                exit 1
            fi

            # Create JSON payload
            if [ -n "$SOUND" ]; then
              JSON=$(${pkgs.jq}/bin/jq -nc \
                --arg title "$TITLE" \
                --arg message "$MESSAGE" \
                --arg sound "$SOUND" \
                '{title: $title, message: $message, sound: $sound}')
            else
              JSON=$(${pkgs.jq}/bin/jq -nc \
                --arg title "$TITLE" \
                --arg message "$MESSAGE" \
                '{title: $title, message: $message}')
            fi

            # Send notification using netcat and capture response
            RESPONSE=$(echo "$JSON" | ${pkgs.netcat}/bin/nc -w 2 "$HOST_IP" "$PORT" 2>&1)

            # Check if response contains OK
            if echo "$RESPONSE" | grep -q "^OK"; then
              exit 0
            elif echo "$RESPONSE" | grep -q "^ERROR"; then
              echo "$RESPONSE" >&2
              exit 1
            else
              # If no response or connection failed
              echo "Failed to send notification" >&2
              exit 1
            fi
          '';

          default = self.packages.${system}.notify-macos;
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            golangci-lint
            goreleaser
            gnumake
            gotools # includes goimports, godoc, etc.
          ];
        };
      }
    );
}
