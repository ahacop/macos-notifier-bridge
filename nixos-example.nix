# Example NixOS configuration showing how to use the notify-macos script from this flake
#
# In your NixOS configuration, you can reference this flake and use the notify-macos script
# in several ways:
{
  config,
  pkgs,
  ...
}: {
  # Method 1: Add as a system package
  # First, add this flake as an input in your system flake:
  # inputs.macos-notifier-bridge.url = "github:ahacop/macos-notifier-bridge";
  #
  # Then in your configuration:
  # environment.systemPackages = [
  #   inputs.macos-notifier-bridge.packages.${pkgs.system}.notify-macos
  # ];

  # Method 2: Use in a NixOS module with customization
  # This example shows how to create a wrapper with custom defaults
  environment.systemPackages = let
    macos-notifier-bridge = builtins.getFlake "github:ahacop/macos-notifier-bridge";
    notify-macos-custom = pkgs.writeShellScriptBin "notify-macos" ''
      # Custom wrapper that sets the host IP from NixOS configuration
      export MACOS_HOST_IP="${config.networking.defaultGateway.address or "192.168.178.124"}"
      exec ${macos-notifier-bridge.packages.${pkgs.system}.notify-macos}/bin/notify-macos "$@"
    '';
  in [
    notify-macos-custom
  ];

  # Method 3: Create a systemd service that uses the script
  # systemd.services.example-notification = {
  #   description = "Example service that sends notifications";
  #   after = [ "network.target" ];
  #   serviceConfig = {
  #     Type = "oneshot";
  #     ExecStart = "${inputs.macos-notifier-bridge.packages.${pkgs.system}.notify-macos}/bin/notify-macos 'System started' 'NixOS Boot'";
  #   };
  # };

  # Optional: Set the macOS host IP as a system-wide environment variable
  # environment.variables.MACOS_HOST_IP = "192.168.1.100";
}
