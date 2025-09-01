.PHONY: build dev docs clean install install-air install-deps help update-blender-init release-macos

# Variables
BINARY_NAME=mix
BUILD_DIR=go_backend/build
MAIN_PATH=./go_backend/main.go

# Build optimization variables
VERSION=$(shell git tag --sort=committerdate | grep -E '[0-9]' | tail -1 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
TARGET_TRIPLE=$(shell rustc -Vv | grep host | cut -f2 -d' ')

# Common build flags for optimized binaries
BUILD_FLAGS=-a
LDFLAGS=-s -w -X mix/internal/version.Version=$(VERSION) -X main.buildTime=$(BUILD_TIME)
CGO_ENV=CGO_ENABLED=0

# Default target
help:
	@echo "Available targets:"
	@echo "  dev         - Install dependencies and run development servers"
	@echo "  docs        - Run documentation development server"
	@echo "  install     - Install system dependencies (one-time setup)"
	@echo "  install-deps - Install project dependencies"
	@echo "  build       - Build the binary to $(BUILD_DIR)/release/ directory"
	@echo "  build-sidecar - Build Tauri-compatible sidecar binary with platform suffix"
	@echo "  clean       - Clean build artifacts"
	@echo "  install-air - Install Air if not present"
	@echo "  tail-log    - Show the last 100 lines of the log"
	@echo "  test-env    - Validate environment variables and configuration"
	@echo "  test-connection - Test connection between frontend and backend"
	@echo "  test-installation - Test if all dependencies are installed"
	@echo "  test-all    - Run all validation tests"
	@echo "  help        - Show this help message"
	@echo ""


# Run development server with hot reloading (installs deps first)
dev: install-deps 
	@ENV=development ./scripts/shoreman.sh

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
	cd go_backend && go mod download
	@echo "Installing capture script dependencies..."
	cd go_backend && bun install
	@echo "Installing Tauri app dependencies..."
	cd tauri_app && bun i
	@echo "✅ All dependencies installed!"

# Internal target for optimized builds
# Usage: make _build-optimized OUTPUT_PATH=path/to/binary [GOOS=os] [GOARCH=arch]
_build-optimized:
	@echo "Building optimized binary..."
	@echo "Version: $(VERSION)"
	@echo "Build time: $(BUILD_TIME)"
	@mkdir -p $(dir $(OUTPUT_PATH))
	cd go_backend && \
	$(CGO_ENV) $(if $(GOOS),GOOS=$(GOOS)) $(if $(GOARCH),GOARCH=$(GOARCH)) go build \
		$(BUILD_FLAGS) \
		-ldflags="$(LDFLAGS)" \
		-o $(OUTPUT_PATH) \
		main.go

# Build Tauri-compatible sidecar binary with platform-specific naming
build-sidecar:
	@echo "Building optimized Tauri sidecar binary for platform: $(TARGET_TRIPLE)"
	@$(MAKE) _build-optimized OUTPUT_PATH=build/release/$(BINARY_NAME)-$(TARGET_TRIPLE)
	@echo "Tauri sidecar binary built: $(BUILD_DIR)/release/$(BINARY_NAME)-$(TARGET_TRIPLE)"

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

test-all: test-env test-dev-env test-connection test-installation
	@echo "All validation tests completed."