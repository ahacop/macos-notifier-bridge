class MacosNotifyBridge < Formula
  desc "TCP server that bridges notifications to macOS"
  homepage "https://github.com/ahacop/macos-notify-bridge"
  version "0.6.0"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/ahacop/macos-notify-bridge/releases/download/v0.6.0/macos-notify-bridge_0.6.0_darwin_arm64.tar.gz"
      sha256 "57a7ff918c1bfeaf70726424e2e84d331d47e97a93eab36d785758c49e944784"
    else
      url "https://github.com/ahacop/macos-notify-bridge/releases/download/v0.6.0/macos-notify-bridge_0.6.0_darwin_x86_64.tar.gz"
      sha256 "f04949b99a4486c513ffea6ecab11a8c898b2b2ab9681f67810ca61094cb0416"
    end
  end

  depends_on "terminal-notifier"
  depends_on :macos

  def install
    bin.install "macos-notify-bridge"

    # Stop the service if it's running (for smooth upgrades)
    begin
      services_output = `brew services list 2>/dev/null`
      if services_output.include?("macos-notify-bridge") && services_output.include?("started")
        system "brew", "services", "stop", "macos-notify-bridge"
        ohai "Stopped running service for upgrade"
      end
    rescue
      # Ignore errors if brew services is not available
    end

    # Create the app bundle in Homebrew's prefix to avoid permission issues
    # The app will be installed to #{prefix}/Applications/
    app_bundle_dir = prefix/"Applications"
    app_bundle_dir.mkpath

    system "bash", "scripts/setup-app-bundle.sh", "macos-notify-bridge-icon.png", app_bundle_dir.to_s
  end

  def post_install
    # Path to the app bundle in Homebrew's prefix
    homebrew_app_path = prefix/"Applications/MacOS Notify Bridge.app"
    system_app_path = "/Applications/MacOS Notify Bridge.app"

    # Try to create a symlink in /Applications (may fail due to permissions)
    if File.exist?(homebrew_app_path) && !File.exist?(system_app_path)
      begin
        File.symlink(homebrew_app_path, system_app_path)
        ohai "Created symlink in /Applications"
      rescue Errno::EACCES, Errno::EPERM
        # Expected when running without sudo - we'll handle this in caveats
      end
    end

    # Register the app with Launch Services (use whichever path exists)
    app_to_register = File.exist?(system_app_path) ? system_app_path : homebrew_app_path
    if File.exist?(app_to_register)
      system "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister",
             "-f", "-r", "-domain", "local", "-domain", "user", app_to_register

      ohai "App bundle registered with macOS from #{app_to_register}"
    end

    # Restart the service if it was running before upgrade
    begin
      services_output = `brew services list 2>/dev/null`
      if services_output.include?("macos-notify-bridge") && services_output.include?("stopped")
        system "brew", "services", "start", "macos-notify-bridge"
        ohai "Restarted service after upgrade"
      end
    rescue
      # Ignore errors if brew services is not available
    end
  end

  def post_uninstall
    # Remove the symlink or actual app from /Applications
    system_app_path = "/Applications/MacOS Notify Bridge.app"
    if File.exist?(system_app_path)
      # Check if it's a symlink pointing to our Homebrew installation
      if File.symlink?(system_app_path)
        File.unlink(system_app_path)
        ohai "Removed symlink from #{system_app_path}"
      else
        # It's a real directory - only remove if we have permission
        begin
          FileUtils.rm_rf(system_app_path)
          ohai "Removed #{system_app_path}"
        rescue Errno::EACCES, Errno::EPERM
          opoo "Could not remove #{system_app_path} - insufficient permissions"
        end
      end
    end
  end

  service do
    run [opt_bin/"macos-notify-bridge"]
    keep_alive true
    log_path var/"log/macos-notify-bridge.log"
    error_log_path var/"log/macos-notify-bridge.log"
    environment_variables PORT: "9876", PATH: std_service_path_env
  end

  test do
    # Test that the binary runs and shows version
    assert_match "macos-notify-bridge version", shell_output("#{bin}/macos-notify-bridge --version")

    # Test that it can start (will fail without terminal-notifier in CI, but that's ok)
    pid = fork do
      exec "#{bin}/macos-notify-bridge", "--port", "19876"
    end
    sleep 2

    begin
      # Test connection
      require "socket"
      TCPSocket.new("localhost", 19876).close
    ensure
      Process.kill("TERM", pid)
      Process.wait(pid)
    end
  end

  def caveats
    homebrew_app = "#{opt_prefix}/Applications/MacOS Notify Bridge.app"
    system_app = "/Applications/MacOS Notify Bridge.app"

    <<~EOS
      To start macos-notify-bridge as a service:
        brew services start macos-notify-bridge

      Or run it manually:
        macos-notify-bridge

      The service will listen on port 9876 by default.

      The MacOS Notify Bridge app bundle is installed at:
        #{homebrew_app}

      To make the app available in /Applications (optional):
        sudo ln -s "#{homebrew_app}" "#{system_app}"

      This allows using the custom sender ID with terminal-notifier:
        terminal-notifier -sender com.ahacop.macos-notify-bridge -title "Test" -message "Hello!"

      Test the bridge with:
        echo '{"title":"Test","message":"Hello from Homebrew!"}' | nc localhost 9876
    EOS
  end
end
