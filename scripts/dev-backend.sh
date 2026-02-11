#!/bin/bash
# Hot reload development script for mix backend
# Uses CompileDaemon to watch for file changes and auto-rebuild/restart

set -e

# Load environment variables
source ./scripts/load_env.sh

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
echo "$(date +"%H:%M:%S") MIX BACKEND DEV SERVER STARTED" >> dev.log
echo "=================================================================" >> dev.log

echo "🔥 Starting mix backend with hot reload..."
echo "   Press Ctrl+C to stop"
echo "   Logs: dev.log (use 'task tail-log' to view)"
echo ""

# Ensure build directory exists
mkdir -p mix_agent/build/debug

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
  -exclude-dir=mix_agent/build \
  -exclude-dir=mix_agent/tmp \
  -exclude-dir=mix_dev_tool \
  -exclude-dir=docs \
  -exclude-dir=node_modules \
  -exclude-dir=build \
  -exclude-dir=dist \
  -exclude-dir=vendor \
  -graceful-kill=true \
  -build="go build -o mix_agent/build/debug/mix ./mix_agent" \
  -command="mix_agent/build/debug/mix --http-port 8088 --dangerously-skip-permissions" \
  2>&1 | log_output
