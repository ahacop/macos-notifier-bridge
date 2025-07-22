//go:build integration
// +build integration

// Package main provides integration tests for the macOS notification bridge server.
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestIntegrationFullServerLifecycle tests the complete server lifecycle
func TestIntegrationFullServerLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "test-server")

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build binary: %v\nOutput: %s", err, output)
	}

	// Find an available port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Logf("failed to close listener: %v", err)
	}

	// Create mock terminal-notifier
	mockNotifierPath := filepath.Join(tempDir, "terminal-notifier")
	mockNotifierContent := `#!/bin/sh
echo "Mock notification: $*" >> ` + filepath.Join(tempDir, "notifications.log") + `
exit 0
`
	if err := os.WriteFile(mockNotifierPath, []byte(mockNotifierContent), 0755); err != nil {
		t.Fatalf("failed to create mock terminal-notifier: %v", err)
	}

	// Start the server with mock terminal-notifier in PATH
	cmd := exec.Command(binaryPath, "-p", fmt.Sprintf("%d", port), "-v")
	cmd.Env = append(os.Environ(), fmt.Sprintf("PATH=%s:%s", tempDir, os.Getenv("PATH")))

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	t.Cleanup(func() {
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Logf("failed to send SIGTERM: %v", err)
		}
		if err := cmd.Wait(); err != nil && !strings.Contains(err.Error(), "signal: terminated") {
			t.Logf("wait error: %v", err)
		}
	})

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Test 1: Send valid notification
	t.Run("valid notification", func(t *testing.T) {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		t.Cleanup(func() {
			if err := conn.Close(); err != nil {
				t.Logf("failed to close connection: %v", err)
			}
		})

		notification := NotificationRequest{
			Title:   "Integration Test",
			Message: "This is a test notification",
		}
		data, _ := json.Marshal(notification)

		if _, err := conn.Write(append(data, '\n')); err != nil {
			t.Fatalf("failed to send notification: %v", err)
		}

		response := make([]byte, 1024)
		n, err := conn.Read(response)
		if err != nil {
			t.Fatalf("failed to read response: %v", err)
		}

		if !strings.Contains(string(response[:n]), "OK") {
			t.Errorf("expected OK response, got: %s", string(response[:n]))
		}
	})

	// Test 2: Send invalid JSON
	t.Run("invalid json", func(t *testing.T) {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		t.Cleanup(func() {
			if err := conn.Close(); err != nil {
				t.Logf("failed to close connection: %v", err)
			}
		})

		if _, err := conn.Write([]byte("invalid json\n")); err != nil {
			t.Fatalf("failed to send data: %v", err)
		}

		response := make([]byte, 1024)
		n, err := conn.Read(response)
		if err != nil {
			t.Fatalf("failed to read response: %v", err)
		}

		if !strings.Contains(string(response[:n]), "ERROR") {
			t.Errorf("expected ERROR response, got: %s", string(response[:n]))
		}
	})

	// Test 3: Valid notification with sound
	t.Run("valid notification with sound", func(t *testing.T) {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		t.Cleanup(func() {
			if err := conn.Close(); err != nil {
				t.Logf("failed to close connection: %v", err)
			}
		})

		notification := NotificationRequest{
			Title:   "Sound Test",
			Message: "This notification has sound",
			Sound:   "Hero",
		}
		data, _ := json.Marshal(notification)

		if _, err := conn.Write(append(data, '\n')); err != nil {
			t.Fatalf("failed to send notification: %v", err)
		}

		response := make([]byte, 1024)
		n, err := conn.Read(response)
		if err != nil {
			t.Fatalf("failed to read response: %v", err)
		}

		if !strings.Contains(string(response[:n]), "OK") {
			t.Errorf("expected OK response, got: %s", string(response[:n]))
		}
	})

	// Test 4: Connection timeout
	t.Run("connection timeout", func(t *testing.T) {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		t.Cleanup(func() {
			if err := conn.Close(); err != nil {
				t.Logf("failed to close connection: %v", err)
			}
		})

		// Don't send anything, just wait
		time.Sleep(100 * time.Millisecond)

		// Connection should still be open (timeout is 30s)
		if _, err := conn.Write([]byte("test")); err != nil {
			t.Logf("connection closed as expected for timeout test")
		}
	})

	// Test 5: Graceful shutdown
	t.Run("graceful shutdown", func(t *testing.T) {
		// Send SIGTERM
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Fatalf("failed to send SIGTERM: %v", err)
		}

		// Wait for process to exit
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case err := <-done:
			if err != nil && !strings.Contains(err.Error(), "signal: terminated") {
				t.Errorf("unexpected error on shutdown: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("server did not shut down within 5 seconds")
		}
	})

	// Verify notifications were logged
	logPath := filepath.Join(tempDir, "notifications.log")
	if logData, err := os.ReadFile(logPath); err == nil {
		if !strings.Contains(string(logData), "Integration Test") {
			t.Errorf("expected notification not found in log: %s", string(logData))
		}
		if !strings.Contains(string(logData), "Sound Test") {
			t.Errorf("expected sound notification not found in log: %s", string(logData))
		}
		if !strings.Contains(string(logData), "Sound: Hero") {
			t.Errorf("expected sound parameter not found in log: %s", string(logData))
		}
		// Verify -sender flag is always present
		if !strings.Contains(string(logData), "Sender: com.ahacop.macos-notify-bridge") {
			t.Errorf("expected sender parameter not found in log: %s", string(logData))
		}
	}
}

