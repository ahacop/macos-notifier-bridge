class MacosNotifyBridge < Formula
  desc "TCP server that bridges notifications to macOS"
  homepage "https://github.com/ahacop/macos-notify-bridge"
  version "0.2.0"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/ahacop/macos-notify-bridge/releases/download/v0.2.0/macos-notify-bridge_0.2.0_darwin_arm64.tar.gz"
      sha256 "76dc231e397d6c8f9878b842ec9facc9682803105f1d5e78aac79e4d3511b932"
    else
      url "https://github.com/ahacop/macos-notify-bridge/releases/download/v0.2.0/macos-notify-bridge_0.2.0_darwin_x86_64.tar.gz"
      sha256 "15d8786bbcd181178403ca846f849c19d3bd2260e9213b0c424e3ec60ab0f5a5"
    end
  end

  depends_on "terminal-notifier"
  depends_on :macos

  def install
    bin.install "macos-notify-bridge"

    # Create the app bundle with custom icon in the formula's prefix
    # This keeps it out of /Applications but still accessible to terminal-notifier
    system "bash", "scripts/setup-app-bundle.sh", "macos-notify-bridge-icon.png", prefix
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
