# macOS Notify Bridge

A lightweight TCP server that bridges network notification requests to native macOS notifications using `terminal-notifier`.

## Features

- 🚀 Simple TCP server listening for JSON notification requests
- 🔔 Native macOS notifications via `terminal-notifier`
- 🔧 Configurable port and host binding
- 🌐 Automatic VM bridge interface detection for VM connectivity
- 📝 Verbose logging mode
- 🛡️ Graceful shutdown handling
- 🍺 Easy installation via Homebrew
- 🎯 Zero dependencies (except `terminal-notifier`)

## Installation

### Via Homebrew (Recommended)

```bash
# Add the tap with the full repository URL
brew tap ahacop/macos-notify-bridge https://github.com/ahacop/macos-notify-bridge

# Install the formula
brew install ahacop/macos-notify-bridge/macos-notify-bridge

# Start as a service
brew services start macos-notify-bridge
```

### From Source

```bash
# Clone the repository
git clone https://github.com/ahacop/macos-notify-bridge.git
cd macos-notify-bridge

# Build
go build -o macos-notify-bridge

# Run
./macos-notify-bridge
```

### Download Binary

Download the latest release from the [releases page](https://github.com/ahacop/macos-notify-bridge/releases).

## Prerequisites

- macOS
- `terminal-notifier` installed (`brew install terminal-notifier`)

## Usage

### Starting the Server

```bash
# Default configuration (port 9876, localhost only)
macos-notify-bridge

# Custom port
macos-notify-bridge --port 8080
# or
macos-notify-bridge -p 8080

# Automatically detect and bind to VM bridge interfaces
macos-notify-bridge --auto-detect-bridges
# or
macos-notify-bridge -a

# Bind to specific addresses (can be used multiple times)
macos-notify-bridge --bind localhost --bind 192.168.122.1
# or
macos-notify-bridge -b localhost -b 192.168.122.1

# Enable verbose logging
macos-notify-bridge --verbose
# or
macos-notify-bridge -v

# Show version
macos-notify-bridge --version
```

### Environment Variables

You can also set the port using the `PORT` environment variable:

```bash
PORT=8080 macos-notify-bridge
```

### Sending Notifications

The server expects JSON requests in the following format:

```json
{
  "title": "Notification Title",
  "message": "Notification message content"
}
```

#### Using netcat

```bash
echo '{"title":"Test","message":"Hello from netcat!"}' | nc localhost 9876
```

#### Using curl

```bash
echo '{"title":"Test","message":"Hello from curl!"}' | curl -X POST --data-binary @- telnet://localhost:9876
```

#### Using Python

```python
import socket
import json

def send_notification(title, message, host='localhost', port=9876):
    data = json.dumps({'title': title, 'message': message}) + '\n'

    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.connect((host, port))
        s.sendall(data.encode())
        response = s.recv(1024).decode().strip()
        return response

# Send a notification
result = send_notification("Python Alert", "Task completed successfully!")
print(f"Server response: {result}")
```

#### Using Node.js

```javascript
const net = require("net");

function sendNotification(title, message, host = "localhost", port = 9876) {
  return new Promise((resolve, reject) => {
    const client = net.createConnection({ port, host }, () => {
      const data = JSON.stringify({ title, message }) + "\n";
      client.write(data);
    });

    client.on("data", (data) => {
      resolve(data.toString().trim());
      client.end();
    });

    client.on("error", reject);
  });
}

// Send a notification
sendNotification("Node.js Alert", "Build completed!")
  .then((response) => console.log("Server response:", response))
  .catch((err) => console.error("Error:", err));
```

#### Using Bash Function

Add this to your `.bashrc` or `.zshrc`:

```bash
notify() {
  local title="${1:-Notification}"
  local message="${2:-No message provided}"
  echo "{\"title\":\"$title\",\"message\":\"$message\"}" | nc localhost 9876
}

# Usage
notify "Build Complete" "Your project has been built successfully!"
```

## Configuration

### Command Line Flags

- `--port, -p`: TCP port to listen on (default: 9876)
- `--auto-detect-bridges, -a`: Automatically detect and bind to VM bridge interfaces
- `--bind, -b`: Bind to specific address (can be used multiple times)
- `--verbose, -v`: Enable verbose logging
- `--version`: Display version information

### Input Limits

To prevent abuse, the following size limits are enforced:

- **Title**: Maximum 256 characters
- **Message**: Maximum 1024 characters
- **Sound**: Maximum 64 characters

### VM Connectivity

The server supports automatic detection of VM bridge interfaces, allowing notifications to be sent from virtual machines running on the same host:

```bash
# Enable automatic VM bridge detection
macos-notify-bridge --auto-detect-bridges
```

This will automatically bind to common VM bridge interfaces:

- `virbr*` - libvirt/KVM/QEMU bridges
- `vmnet*` - VMware bridges
- `vboxnet*` - VirtualBox host-only networks
- `docker0` - Docker default bridge
- `br-*` - Docker custom bridges

You can also manually specify which addresses to bind to:

```bash
# Bind to specific interfaces
macos-notify-bridge --bind localhost --bind 192.168.122.1
```

When running with `--verbose`, the server will log which interfaces it's listening on:

```
Server listening on localhost:9876
Server listening on 192.168.122.1:9876
Server listening on 172.17.0.1:9876
```

### As a Service

When installed via Homebrew, the service will:

- Start automatically on system boot
- Log to `/usr/local/var/log/macos-notify-bridge.log`
- Restart automatically if it crashes

To manage the service:

```bash
# Start the service
brew services start macos-notify-bridge

# Stop the service
brew services stop macos-notify-bridge

# Restart the service
brew services restart macos-notify-bridge

# Check service status
brew services list
```

## Security Considerations

- By default, the server binds exclusively to localhost (127.0.0.1) for security
- When using `--auto-detect-bridges`, the server will also bind to VM bridge interfaces (virbr*, vmnet*, vboxnet*, docker0, br-*)
- All input fields have size limits to prevent abuse
- The server includes a 30-second timeout for connections to prevent hanging
- No authentication is implemented - suitable for local-only use
- For additional security, consider:
  - Running with restricted user permissions
  - Implementing rate limiting via firewall rules
  - Monitoring logs for suspicious activity
  - Using `--bind` to explicitly specify allowed interfaces instead of auto-detection

## Development

[![CI](https://github.com/ahacop/macos-notify-bridge/actions/workflows/ci.yml/badge.svg)](https://github.com/ahacop/macos-notify-bridge/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/ahacop/macos-notify-bridge/branch/main/graph/badge.svg)](https://codecov.io/gh/ahacop/macos-notify-bridge)

### Prerequisites

- Go 1.21 or later
- `make` command
- `golangci-lint` (installed automatically by `make deps`)

### Building

```bash
# Build the binary
make build

# Build with custom version
make build VERSION=1.2.3

# Clean build artifacts
make clean
```

### Testing

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests only
make test-integration

# Run tests with coverage report
make test-coverage
```

### Code Quality

```bash
# Format code
make fmt

# Check if code is properly formatted
make fmt-check

# Run linter
make lint

# Run everything (fmt, lint, test, build)
make all
```

### Development Setup

```bash
# Clone the repository
git clone https://github.com/ahacop/macos-notify-bridge.git
cd macos-notify-bridge

# Install development dependencies
make deps

# Run tests to verify setup
make test

# Build and install locally
make install
```

### Manual Testing

```bash
# Start the server
./macos-notify-bridge -v

# In another terminal, test with netcat
echo '{"title":"Test","message":"Development test"}' | nc localhost 9876

# Test error handling
echo 'invalid json' | nc localhost 9876
echo '{"title":"Missing message"}' | nc localhost 9876

# Use the Python test client
python3 test_client.py
```

### Creating a Release

This project uses [goreleaser](https://goreleaser.com/) for releases:

```bash
# Test release process without publishing
make release-dry

# Tag a new version
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0

# GitHub Actions will automatically create the release
```

### CI/CD

The project uses GitHub Actions for continuous integration:

- **CI Pipeline**: Runs on every push and pull request

  - Linting with golangci-lint
  - Unit and integration tests
  - Multi-platform builds
  - Code coverage reporting

- **Release Pipeline**: Runs on version tags (v*.*.\*)
  - Automated release creation
  - Binary distribution
  - Homebrew formula updates

## Troubleshooting

### Server won't start

- Check if the port is already in use: `lsof -i :9876`
- Ensure `terminal-notifier` is installed: `which terminal-notifier`

### Notifications not appearing

- Check macOS notification settings for Terminal
- Run with `--verbose` flag to see detailed logs
- Ensure notification JSON is properly formatted

### Connection refused

- Verify the server is running: `ps aux | grep macos-notify-bridge`
- Check firewall settings if connecting from another machine

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built on top of [terminal-notifier](https://github.com/julienXX/terminal-notifier)
- Inspired by the need for simple cross-platform notification solutions
