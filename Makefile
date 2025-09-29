.PHONY: build dev docs clean install install-air install-deps help update-blender-init release-macos go_lint go-test generate-mix_sdk gsap-server

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
	@echo "  dev         - Install dependencies and run all development servers (backend, frontend, GSAP)"
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
	@echo "  frontend-typecheck - Run TypeScript typecheck on frontend code"
	@echo "  frontend-lint - Run linter on frontend code"
	@echo "  go_lint     - Run golangci-lint on Go backend code"
	@echo "  go-test     - Run Go tests with coverage"
	@echo "  generate-openapi - Generate JSON OpenAPI spec"
	@echo "  help        - Show this help message"
	@echo ""


# Run development server with hot reloading (installs deps first)
# This starts backend, frontend, and GSAP server together
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
	cd mix_agent && go mod download
	@echo "Installing capture script dependencies..."
	# cd mix_agent && bun install
	@echo "Installing Tauri app dependencies..."
	cd mix_playground && bun i
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

# Run Go tests with coverage
go-test:
	@echo "Running Go tests..."
	@./scripts/tests/run_go_tests.sh

# Run TypeScript typecheck on frontend code
frontend-typecheck:
	@echo "Running frontend TypeScript typecheck..."
	cd mix_playground && bun run typecheck

frontend-lint:
	@echo "Running frontend lint..."
	cd mix_playground && bun knip

# Run golangci-lint on Go backend code
go_lint:
	@echo "Running golangci-lint on Go backend code..."
	cd mix_agent && golangci-lint run ./...

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
	@echo "Cleaning Tauri frontend (mix_playground)..."
	@rm -rf mix_playground/node_modules || true
	@rm -rf mix_playground/src-tauri/target || true
	@rm -rf mix_playground/dist || true
	@echo "Cleaning GSAP animations (packages/gsap_animations)..."
	@rm -rf packages/gsap_animations/build || true
	@rm -rf packages/gsap_animations/tmp || true
	@rm -rf packages/gsap_animations/node_modules || true
	@echo "✅ All build artifacts cleaned!"