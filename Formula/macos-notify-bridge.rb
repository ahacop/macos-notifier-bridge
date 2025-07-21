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

    # Create the app bundle with custom icon in /Applications
    # This ensures it's properly registered with macOS
    app_path = "/Applications/MacOS Notify Bridge.app"
    
    # Remove existing app bundle if it exists (for upgrades)
    if File.exist?(app_path)
      FileUtils.rm_rf(app_path)
      ohai "Removing existing app bundle for upgrade"
    end
    
    system "bash", "scripts/setup-app-bundle.sh", "macos-notify-bridge-icon.png", "/Applications"
  end

  def post_install
    # Reset Launch Services database to ensure the app is properly registered
    # This is especially important after upgrades
    app_path = "/Applications/MacOS Notify Bridge.app"
    if File.exist?(app_path)
      system "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister", 
             "-f", "-r", "-domain", "local", "-domain", "user", app_path
      
      ohai "App bundle registered with macOS"
    end
  end

  def post_uninstall
    # Remove the app bundle from /Applications
    app_path = "/Applications/MacOS Notify Bridge.app"
    if File.exist?(app_path)
      FileUtils.rm_rf(app_path)
      ohai "Removed #{app_path}"
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
    <<~EOS
      To start macos-notify-bridge as a service:
        brew services start macos-notify-bridge

      Or run it manually:
        macos-notify-bridge

      The service will listen on port 9876 by default.

      Test it with:
        echo '{"title":"Test","message":"Hello from Homebrew!"}' | nc localhost 9876
    EOS
  end
end
