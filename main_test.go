package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ahacop/macos-notify-bridge/internal/testutil"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		port    int
		verbose bool
	}{
		{"default config", "localhost", 9876, false},
		{"verbose mode", "0.0.0.0", 8080, true},
		{"custom host", "127.0.0.1", 9999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(tt.host, tt.port, tt.verbose)
			if server.host != tt.host {
				t.Errorf("expected host %s, got %s", tt.host, server.host)
			}
			if server.port != tt.port {
				t.Errorf("expected port %d, got %d", tt.port, server.port)
			}
			if server.verbose != tt.verbose {
				t.Errorf("expected verbose %v, got %v", tt.verbose, server.verbose)
			}
			if server.shutdown == nil {
				t.Error("shutdown channel not initialized")
			}
		})
	}
}

func TestNotificationRequestJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		valid   bool
		title   string
		message string
		sound   string
	}{
		{
			name:    "valid request",
			input:   `{"title":"Test","message":"Hello"}`,
			valid:   true,
			title:   "Test",
			message: "Hello",
		},
		{
			name:    "missing title",
			input:   `{"message":"Hello"}`,
			valid:   false,
			title:   "",
			message: "Hello",
		},
		{
			name:    "missing message",
			input:   `{"title":"Test"}`,
			valid:   false,
			title:   "Test",
			message: "",
		},
		{
			name:    "empty strings",
			input:   `{"title":"","message":""}`,
			valid:   false,
			title:   "",
			message: "",
		},
		{
			name:  "invalid json",
			input: `invalid json`,
			valid: false,
		},
		{
			name:    "extra fields",
			input:   `{"title":"Test","message":"Hello","extra":"field"}`,
			valid:   true,
			title:   "Test",
			message: "Hello",
		},
		{
			name:    "with sound field",
			input:   `{"title":"Test","message":"Hello","sound":"Hero"}`,
			valid:   true,
			title:   "Test",
			message: "Hello",
			sound:   "Hero",
		},
		{
			name:    "with empty sound field",
			input:   `{"title":"Test","message":"Hello","sound":""}`,
			valid:   true,
			title:   "Test",
			message: "Hello",
			sound:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req NotificationRequest
			err := json.Unmarshal([]byte(tt.input), &req)

			if err != nil && tt.valid {
				t.Errorf("expected valid JSON but got error: %v", err)
			}

			if tt.valid && err == nil {
				if req.Title != tt.title {
					t.Errorf("expected title %q, got %q", tt.title, req.Title)
				}
				if req.Message != tt.message {
					t.Errorf("expected message %q, got %q", tt.message, req.Message)
				}
				if req.Sound != tt.sound {
					t.Errorf("expected sound %q, got %q", tt.sound, req.Sound)
				}

				// Check if it would be considered valid by our logic
				isValid := req.Title != "" && req.Message != ""
				if isValid != tt.valid {
					t.Errorf("expected valid=%v but validation logic returned %v", tt.valid, isValid)
				}
			}
		})
	}
}

func TestIsFlagPassed(t *testing.T) {
	// Save original command line flags
	oldCommandLine := flag.CommandLine
	oldArgs := os.Args

	// Create new flag set for testing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Define test flags
	flag.Int("testport", 9876, "test port")
	flag.String("testhost", "localhost", "test host")
	flag.Bool("testverbose", false, "test verbose")

	// Test with flags passed
	os.Args = []string{"cmd", "-testport=8080", "-testhost=127.0.0.1"}
	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if !isFlagPassed("testport") {
		t.Error("expected testport to be passed")
	}
	if !isFlagPassed("testhost") {
		t.Error("expected testhost to be passed")
	}
	if isFlagPassed("testverbose") {
		t.Error("expected testverbose to NOT be passed")
	}

	// Restore original flags
	flag.CommandLine = oldCommandLine
	os.Args = oldArgs
}

