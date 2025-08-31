#!/bin/bash

# Create build directory if it doesn't exist
mkdir -p ./go_backend/build/debug

# Load environment variables
source ./scripts/load_env.sh


# Run the Go backend directly with required parameters
cd go_backend && go build -o ./build/debug/mix ./main.go && ./build/debug/mix --http-port 8088 --dangerously-skip-permissions