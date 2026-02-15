#!/bin/bash
# Hot reload development script for browser-service
# Uses CompileDaemon to watch for file changes and auto-rebuild/restart

set -e

# Ensure CompileDaemon is installed
if ! command -v CompileDaemon &> /dev/null; then
  echo "Installing CompileDaemon..."
  go install github.com/githubnemo/CompileDaemon@latest
fi

# Rotate previous log
if [[ -f dev.log ]]; then
  cp dev.log dev-prev.log
fi
> dev.log

echo "=================================================================" >> dev.log
echo "$(date +"%H:%M:%S") BROWSER-SERVICE DEV SERVER STARTED" >> dev.log
echo "=================================================================" >> dev.log

echo "🔥 Starting browser-service with hot reload..."
echo "   WebSocket server on ws://localhost:8081"
echo "   Press Ctrl+C to stop"
echo "   Logs: dev.log (use 'task tail-dev-log' to view)"
echo ""

# Logging function that adds timestamps and writes to both stdout and dev.log
log_output() {
  local max_length="${MAX_LOG_LINE_LENGTH:-500}"

  while IFS= read -r data; do
    # Skip compiler warnings we don't care about
    if [[ "$data" == "ld: warning:"* ]] || [[ "$data" == "watching "* ]]; then
      continue
    fi

    local timestamp="$(date +"%H:%M:%S")"

    # Truncate long log entries
    if [ ${#data} -gt $max_length ]; then
      data="${data:0:$max_length}..."
    fi

    # Print to stdout
    printf "%s | %s\n" "$timestamp" "$data"
    # Write to log file (without color codes)
    printf "%s | %s\n" "$timestamp" "$data" >> dev.log
  done
}

# Run CompileDaemon with polling for file watching, pipe output through logger
# - polling: Use polling instead of fsnotify (more reliable cross-platform)
# - polling-interval: Check for changes every 500ms
# - log-prefix: Disable CompileDaemon timestamps
# - build: Command to build the binary
# - command: Command to run after successful build
# - exclude-dir: Ignore changes in build artifacts and vendor
# - graceful-kill: Send SIGTERM for clean shutdown before SIGKILL
CompileDaemon \
  -polling \
  -polling-interval=500 \
  -log-prefix=false \
  -exclude-dir=.git \
  -exclude-dir=bin \
  -exclude-dir=dist \
  -exclude-dir=vendor \
  -exclude-dir=test/testdata \
  -graceful-kill=true \
  -build="go build -o bin/browser-service ./cmd/server" \
  -command="bin/browser-service $@" \
  2>&1 | log_output
