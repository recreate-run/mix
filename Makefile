.PHONY: build dev clean install install-air install-deps help update-blender-init release release-all release-macos release-linux release-windows

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
	@echo "  install     - Install system dependencies (one-time setup)"
	@echo "  install-deps - Install project dependencies"
	@echo "  build       - Build the binary to $(BUILD_DIR)/release/ directory"
	@echo "  build-sidecar - Build Tauri-compatible sidecar binary with platform suffix"
	@echo "  release     - Build complete release (Go backend + Tauri app for current platform)"
	@echo "  release-all - Build releases for all supported platforms"
	@echo "  release-macos - Build macOS release (Intel + Apple Silicon)"
	@echo "  release-linux - Build Linux release"
	@echo "  release-windows - Build Windows release"
	@echo "  clean       - Clean build artifacts"
	@echo "  install-air - Install Air if not present"
	@echo "  tail-log    - Show the last 100 lines of the log"
	@echo "  help        - Show this help message"
	@echo ""


# Run development server with hot reloading (installs deps first)
dev: install-deps 
	@ENV=development ./scripts/shoreman.sh

# Install system dependencies (one-time setup)
install:
	@echo "Installing system dependencies..."
	
	# Install Homebrew if not present (required for FFmpeg and preferred for other tools)
	@if ! command -v brew >/dev/null 2>&1; then \
		echo "📦 Installing Homebrew..."; \
		/bin/bash -c "$$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"; \
		eval "$$(/opt/homebrew/bin/brew shellenv)"; \
		export PATH="/opt/homebrew/bin:$$PATH"; \
	fi
	
	# Install FFmpeg if not present
	@if ! command -v ffmpeg >/dev/null 2>&1; then \
		echo "📦 Installing FFmpeg..."; \
		brew install ffmpeg; \
	fi
	
	# Install Go if not present
	@if ! command -v go >/dev/null 2>&1; then \
		echo "📦 Installing Go..."; \
		if command -v brew >/dev/null 2>&1; then \
			brew install go; \
		else \
			echo "Installing Go via official installer..."; \
			curl -L "https://go.dev/dl/go1.22.0.darwin-amd64.pkg" -o /tmp/go-installer.pkg && \
			sudo installer -pkg /tmp/go-installer.pkg -target / && \
			rm /tmp/go-installer.pkg; \
		fi; \
		export PATH="/usr/local/go/bin:$$PATH"; \
	fi
	
	# Install Rust/Cargo if not present
	@if ! command -v cargo >/dev/null 2>&1; then \
		echo "📦 Installing Rust/Cargo..."; \
		curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y; \
		source "$$HOME/.cargo/env"; \
		export PATH="$$HOME/.cargo/bin:$$PATH"; \
	fi
	
	# Install Bun if not present
	@if ! command -v bun >/dev/null 2>&1; then \
		echo "📦 Installing Bun..."; \
		curl -fsSL https://bun.sh/install | bash; \
		export PATH="$$HOME/.bun/bin:$$PATH"; \
	fi
	
	# Install UV if not present
	@if ! command -v uv >/dev/null 2>&1; then \
		echo "📦 Installing UV (Python package installer)..."; \
		if command -v brew >/dev/null 2>&1; then \
			brew install uv; \
		else \
			curl -LsSf https://astral.sh/uv/install.sh | sh; \
			export PATH="$$HOME/.local/bin:$$PATH"; \
		fi; \
	fi
	
	# Install tools
	@echo "Installing tools..."
	@uv tool install multimodal-analyzer
	
	@echo "✅ System dependencies installed!"

# Install project dependencies
install-deps: install
	@echo "Installing project dependencies..."
	@echo "Installing Air (Go hot reload)..."
	@command -v air >/dev/null 2>&1 || go install github.com/air-verse/air@latest
	@echo "Installing Go dependencies..."
	cd go_backend && go mod download
	@echo "Installing Tauri app dependencies..."
	cd tauri_app && bun i
	@echo "Installing remotion template dependencies..."
	cd packages/remotion_starter_template && bun install
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

# Release targets
RELEASE_DIR=releases
TAURI_DIR=tauri_app

# Build complete release for current platform
release: build-sidecar
	@echo "Building release for current platform..."
	@mkdir -p $(RELEASE_DIR)
	cd $(TAURI_DIR) && bun run tauri build
	@echo "✅ Release built successfully!"
	@echo "Desktop app: $(TAURI_DIR)/src-tauri/target/release/bundle/"
	@echo "Backend binary: $(BUILD_DIR)/release/"

# Build releases for all supported platforms
release-all: release-macos release-linux release-windows
	@echo "✅ All platform releases built!"

# Build macOS releases (Intel + Apple Silicon)
release-macos: install-deps
	@echo "Building macOS releases..."
	@mkdir -p $(RELEASE_DIR)
	
	# Build Go backend for both architectures
	@$(MAKE) _build-optimized OUTPUT_PATH=../$(BUILD_DIR)/release/$(BINARY_NAME)-x86_64-apple-darwin GOOS=darwin GOARCH=amd64
	@$(MAKE) _build-optimized OUTPUT_PATH=../$(BUILD_DIR)/release/$(BINARY_NAME)-aarch64-apple-darwin GOOS=darwin GOARCH=arm64
	
	# Build Tauri apps for both architectures
	cd $(TAURI_DIR) && bun run tauri build --target x86_64-apple-darwin
	cd $(TAURI_DIR) && bun run tauri build --target aarch64-apple-darwin
	
	@echo "✅ macOS releases built!"

# Build Linux release
release-linux: install-deps
	@echo "Building Linux release..."
	@mkdir -p $(RELEASE_DIR)
	
	# Build Go backend for Linux
	@$(MAKE) _build-optimized OUTPUT_PATH=../$(BUILD_DIR)/release/$(BINARY_NAME)-x86_64-unknown-linux-gnu GOOS=linux GOARCH=amd64
	
	# Build Tauri app for Linux
	cd $(TAURI_DIR) && bun run tauri build --target x86_64-unknown-linux-gnu
	
	@echo "✅ Linux release built!"

# Build Windows release  
release-windows: install-deps
	@echo "Building Windows release..."
	@mkdir -p $(RELEASE_DIR)
	
	# Build Go backend for Windows
	@$(MAKE) _build-optimized OUTPUT_PATH=../$(BUILD_DIR)/release/$(BINARY_NAME)-x86_64-pc-windows-msvc.exe GOOS=windows GOARCH=amd64
	
	# Build Tauri app for Windows
	cd $(TAURI_DIR) && bun run tauri build --target x86_64-pc-windows-msvc
	
	@echo "✅ Windows release built!"