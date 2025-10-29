.PHONY: build dev frontend-only docs clean install install-air install-deps help update-blender-init release-macos go-lint go-test generate-mix_sdk gsap-server

# Variables
BINARY_NAME=mix
BUILD_DIR=mix_agent/build
MAIN_PATH=./mix_agent/main.go

# Build optimization variables
VERSION=$(shell git tag --sort=committerdate | grep -E '[0-9]' | tail -1 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
TARGET_TRIPLE=$(shell rustc -Vv | grep host | cut -f2 -d' ')

# Common build flags for optimized binaries
BUILD_FLAGS=-a
LDFLAGS=-s -w -X mix/internal/version.Version=$(VERSION) -X main.buildTime=$(BUILD_TIME)
CGO_ENV=CGO_ENABLED=0

# SDK generation variables
SDK_OUTPUT_DIR=../mix-typescript-mix_sdk
OPENAPI_ENDPOINT=http://localhost:8088/doc

# Default target
help:
	@echo "Available targets:"
	@echo "  dev         - Install dependencies and run all development servers (backend, web frontend, GSAP)"
	@echo "  dev-kill    - Stop all development servers started by 'make dev'"
	@echo "  frontend-only - Run only the web frontend development server (no backend)"
	@echo "  docs        - Run documentation development server"
	@echo "  install     - Install system dependencies (one-time setup)"
	@echo "  install-deps - Install project dependencies"
	@echo "  build       - Build optimized binary for current platform"
	@echo "  build-all   - Build binaries for all platforms and architectures"
	@echo "  build-macos - Build for all macOS architectures (Intel + Apple Silicon)"
	@echo "  build-linux - Build for all Linux architectures (x64 + ARM64)"
	@echo "  build-windows - Build for Windows (x64)"
	@echo "  build-darwin-amd64  - Build for macOS (Intel)"
	@echo "  build-darwin-arm64  - Build for macOS (Apple Silicon)"
	@echo "  build-linux-amd64   - Build for Linux (x64)"
	@echo "  build-linux-arm64   - Build for Linux (ARM64)"
	@echo "  build-windows-amd64 - Build for Windows (x64)"
	@echo "  build-mac-intel     - Alias for build-darwin-amd64"
	@echo "  build-mac-arm       - Alias for build-darwin-arm64"
	@echo "  release     - Create release with GoReleaser"
	@echo "  release-test - Test release build (dry run)"
	@echo "  release-snapshot - Create snapshot release for testing"
	@echo "  scripts/release.sh - Enhanced release script with validation"
	@echo "  clean       - Clean build artifacts"
	@echo "  install-air - Install Air if not present"
	@echo "  tail-log    - Show the last 100 lines of the log"
	@echo "  test-env    - Validate environment variables and configuration"
	@echo "  test-connection - Test connection between frontend and backend (requires running servers)"
	@echo "  test-installation - Test if all dependencies are installed"
	@echo "  test-all    - Run all environment validation tests (no running servers required)"
	@echo "  frontend-typecheck - Run TypeScript typecheck on frontend code"
	@echo "  lint        - Run linters for both Go backend and frontend"
	@echo "  frontend-format - Run knip linter on frontend code"
	@echo "  frontend-lint - Run knip linter on frontend code"
	@echo "  frontend-knip - Run biome linter on frontend code"
	@echo "  go-lint     - Run golangci-lint on Go backend code"
	@echo "  go-test     - Run Go tests with coverage"
	@echo "  sqlc-generate - Regenerate database query code with sqlc"
	@echo "  generate-openapi - Generate JSON OpenAPI spec"
	@echo "  help        - Show this help message"
	@echo ""


# Run development server with hot reloading (installs deps first)
# This starts backend (Go), web frontend (browser), and GSAP server together
dev: install-deps
	@ENV=development ./scripts/shoreman.sh

# Stop all development servers
dev-kill:
	@echo "Stopping all development servers..."
	@if [ -f .shoreman.pid ]; then \
		pid=$$(cat .shoreman.pid); \
		if kill -0 $$pid 2>/dev/null; then \
			echo "Sending SIGTERM to shoreman (PID $$pid)..."; \
			kill $$pid && echo "✅ Development servers stopped"; \
		else \
			echo "PID file exists but process not running, cleaning up..."; \
			rm -f .shoreman.pid; \
		fi \
	else \
		echo "No PID file found. Attempting to kill processes manually..."; \
		pkill -f "air" || true; \
		pkill -f "bun run tauri dev" || true; \
		pkill -f "mix --http-port" || true; \
		echo "✅ Killed stray processes (if any)"; \
	fi

# Run only frontend development server (no backend)
frontend-only: install-deps
	@echo "Starting frontend-only development server..."
	@echo "Frontend will be available at http://localhost:1420"
	@if [ -f .env ]; then \
		export $$(grep -v '^#' .env | grep '^VITE_' | xargs) && cd mix_dev_tool && bun run dev; \
	else \
		echo "Warning: .env file not found, using defaults" && cd mix_dev_tool && bun run dev; \
	fi

# Run documentation development server
docs:
	cd docs && bun run dev

# Install system dependencies (one-time setup)
install:
	@./scripts/install_deps.sh

# Install project dependencies
install-deps: install
	@echo "Installing project dependencies..."
	@echo "Installing Air (Go hot reload)..."
	@command -v air >/dev/null 2>&1 || go install github.com/air-verse/air@latest
	@echo "Installing Go dependencies..."
	cd mix_agent && go mod download
	@echo "Installing capture script dependencies..."
	# cd mix_agent && bun install
	@echo "Installing Tauri app dependencies..."
	cd mix_dev_tool && bun i
	@echo "Installing GSAP animations dependencies..."
	cd packages/gsap_animations && bun install
	@echo "✅ All dependencies installed!"

# Internal target for optimized builds
# Usage: make _build-optimized OUTPUT_PATH=path/to/binary [GOOS=os] [GOARCH=arch]
_build-optimized:
	@echo "Building optimized binary..."
	@echo "Version: $(VERSION)"
	@echo "Build time: $(BUILD_TIME)"
	@mkdir -p $(dir $(OUTPUT_PATH))
	cd mix_agent && \
	$(CGO_ENV) $(if $(GOOS),GOOS=$(GOOS)) $(if $(GOARCH),GOARCH=$(GOARCH)) go build \
		$(BUILD_FLAGS) \
		-ldflags="$(LDFLAGS)" \
		-o ../$(OUTPUT_PATH) \
		main.go

# Build production binary for current platform
build:
	@echo "Building optimized binary for current platform..."
	@$(MAKE) _build-optimized OUTPUT_PATH=$(BUILD_DIR)/release/$(BINARY_NAME)
	@echo "Binary built: $(BUILD_DIR)/release/$(BINARY_NAME)"

# Cross-platform build targets
build-all: build-macos build-linux build-windows
	@echo "✅ All platform binaries built successfully!"

# Platform-specific build groups
build-macos: build-darwin-amd64 build-darwin-arm64
	@echo "✅ All macOS binaries built successfully!"

build-linux: build-linux-amd64 build-linux-arm64
	@echo "✅ All Linux binaries built successfully!"

build-windows: build-windows-amd64
	@echo "✅ All Windows binaries built successfully!"

# macOS build targets
build-darwin-amd64:
	@echo "Building for macOS (Intel)..."
	@$(MAKE) _build-optimized OUTPUT_PATH=$(BUILD_DIR)/release/$(BINARY_NAME)-mac-intel GOOS=darwin GOARCH=amd64

build-darwin-arm64:
	@echo "Building for macOS (Apple Silicon)..."
	@$(MAKE) _build-optimized OUTPUT_PATH=$(BUILD_DIR)/release/$(BINARY_NAME)-mac-apple-silicon GOOS=darwin GOARCH=arm64

# Linux build targets
build-linux-amd64:
	@echo "Building for Linux (x64)..."
	@$(MAKE) _build-optimized OUTPUT_PATH=$(BUILD_DIR)/release/$(BINARY_NAME)-linux-x64 GOOS=linux GOARCH=amd64

build-linux-arm64:
	@echo "Building for Linux (ARM64)..."
	@$(MAKE) _build-optimized OUTPUT_PATH=$(BUILD_DIR)/release/$(BINARY_NAME)-linux-arm64 GOOS=linux GOARCH=arm64

# Windows build targets
build-windows-amd64:
	@echo "Building for Windows (x64)..."
	@$(MAKE) _build-optimized OUTPUT_PATH=$(BUILD_DIR)/release/$(BINARY_NAME)-windows-x64.exe GOOS=windows GOARCH=amd64

# Alias for convenience
build-mac-intel: build-darwin-amd64
build-mac-arm: build-darwin-arm64

# Release using GoReleaser
release:
	@echo "Creating release with GoReleaser..."
	@command -v goreleaser >/dev/null 2>&1 || { echo "GoReleaser not found. Install with: brew install goreleaser"; exit 1; }
	@goreleaser release --clean

# Test release build (dry run)
release-test:
	@echo "Testing release build with GoReleaser..."
	@command -v goreleaser >/dev/null 2>&1 || { echo "GoReleaser not found. Install with: brew install goreleaser"; exit 1; }
	@goreleaser build --snapshot --clean

# Release snapshot (for testing)
release-snapshot:
	@echo "Creating snapshot release..."
	@command -v goreleaser >/dev/null 2>&1 || { echo "GoReleaser not found. Install with: brew install goreleaser"; exit 1; }
	@goreleaser release --snapshot --clean

# Display the last 100 lines of development log with ANSI codes stripped
tail-log:
	@tail -100 ./dev.log | perl -pe 's/\e\[[0-9;]*m(?:\e\[K)?//g'

# Validation targets
test-env:
	@./scripts/tests/validate_env.sh

test-connection:
	@./scripts/tests/test_connection.sh

test-dev-env:
	@./scripts/tests/validate_dev_env.sh

test-installation:
	@./scripts/test_installation.sh

test-all: test-env test-dev-env test-installation
	@echo "✅ All validation tests completed."

# Run Go tests with coverage
go-test:
	@echo "Running Go tests..."
	@./scripts/tests/run_go_tests.sh

# Run TypeScript typecheck on frontend code
frontend-typecheck:
	@echo "Running frontend TypeScript typecheck..."
	cd mix_dev_tool && bun run typecheck

frontend-format:
	@echo "Running biome formatter on frontend..."
	cd mix_dev_tool && bunx biome format --write

go-format:
	@echo "Formatting Go code..."
	cd mix_agent && gofmt -w .
	@echo "✅ Code formatted successfully!"

frontend-lint:
	@echo "Running biome linter on frontend..."
	cd mix_dev_tool && bunx biome check --write

frontend-knip:
	@echo "Running knip linter on frontend..."
	cd mix_dev_tool && bun knip

# Run linters for both Go backend and frontend
lint: go-lint frontend-lint
	@echo "✅ All linting completed successfully!"

format: go-format frontend-format
	@echo "✅ All formatting completed successfully!"

# Run golangci-lint on Go backend code
go-lint:
	@echo "Running golangci-lint on Go backend code..."
	cd mix_agent && golangci-lint run ./...

# Regenerate database query code with sqlc
sqlc-generate:
	@echo "Installing sqlc (v1.29.0)..."
	@cd mix_agent && go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0
	@echo "Regenerating database query code..."
	@cd mix_agent && sqlc generate
	@echo "✅ Database query code regenerated successfully!"

# Generate TypeScript SDK from OpenAPI specification
generate-openapi:
	@echo "Generating TypeScript SDK from OpenAPI spec..."
	@echo "Using configuration from mix_sdk/gen.yaml"
	@echo "Downloading OpenAPI spec from $(OPENAPI_ENDPOINT)..."
	@curl -s $(OPENAPI_ENDPOINT) > mix_sdk/openapi-spec.json
	@echo "Saving pretty-printed OpenAPI document to mix_sdk/openapi.json..."
	@curl -s $(OPENAPI_ENDPOINT) | jq '.' > mix_sdk/openapi.json
# 	@echo "Running Speakeasy SDK generation..."
# 	@speakeasy generate mix_sdk --schema mix_sdk/openapi-spec.json --lang typescript --out $(SDK_OUTPUT_DIR)
# 	@echo "Installing SDK dependencies with bun..."
# 	@cd $(SDK_OUTPUT_DIR) && bun install
# 	@echo "Building TypeScript SDK..."
# 	@cd $(SDK_OUTPUT_DIR) && bun run build
# 	@echo "✅ TypeScript SDK generated and built successfully at $(SDK_OUTPUT_DIR)"
# 	@echo "📖 See mix_sdk/README.md for usage instructions"
# 	@echo "📄 OpenAPI document saved at mix_sdk/openapi.json"
# 	@rm -f mix_sdk/openapi-spec.json

# Clean all build artifacts and dependencies
clean:
	@echo "🧹 Cleaning build artifacts and dependencies..."
	@echo "Cleaning Go backend (mix_agent)..."
	@rm -rf mix_agent/build || true
	@rm -f mix_agent/mix || true
	@cd mix_agent && go clean || true
	@echo "Cleaning Tauri frontend (mix_dev_tool)..."
	@rm -rf mix_dev_tool/node_modules || true
	@rm -rf mix_dev_tool/src-tauri/target || true
	@rm -rf mix_dev_tool/dist || true
	@echo "Cleaning GSAP animations (packages/gsap_animations)..."
	@rm -rf packages/gsap_animations/build || true
	@rm -rf packages/gsap_animations/tmp || true
	@rm -rf packages/gsap_animations/node_modules || true
	@echo "✅ All build artifacts cleaned!"