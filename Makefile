.PHONY: build dev clean install-air install-deps help update-blender-init release-macos

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
	@echo "  install-deps - Install all project dependencies"
	@echo "  build       - Build the binary to $(BUILD_DIR)/debug/ directory"
	@echo "  build-sidecar - Build Tauri-compatible sidecar binary with platform suffix"
	@echo "  build-all   - Build both regular and Tauri-compatible binaries"
	@echo "  release-macos - Build optimized Apple Silicon release binary"
	@echo "  clean       - Clean build artifacts"
	@echo "  install-air - Install Air if not present"
	@echo "  tail-log    - Show the last 100 lines of the log"
	@echo "  help        - Show this help message"
	@echo ""
	@echo "System Requirements:"
	@echo "  - Go (https://golang.org/)"
	@echo "  - Rust/Cargo (https://rustup.rs/)"
	@echo "  - Bun (https://bun.sh/)"

# Install all project dependencies
install-deps:
	@echo "Checking system dependencies..."
	@command -v cargo >/dev/null 2>&1 || { echo "❌ Rust/Cargo not found. Please install Rust: https://rustup.rs/"; exit 1; }
	@command -v bun >/dev/null 2>&1 || { echo "❌ Bun not found. Please install Bun: https://bun.sh/"; exit 1; }
	@command -v go >/dev/null 2>&1 || { echo "❌ Go not found. Please install Go: https://golang.org/"; exit 1; }
	@echo "✅ System dependencies verified"
	@echo "Installing Air (Go hot reload)..."
	@command -v air >/dev/null 2>&1 || go install github.com/air-verse/air@latest
	@echo "Installing Go dependencies..."
	cd go_backend && go mod download
	@echo "Installing Tauri app dependencies..."
	cd tauri_app && bun install
	@echo "Installing remotion template dependencies..."
	cd packages/remotion_starter_template && bun install
	@echo "✅ All dependencies installed!"

# Build binary to build directory
build:
	@mkdir -p $(BUILD_DIR)/debug
	cd go_backend && go build -o build/debug/$(BINARY_NAME) main.go
	@echo "Binary built: $(BUILD_DIR)/debug/$(BINARY_NAME)"

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
	@$(MAKE) _build-optimized OUTPUT_PATH=build/debug/$(BINARY_NAME)-$(TARGET_TRIPLE)
	@echo "Tauri sidecar binary built: $(BUILD_DIR)/debug/$(BINARY_NAME)-$(TARGET_TRIPLE)"

# Build both regular and Tauri-compatible binaries
build-all: build build-sidecar

# Build optimized Apple Silicon release binary
release-macos:
	@echo "Building optimized Apple Silicon release binary..."
	@$(MAKE) _build-optimized OUTPUT_PATH=build/$(BINARY_NAME)-darwin-arm64 GOOS=darwin GOARCH=arm64
	@echo "Generating checksums..."
	@cd $(BUILD_DIR) && shasum -a 256 $(BINARY_NAME)-darwin-arm64 > $(BINARY_NAME)-darwin-arm64.sha256
	@echo "Release binary built successfully:"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64
	@echo "SHA256:"
	@cat $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64.sha256

# Run development server with hot reloading (installs deps first)
dev: install-deps 
	@ENV=development ./scripts/shoreman.sh
	
# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f tmp/
	rm -f build-errors.log
	@echo "Build artifacts cleaned"

# Display the last 100 lines of development log with ANSI codes stripped
tail-log:
	@tail -100 ./dev.log | perl -pe 's/\e\[[0-9;]*m(?:\e\[K)?//g'