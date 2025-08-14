.PHONY: build dev clean install-air install-deps help update-blender-init

# Variables
BINARY_NAME=mix
BUILD_DIR=go_backend/build
MAIN_PATH=./go_backend/main.go

# Default target
help:
	@echo "Available targets:"
	@echo "  dev         - Install dependencies and run development servers"
	@echo "  install-deps - Install all project dependencies"
	@echo "  build       - Build the binary to $(BUILD_DIR)/ directory"
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
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

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