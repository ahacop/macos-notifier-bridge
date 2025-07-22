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

	"github.com/ahacop/macos-notify-bridge/internal/netutil"
)

const version = "0.1.0"

const (
	maxTitleLength   = 256
	maxMessageLength = 1024
	maxSoundLength   = 64
)

// arrayFlags allows multiple values for a flag
type arrayFlags []string

func (a *arrayFlags) String() string {
	return strings.Join(*a, ",")
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

// NotificationRequest represents a notification request from a client.
type NotificationRequest struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Sound   string `json:"sound,omitempty"`
}

// Server represents the notification bridge server.
type Server struct {
	bindAddresses     []string
	port              int
	verbose           bool
	autoDetectBridges bool
	listeners         []net.Listener
	wg                sync.WaitGroup
	shutdown          chan struct{}
	listenerErrors    chan error
}

// NewServer creates a new notification bridge server instance.
func NewServer(port int, verbose bool, bindAddresses []string, autoDetectBridges bool) *Server {
	return &Server{
		bindAddresses:     bindAddresses,
		port:              port,
		verbose:           verbose,
		autoDetectBridges: autoDetectBridges,
		shutdown:          make(chan struct{}),
		listenerErrors:    make(chan error, 10),
	}
}

// Start starts the server and begins listening for connections.
func (s *Server) Start() error {
	// Get all bind addresses
	addresses, err := s.getBindAddresses()
	if err != nil {
		return fmt.Errorf("failed to get bind addresses: %w", err)
	}

	if len(addresses) == 0 {
		return fmt.Errorf("no bind addresses available")
	}

	// Create listeners for each address
	for _, bindAddr := range addresses {
		addr := fmt.Sprintf("%s:%d", bindAddr, s.port)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			// Log error but continue with other addresses
			log.Printf("Failed to listen on %s: %v", addr, err)
			continue
		}
		s.listeners = append(s.listeners, listener)
		log.Printf("Server listening on %s", addr)

		// Start accepting connections on this listener
		go s.acceptConnections(listener)
	}

	if len(s.listeners) == 0 {
		return fmt.Errorf("failed to create any listeners")
	}

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
	for _, listener := range s.listeners {
		if err := listener.Close(); err != nil {
			if s.verbose {
				log.Printf("Error closing listener: %v", err)
			}
		}
	}
	s.wg.Wait()
	log.Println("Server stopped")
}

// getBindAddresses returns the list of addresses to bind to
func (s *Server) getBindAddresses() ([]string, error) {
	if len(s.bindAddresses) > 0 {
		// Use explicitly provided addresses
		return s.bindAddresses, nil
	}

	// Use auto-detection or default
	return netutil.GetAllBindAddresses(s.autoDetectBridges)
}

func (s *Server) acceptConnections(listener net.Listener) {
	for {
		select {
		case <-s.shutdown:
			return
		default:
			conn, err := listener.Accept()
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

	// Validate input lengths
	if len(req.Title) > maxTitleLength {
		if _, err := fmt.Fprintf(conn, "ERROR: Title too long (max %d characters)\n", maxTitleLength); err != nil {
			if s.verbose {
				log.Printf("Error writing error response: %v", err)
			}
		}
		return
	}
	if len(req.Message) > maxMessageLength {
		if _, err := fmt.Fprintf(conn, "ERROR: Message too long (max %d characters)\n", maxMessageLength); err != nil {
			if s.verbose {
				log.Printf("Error writing error response: %v", err)
			}
		}
		return
	}
	if len(req.Sound) > maxSoundLength {
		if _, err := fmt.Fprintf(conn, "ERROR: Sound name too long (max %d characters)\n", maxSoundLength); err != nil {
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
		port              = flag.Int("port", 9876, "Port to listen on")
		portP             = flag.Int("p", 9876, "Port to listen on (short)")
		verbose           = flag.Bool("verbose", false, "Enable verbose logging")
		verboseV          = flag.Bool("v", false, "Enable verbose logging (short)")
		showVersion       = flag.Bool("version", false, "Show version")
		autoDetectBridges = flag.Bool("auto-detect-bridges", false, "Automatically detect and bind to VM bridge interfaces")
		autoDetectA       = flag.Bool("a", false, "Automatically detect and bind to VM bridge interfaces (short)")
	)

	// Custom flag for multiple bind addresses
	var bindAddresses arrayFlags
	flag.Var(&bindAddresses, "bind", "Bind address (can be specified multiple times)")
	flag.Var(&bindAddresses, "b", "Bind address (can be specified multiple times, short)")

	flag.Parse()

	if *showVersion {
		fmt.Printf("macos-notify-bridge version %s\n", version)
		os.Exit(0)
	}

	// Use short flags if they were explicitly set
	if isFlagPassed("p") {
		*port = *portP
	}
	if isFlagPassed("v") {
		*verbose = *verboseV
	}
	if isFlagPassed("a") {
		*autoDetectBridges = *autoDetectA
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

	// Merge bind addresses from both flags
	var allBindAddresses []string
	for _, addr := range bindAddresses {
		allBindAddresses = append(allBindAddresses, addr)
	}

	// Handle short bind flag
	var bindAddressesShort arrayFlags
	flag.VisitAll(func(f *flag.Flag) {
		if f.Name == "b" && f.Value != nil {
			if av, ok := f.Value.(*arrayFlags); ok {
				bindAddressesShort = *av
			}
		}
	})
	for _, addr := range bindAddressesShort {
		allBindAddresses = append(allBindAddresses, addr)
	}

	server := NewServer(*port, *verbose, allBindAddresses, *autoDetectBridges)
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
