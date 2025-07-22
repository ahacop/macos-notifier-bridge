package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const version = "0.1.0"

// NotificationRequest represents a notification request from a client.
type NotificationRequest struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Sound   string `json:"sound,omitempty"`
}

// Server represents the notification bridge server.
type Server struct {
	host     string
	port     int
	verbose  bool
	listener net.Listener
	wg       sync.WaitGroup
	shutdown chan struct{}
}

// NewServer creates a new notification bridge server instance.
func NewServer(host string, port int, verbose bool) *Server {
	return &Server{
		host:     host,
		port:     port,
		verbose:  verbose,
		shutdown: make(chan struct{}),
	}
}

// Start starts the server and begins listening for connections.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	log.Printf("Server listening on %s", addr)

	go s.acceptConnections()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")
	s.Stop()
	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() {
	close(s.shutdown)
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			if s.verbose {
				log.Printf("Error closing listener: %v", err)
			}
		}
	}
	s.wg.Wait()
	log.Println("Server stopped")
}

func (s *Server) acceptConnections() {
	for {
		select {
		case <-s.shutdown:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.shutdown:
					return
				default:
					if s.verbose {
						log.Printf("Error accepting connection: %v", err)
					}
					continue
				}
			}

			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer func() {
		if err := conn.Close(); err != nil && s.verbose {
			log.Printf("Error closing connection: %v", err)
		}
	}()

	if s.verbose {
		log.Printf("New connection from %s", conn.RemoteAddr())
	}

	// Set read timeout
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		if s.verbose {
			log.Printf("Error setting read deadline: %v", err)
		}
		// Continue anyway, connection might still work
	}

	reader := bufio.NewReader(conn)
	data, err := reader.ReadString('\n')
	if err != nil {
		if s.verbose {
			log.Printf("Error reading from connection: %v", err)
		}
		if _, err := conn.Write([]byte("ERROR: Failed to read request\n")); err != nil {
			if s.verbose {
				log.Printf("Error writing error response: %v", err)
			}
		}
		return
	}

	data = strings.TrimSpace(data)
	if s.verbose {
		log.Printf("Received: %s", data)
	}

	var req NotificationRequest
	if err := json.Unmarshal([]byte(data), &req); err != nil {
		if s.verbose {
			log.Printf("Error parsing JSON: %v", err)
		}
		if _, err := conn.Write([]byte("ERROR: Invalid JSON\n")); err != nil {
			if s.verbose {
				log.Printf("Error writing error response: %v", err)
			}
		}
		return
	}

	if req.Title == "" || req.Message == "" {
		if _, err := conn.Write([]byte("ERROR: Missing title or message\n")); err != nil {
			if s.verbose {
				log.Printf("Error writing error response: %v", err)
			}
		}
		return
	}

	if err := s.sendNotification(req.Title, req.Message, req.Sound); err != nil {
		if s.verbose {
			log.Printf("Error sending notification: %v", err)
		}
		if _, err := fmt.Fprintf(conn, "ERROR: %v\n", err); err != nil {
			if s.verbose {
				log.Printf("Error writing error response: %v", err)
			}
		}
		return
	}

	if _, err := conn.Write([]byte("OK\n")); err != nil {
		if s.verbose {
			log.Printf("Error writing OK response: %v", err)
		}
	}
}

func (s *Server) sendNotification(title, message, sound string) error {
	args := []string{
		"-title", title,
		"-message", message,
		"-sender", "com.ahacop.macos-notify-bridge",
	}
	if sound != "" {
		args = append(args, "-sound", sound)
	}

	cmd := exec.Command("terminal-notifier", args...)

	if s.verbose {
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("terminal-notifier failed: %w, output: %s", err, string(output))
		}
		log.Printf("Notification sent: %s - %s (sound: %s)", title, message, sound)
	} else {
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("terminal-notifier failed: %w", err)
		}
	}

	return nil
}

func main() {
	var (
		port        = flag.Int("port", 9876, "Port to listen on")
		portP       = flag.Int("p", 9876, "Port to listen on (short)")
		host        = flag.String("host", "0.0.0.0", "Host to bind to")
		hostH       = flag.String("h", "0.0.0.0", "Host to bind to (short)")
		verbose     = flag.Bool("verbose", false, "Enable verbose logging")
		verboseV    = flag.Bool("v", false, "Enable verbose logging (short)")
		showVersion = flag.Bool("version", false, "Show version")
	)

	flag.Parse()

	if *showVersion {
		fmt.Printf("macos-notify-bridge version %s\n", version)
		os.Exit(0)
	}

	// Use short flags if they were explicitly set
	if isFlagPassed("p") {
		*port = *portP
	}
	if isFlagPassed("h") {
		*host = *hostH
	}
	if isFlagPassed("v") {
		*verbose = *verboseV
	}

	// Check if terminal-notifier is available
	if _, err := exec.LookPath("terminal-notifier"); err != nil {
		log.Fatal("terminal-notifier not found. Please install it: brew install terminal-notifier")
	}

	// Check for PORT environment variable
	if envPort := os.Getenv("PORT"); envPort != "" && !isFlagPassed("port") && !isFlagPassed("p") {
		var p int
		if _, err := fmt.Sscanf(envPort, "%d", &p); err == nil {
			*port = p
		}
	}

	server := NewServer(*host, *port, *verbose)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