func TestHandleConnectionLogic(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput string
		expectError    bool
	}{
		{
			name:           "valid notification",
			input:          `{"title":"Test","message":"Hello"}`,
			expectedOutput: "OK",
			expectError:    false,
		},
		{
			name:           "invalid json",
			input:          `invalid json`,
			expectedOutput: "ERROR: Invalid JSON",
			expectError:    true,
		},
		{
			name:           "missing title",
			input:          `{"message":"Hello"}`,
			expectedOutput: "ERROR: Missing title or message",
			expectError:    true,
		},
		{
			name:           "missing message",
			input:          `{"title":"Test"}`,
			expectedOutput: "ERROR: Missing title or message",
			expectError:    true,
		},
		{
			name:           "empty title",
			input:          `{"title":"","message":"Hello"}`,
			expectedOutput: "ERROR: Missing title or message",
			expectError:    true,
		},
		{
			name:           "empty message",
			input:          `{"title":"Test","message":""}`,
			expectedOutput: "ERROR: Missing title or message",
			expectError:    true,
		},
		{
			name:           "valid notification with sound",
			input:          `{"title":"Test","message":"Hello","sound":"Hero"}`,
			expectedOutput: "OK",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock connection using net.Pipe
			client, server := net.Pipe()
			t.Cleanup(func() {
				if err := client.Close(); err != nil {
					t.Logf("failed to close client: %v", err)
				}
			})
			t.Cleanup(func() {
				if err := server.Close(); err != nil {
					t.Logf("failed to close server: %v", err)
				}
			})

			// We don't use the server instance here, just testing the logic

			// Write test input to client side
			go func() {
				if _, err := client.Write([]byte(tt.input + "\n")); err != nil {
					t.Errorf("failed to write test input: %v", err)
				}
			}()

			// Process the connection
			go func() {
				// Read and process similar to handleConnection
				// but without actually sending notifications
				data := make([]byte, 1024)
				n, _ := server.Read(data)
				input := strings.TrimSpace(string(data[:n]))

				var req NotificationRequest
				var response string

				if err := json.Unmarshal([]byte(input), &req); err != nil {
					response = "ERROR: Invalid JSON\n"
				} else if req.Title == "" || req.Message == "" {
					response = "ERROR: Missing title or message\n"
				} else {
					// Mock successful notification
					response = "OK\n"
				}

				if _, err := server.Write([]byte(response)); err != nil {
					t.Logf("failed to write response: %v", err)
				}
			}()

			// Read response
			response := make([]byte, 1024)
			n, err := client.Read(response)
			if err != nil {
				t.Fatalf("error reading response: %v", err)
			}

			actualOutput := strings.TrimSpace(string(response[:n]))
			if !strings.Contains(actualOutput, tt.expectedOutput) {
				t.Errorf("expected output to contain %q, got %q", tt.expectedOutput, actualOutput)
			}
		})
	}
}

func TestServerStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
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

	server := NewServer("localhost", port, false)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		err := server.Start()
		if err != nil {
			serverErr <- err
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Check if server is listening
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("server not listening: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Logf("failed to close connection: %v", err)
	}

	// Stop server
	server.Stop()

	// Check for server errors
	select {
	case err := <-serverErr:
		t.Fatalf("server error: %v", err)
	case <-time.After(time.Second):
		// No error, server stopped gracefully
	}

	// Verify server is no longer listening
	_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err == nil {
		t.Error("server still listening after stop")
	}
}

func TestConcurrentConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Create temporary directory for mock
	tempDir := t.TempDir()

	// Create mock terminal-notifier
	_, err := testutil.CreateMockTerminalNotifier(tempDir)
	if err != nil {
		t.Fatalf("failed to create mock terminal-notifier: %v", err)
	}

	// Set PATH to use our mock
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", tempDir+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Setenv("PATH", oldPath); err != nil {
			t.Logf("failed to restore PATH: %v", err)
		}
	})

	// Find an available port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Logf("failed to close listener: %v", err)
	}

	server := NewServer("localhost", port, false)

	// Start server
	go func() {
		if err := server.Start(); err != nil {
			t.Logf("server start error: %v", err)
		}
	}()
	t.Cleanup(func() {
		server.Stop()
	})

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Send multiple concurrent connections
	numConnections := 10
	done := make(chan bool, numConnections)

	for i := 0; i < numConnections; i++ {
		go func(id int) {
			conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
			if err != nil {
				t.Errorf("connection %d failed: %v", id, err)
				done <- false
				return
			}
			t.Cleanup(func() {
				if err := conn.Close(); err != nil {
					t.Logf("connection %d: failed to close: %v", id, err)
				}
			})

			// Send notification
			notification := fmt.Sprintf(`{"title":"Test %d","message":"Message %d"}`, id, id)
			_, err = conn.Write([]byte(notification + "\n"))
			if err != nil {
				t.Errorf("connection %d write failed: %v", id, err)
				done <- false
				return
			}

			// Read response
			response := make([]byte, 1024)
			if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
				t.Errorf("connection %d failed to set deadline: %v", id, err)
				done <- false
				return
			}
			n, err := conn.Read(response)
			if err != nil {
				t.Errorf("connection %d read failed: %v", id, err)
				done <- false
				return
			}

			if !strings.Contains(string(response[:n]), "OK") {
				t.Errorf("connection %d got unexpected response: %s", id, string(response[:n]))
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all connections to complete
	successCount := 0
	for i := 0; i < numConnections; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount != numConnections {
		t.Errorf("only %d/%d connections succeeded", successCount, numConnections)
	}
}