// TestIntegrationConcurrentLoad tests the server under concurrent load
func TestIntegrationConcurrentLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build and start server similar to above
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "test-server")

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build binary: %v\nOutput: %s", err, output)
	}

	// Find an available port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Logf("failed to close listener: %v", err)
	}

	// Create mock terminal-notifier
	mockNotifierPath := filepath.Join(tempDir, "terminal-notifier")
	mockNotifierContent := `#!/bin/sh
# Simulate some processing time
sleep 0.01
echo "Mock notification: $*" >> ` + filepath.Join(tempDir, "notifications.log") + `
exit 0
`
	if err := os.WriteFile(mockNotifierPath, []byte(mockNotifierContent), 0755); err != nil {
		t.Fatalf("failed to create mock terminal-notifier: %v", err)
	}

	// Start server
	cmd := exec.Command(binaryPath, "-p", fmt.Sprintf("%d", port))
	cmd.Env = append(os.Environ(), fmt.Sprintf("PATH=%s:%s", tempDir, os.Getenv("PATH")))

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	t.Cleanup(func() {
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Logf("failed to send SIGTERM: %v", err)
		}
		if err := cmd.Wait(); err != nil && !strings.Contains(err.Error(), "signal: terminated") {
			t.Logf("wait error: %v", err)
		}
	})

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Send concurrent requests
	numClients := 50
	numRequestsPerClient := 10
	errors := make(chan error, numClients*numRequestsPerClient)
	done := make(chan bool, numClients)

	for i := 0; i < numClients; i++ {
		go func(clientID int) {
			for j := 0; j < numRequestsPerClient; j++ {
				conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
				if err != nil {
					errors <- fmt.Errorf("client %d request %d: connection failed: %v", clientID, j, err)
					continue
				}

				notification := NotificationRequest{
					Title:   fmt.Sprintf("Client %d", clientID),
					Message: fmt.Sprintf("Request %d", j),
				}
				data, _ := json.Marshal(notification)

				if _, err := conn.Write(append(data, '\n')); err != nil {
					errors <- fmt.Errorf("client %d request %d: write failed: %v", clientID, j, err)
					if cerr := conn.Close(); cerr != nil {
						errors <- fmt.Errorf("client %d request %d: close failed: %v", clientID, j, cerr)
					}
					continue
				}

				response := make([]byte, 1024)
				if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
					errors <- fmt.Errorf("client %d request %d: failed to set deadline: %v", clientID, j, err)
					if cerr := conn.Close(); cerr != nil {
						errors <- fmt.Errorf("client %d request %d: close failed: %v", clientID, j, cerr)
					}
					continue
				}
				n, err := conn.Read(response)
				if err != nil {
					errors <- fmt.Errorf("client %d request %d: read failed: %v", clientID, j, err)
					if cerr := conn.Close(); cerr != nil {
						errors <- fmt.Errorf("client %d request %d: close failed: %v", clientID, j, cerr)
					}
					continue
				}

				if !strings.Contains(string(response[:n]), "OK") {
					errors <- fmt.Errorf("client %d request %d: unexpected response: %s", clientID, j, string(response[:n]))
				}

				if err := conn.Close(); err != nil {
					errors <- fmt.Errorf("client %d request %d: close failed: %v", clientID, j, err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all clients to complete
	for i := 0; i < numClients; i++ {
		<-done
	}
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Logf("Error: %v", err)
		errorCount++
	}

	// Allow some errors but not too many (< 5%)
	maxErrors := (numClients * numRequestsPerClient) / 20
	if errorCount > maxErrors {
		t.Errorf("too many errors: %d (max allowed: %d)", errorCount, maxErrors)
	}
}

// TestIntegrationEnvironmentVariable tests PORT environment variable
func TestIntegrationEnvironmentVariable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "test-server")

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build binary: %v\nOutput: %s", err, output)
	}

	// Find an available port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Logf("failed to close listener: %v", err)
	}

	// Create mock terminal-notifier
	mockNotifierPath := filepath.Join(tempDir, "terminal-notifier")
	mockNotifierContent := `#!/bin/sh
exit 0
`
	if err := os.WriteFile(mockNotifierPath, []byte(mockNotifierContent), 0755); err != nil {
		t.Fatalf("failed to create mock terminal-notifier: %v", err)
	}

	// Start server with PORT environment variable
	cmd := exec.Command(binaryPath, "-v")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PATH=%s:%s", tempDir, os.Getenv("PATH")),
		fmt.Sprintf("PORT=%d", port),
	)

	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	t.Cleanup(func() {
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Logf("failed to send SIGTERM: %v", err)
		}
		if err := cmd.Wait(); err != nil && !strings.Contains(err.Error(), "signal: terminated") {
			t.Logf("wait error: %v", err)
		}
	})

	// Read output to verify port
	outputBytes := make([]byte, 1024)
	n, _ := output.Read(outputBytes)
	outputStr := string(outputBytes[:n])

	expectedMsg := fmt.Sprintf("Server listening on 0.0.0.0:%d", port)
	if !strings.Contains(outputStr, expectedMsg) {
		t.Errorf("expected output to contain %q, got: %s", expectedMsg, outputStr)
	}

	// Verify server is actually listening on the port
	time.Sleep(500 * time.Millisecond)
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("server not listening on expected port: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Logf("failed to close connection: %v", err)
	}
}
