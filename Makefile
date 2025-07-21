# Project variables
BINARY_NAME := macos-notify-bridge
GO_FILES := $(shell find . -name '*.go' -not -path "./vendor/*")
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOIMPORTS := goimports

# Golangci-lint
GOLANGCI_LINT := golangci-lint
GOLANGCI_VERSION := v1.61.0

.PHONY: all build test test-unit test-integration test-coverage lint fmt fmt-check clean install help

# Default target
all: fmt lint test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -short -v ./...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -run Integration -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out -covermode=atomic ./...
	@echo "Coverage report:"
	@$(GOCMD) tool cover -func=coverage.out
	@echo "\nTo view HTML coverage report, run: go tool cover -html=coverage.out"

# Run linter
lint:
	@echo "Running linter..."
	@if ! command -v $(GOLANGCI_LINT) > /dev/null; then \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERSION); \
	fi
	$(GOLANGCI_LINT) run

# Format code
fmt:
	@echo "Formatting code..."
	@if ! command -v goimports > /dev/null; then \
		echo "goimports not found. Installing..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	$(GOFMT) -s -w $(GO_FILES)
	$(GOIMPORTS) -w $(GO_FILES)

# Check if code is formatted
fmt-check:
	@echo "Checking code formatting..."
	@if [ -n "$$($(GOFMT) -l $(GO_FILES))" ]; then \
		echo "The following files need formatting:"; \
		$(GOFMT) -l $(GO_FILES); \
		echo "\nRun 'make fmt' to format them."; \
		exit 1; \
	fi
	@echo "All files are properly formatted."

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out
	@rm -f coverage.html
	@$(GOCMD) clean -testcache

# Install the binary
install: build
	@echo "Installing $(BINARY_NAME)..."
	@mkdir -p $(HOME)/bin
	@cp $(BINARY_NAME) $(HOME)/bin/
	@echo "Installed to $(HOME)/bin/$(BINARY_NAME)"

# Test goreleaser without publishing
release-dry:
	@echo "Running goreleaser in dry-run mode..."
	@if ! command -v goreleaser > /dev/null; then \
		echo "goreleaser not found. Please install it first."; \
		exit 1; \
	fi
	goreleaser release --snapshot --clean

# Help
help:
	@echo "Available targets:"
	@echo "  all             - Run fmt, lint, test, and build"
	@echo "  build           - Build the binary"
	@echo "  test            - Run all tests"
	@echo "  test-unit       - Run unit tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  lint            - Run golangci-lint"
	@echo "  fmt             - Format code with gofmt and goimports"
	@echo "  fmt-check       - Check if code is properly formatted"
	@echo "  clean           - Remove build artifacts"
	@echo "  install         - Install binary to ~/bin"
	@echo "  release-dry     - Test goreleaser without publishing"
	@echo "  help            - Show this help message"
