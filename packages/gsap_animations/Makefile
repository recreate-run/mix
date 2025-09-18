.PHONY: run build clean test test-server test-all help

# Variables
BINARY_NAME=gsap-server
BUILD_DIR=build
PORT?=8089

# Default target
help:
	@echo "Available targets:"
	@echo "  run         - Run the server directly (default port: $(PORT))"
	@echo "  dev         - Run with Air for hot reloading (default port: $(PORT))"
	@echo "  build       - Build the server binary to $(BUILD_DIR)/"
	@echo "  test        - Run integration tests"
	@echo "  test-server - Run integration tests with verbose output"
	@echo "  test-all    - Run all tests recursively in the entire package"
	@echo "  clean       - Remove build directory"
	@echo "  help        - Show this help message"

# Run the server directly
run:
	@echo "Starting GSAP animations server on port $(PORT)..."
	@echo "Serving animations from: $$(pwd)"
	@echo "Server will be available at: http://localhost:$(PORT)"
	@GSAP_PORT=$(PORT) go run server.go

# Run with Air for hot reloading
dev:
	@command -v air >/dev/null 2>&1 || (echo "Installing Air..." && go install github.com/air-verse/air@latest)
	@echo "Starting GSAP server with hot reloading on port $(PORT)..."
	@GSAP_PORT=$(PORT) air

# Build the server binary
build:
	@echo "Building GSAP server binary..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) server.go
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

# Run integration tests
test:
	@echo "Running integration tests..."
	@go test -timeout 30s

# Run integration tests with verbose output
test-server:
	@echo "Running integration tests with verbose output..."
	@go test -timeout 30s -v

# Run all tests recursively in the entire package
test-all:
	@echo "Running all tests in packages/gsap_animations/..."
	@go test ./... -timeout 30s -v

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) tmp
	@echo "Clean complete"