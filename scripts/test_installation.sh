#!/bin/bash

# Color codes for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Detect OS
OS="$(uname)"
echo -e "${BOLD}Testing installation on $OS${NC}\n"

# Initialize error counter
ERRORS=0

# Function to check for a command
check_command() {
  local cmd=$1
  local name=$2
  echo -n "Checking for $name... "
  
  if command -v "$cmd" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Found $(command -v "$cmd")${NC}"
    return 0
  else
    echo -e "${RED}✗ Not found${NC}"
    ERRORS=$((ERRORS + 1))
    return 1
  fi
}

# Check required commands
check_command ffmpeg "FFmpeg"
check_command go "Go"
check_command cargo "Rust/Cargo"
check_command bun "Bun"
check_command uv "UV (Python package manager)"
check_command rg "ripgrep"
check_command air "Air (Go hot reload)"

# Check for build directories
echo -e "\n${BOLD}Checking directory structure${NC}"
dirs=(
  "go_backend/build/debug"
  "tauri_app"
  "scripts"
)

for dir in "${dirs[@]}"; do
  echo -n "Checking directory $dir... "
  if [ -d "$dir" ]; then
    echo -e "${GREEN}✓ Found${NC}"
  else
    echo -e "${RED}✗ Not found${NC}"
    echo -e "  Creating directory $dir"
    mkdir -p "$dir"
  fi
done

# Check environment file
echo -e "\n${BOLD}Checking environment file${NC}"
if [ -f ".env" ]; then
  echo -e "${GREEN}✓ .env file exists${NC}"
else
  echo -e "${GREEN}ℹ No .env file found${NC}"
  echo -e "   The application will use default values."
  echo -e "   This is the expected behavior for fresh installations."
fi

# Check frontend environment file
echo -e "\n${BOLD}Checking frontend environment file${NC}"
if [ -f "tauri_app/.env" ]; then
  echo -e "${GREEN}✓ Frontend .env file exists${NC}"
else
  echo -e "${GREEN}ℹ No frontend .env file found${NC}"
  echo -e "   The frontend will use the default backend URL: http://localhost:8088"
  echo -e "   This is the expected behavior for fresh installations."
fi

# Summary
echo -e "\n${BOLD}Installation Test Summary${NC}"
echo "-------------------------"
if [ $ERRORS -eq 0 ]; then
  echo -e "${GREEN}✓ All required dependencies are installed!${NC}"
  exit 0
else
  echo -e "${RED}✗ Found $ERRORS missing dependencies!${NC}"
  echo -e "${YELLOW}Please run 'make install-deps' to install missing dependencies.${NC}"
  exit 1
fi