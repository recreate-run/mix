#!/bin/bash

# Color codes for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Backend port
BACKEND_PORT=8088

# Check if port is already in use
if lsof -i :$BACKEND_PORT -sTCP:LISTEN > /dev/null 2>&1; then
    echo -e "${RED}Error: Port $BACKEND_PORT is already in use.${NC}"
    echo -e "${YELLOW}This may be caused by:${NC}"
    echo "  1. Another instance of the backend is already running"
    echo "  2. Another application is using this port"
    echo -e "\n${YELLOW}Possible solutions:${NC}"
    echo "  1. Kill the existing process: lsof -ti:$BACKEND_PORT | xargs kill -9"
    echo "  2. Change the port in .air.toml and Procfile"
    exit 1
fi

# Create build directory if it doesn't exist
mkdir -p ./mix_agent/build/debug || {
    echo -e "${RED}Error: Failed to create build directory.${NC}"
    echo -e "${YELLOW}Make sure you have write permissions to ./mix_agent/build/debug${NC}"
    exit 1
}

# Load environment variables
echo -e "${BOLD}Loading environment variables...${NC}"
source ./scripts/load_env.sh || {
    echo -e "${RED}Error: Failed to load environment variables.${NC}"
    echo -e "${YELLOW}Make sure scripts/load_env.sh exists and is executable${NC}"
    exit 1
}

# Ensure the Go backend is built successfully
echo -e "${BOLD}Building Go backend...${NC}"
cd mix_agent || {
    echo -e "${RED}Error: Failed to change to mix_agent directory.${NC}"
    echo -e "${YELLOW}Make sure the mix_agent directory exists${NC}"
    exit 1
}

go build -o ./build/debug/mix ./main.go || {
    echo -e "${RED}Error: Failed to build Go backend.${NC}"
    echo -e "${YELLOW}Check for compilation errors in the Go code${NC}"
    exit 1
}

echo -e "${GREEN}Successfully built backend binary.${NC}"

# Run the Go backend with improved error handling
echo -e "${BOLD}Starting backend on port $BACKEND_PORT...${NC}"
./build/debug/mix --http-port $BACKEND_PORT --dangerously-skip-permissions || {
    echo -e "${RED}Error: Backend crashed on startup.${NC}"
    echo -e "${YELLOW}Check dev.log for more details${NC}"
    exit 1
}