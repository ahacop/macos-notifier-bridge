// Package testutil provides testing utilities for the macOS notification bridge.
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateMockTerminalNotifier creates a mock terminal-notifier executable
// that logs notifications instead of displaying them
func CreateMockTerminalNotifier(dir string) (string, error) {
	mockPath := filepath.Join(dir, "terminal-notifier")
	logPath := filepath.Join(dir, "notifications.log")

	mockScript := fmt.Sprintf(`#!/bin/sh
# Mock terminal-notifier for testing

# Parse arguments
TITLE=""
MESSAGE=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    -title)
      TITLE="$2"
      shift 2
      ;;
    -message)
      MESSAGE="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

# Log the notification
echo "$(date '+%%Y-%%m-%%d %%H:%%M:%%S') - Title: $TITLE, Message: $MESSAGE" >> %s

# Exit successfully
exit 0
`, logPath)

	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		return "", fmt.Errorf("failed to create mock terminal-notifier: %w", err)
	}

	return mockPath, nil
}

// CreateFailingMockTerminalNotifier creates a mock that always fails
func CreateFailingMockTerminalNotifier(dir string) (string, error) {
	mockPath := filepath.Join(dir, "terminal-notifier")

	mockScript := `#!/bin/sh
# Mock terminal-notifier that always fails
echo "Mock error: terminal-notifier failed" >&2
exit 1
`

	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		return "", fmt.Errorf("failed to create failing mock terminal-notifier: %w", err)
	}

	return mockPath, nil
}

// ReadNotificationLog reads the mock notification log
func ReadNotificationLog(dir string) (string, error) {
	logPath := filepath.Join(dir, "notifications.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}
