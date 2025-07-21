package testutil

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// NotificationRequest represents a notification request
type NotificationRequest struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

// SendNotification sends a notification to the server and returns the response
func SendNotification(host string, port int, title, message string) (string, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			// Log error but don't fail - connection is already closed
			_ = err
		}
	}()

	// Set timeouts
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return "", fmt.Errorf("failed to set deadline: %w", err)
	}

	// Prepare notification
	req := NotificationRequest{
		Title:   title,
		Message: message,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send notification
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return "", fmt.Errorf("failed to send notification: %w", err)
	}

	// Read response
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(response[:n]), nil
}

// SendRawData sends raw data to the server and returns the response
func SendRawData(host string, port int, data string) (string, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			// Log error but don't fail - connection is already closed
			_ = err
		}
	}()

	// Set timeouts
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return "", fmt.Errorf("failed to set deadline: %w", err)
	}

	// Send data
	if _, err := conn.Write([]byte(data)); err != nil {
		return "", fmt.Errorf("failed to send data: %w", err)
	}

	// Read response
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(response[:n]), nil
}

// FindAvailablePort finds an available TCP port
func FindAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, fmt.Errorf("failed to find available port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		return 0, fmt.Errorf("failed to close listener: %w", err)
	}
	return port, nil
}

// WaitForServer waits for the server to be ready
func WaitForServer(host string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
		if err == nil {
			if err := conn.Close(); err != nil {
				// Ignore close error, we're just checking if server is ready
				_ = err
			}
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("server not ready after %v", timeout)
}
