#!/bin/bash
# Frontend-Backend Connection Test Script
# This script tests the connection between frontend and backend services

# Color codes for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Default backend URL
DEFAULT_BACKEND_URL="http://localhost:8088"
# Backend port
BACKEND_PORT=8088
# Frontend port
FRONTEND_PORT=1420
# Maximum retry attempts
MAX_RETRIES=5
# Timeout for health check (in seconds)
TIMEOUT=2

echo -e "${BOLD}Frontend-Backend Connection Test${NC}"
echo "------------------------------"

# Step 1: Check if backend is running
echo -e "\n${BOLD}1. Checking if backend is running...${NC}"

# Function to check if backend is running
check_backend() {
  local backend_url="${1:-$DEFAULT_BACKEND_URL}"
  local attempt=${2:-1}
  
  echo -e "   Attempt $attempt: Checking backend at $backend_url/health..."
  
  # Use curl with timeout to check backend health
  if curl -s -f -m $TIMEOUT "$backend_url/health" > /dev/null; then
    echo -e "✅ ${GREEN}Backend is running at $backend_url${NC}"
    return 0
  else
    echo -e "❌ ${RED}Backend is not accessible at $backend_url${NC}"
    return 1
  fi
}

# Try to connect to backend with retries
backend_running=false
for i in $(seq 1 $MAX_RETRIES); do
  if check_backend "$DEFAULT_BACKEND_URL" "$i"; then
    backend_running=true
    break
  else
    sleep 2
  fi
done

if [ "$backend_running" = false ]; then
  echo -e "❌ ${RED}Failed to connect to backend after $MAX_RETRIES attempts${NC}"
  echo -e "${YELLOW}Make sure the backend is running by checking:${NC}"
  echo -e "  1. Run 'make tail-log' to see if there are any backend startup errors"
  echo -e "  2. Verify the backend port ($BACKEND_PORT) is not in use by another application"
  echo -e "  3. Check if the backend script has any issues"
  exit 1
fi

# Step 2: Fetch detailed backend health information
echo -e "\n${BOLD}2. Fetching backend health information...${NC}"
if health_info=$(curl -s "$DEFAULT_BACKEND_URL/health"); then
  # Extract relevant information from health response
  status=$(echo "$health_info" | grep -o '"status":"[^"]*"' | cut -d':' -f2 | tr -d '"')
  version=$(echo "$health_info" | grep -o '"version":"[^"]*"' | cut -d':' -f2 | tr -d '"')
  environment=$(echo "$health_info" | grep -o '"environment":"[^"]*"' | cut -d':' -f2 | tr -d '"')
  
  echo -e "   Backend Status: ${GREEN}$status${NC}"
  echo -e "   Version: $version"
  echo -e "   Environment: $environment"
else
  echo -e "⚠️  ${YELLOW}Could not fetch detailed health information${NC}"
fi

# Step 3: Check if frontend is running
echo -e "\n${BOLD}3. Checking if frontend is running...${NC}"

# Function to check if frontend is running
check_frontend() {
  local frontend_url="http://localhost:$FRONTEND_PORT"
  
  # Use curl with timeout to check if frontend is responding
  if curl -s -f -m $TIMEOUT "$frontend_url" > /dev/null; then
    echo -e "✅ ${GREEN}Frontend is running at $frontend_url${NC}"
    return 0
  else
    echo -e "❌ ${RED}Frontend is not accessible at $frontend_url${NC}"
    return 1
  fi
}

# Try to connect to frontend
if check_frontend; then
  frontend_running=true
else
  frontend_running=false
  echo -e "${YELLOW}Make sure the frontend is running by checking:${NC}"
  echo -e "  1. Run 'make tail-log' to see if there are any frontend startup errors"
  echo -e "  2. Verify the frontend port ($FRONTEND_PORT) is not in use by another application"
fi

# Step 4: Check if .env file in mix_dev_tool has correct backend URL
echo -e "\n${BOLD}4. Checking frontend environment configuration...${NC}"
FRONTEND_ENV_FILE="mix_dev_tool/.env"

if [ -f "$FRONTEND_ENV_FILE" ]; then
  echo -e "✅ ${GREEN}Frontend .env file exists${NC}"
  
  # Check if VITE_BACKEND_URL is correctly set
  if grep -q "VITE_BACKEND_URL=.*$BACKEND_PORT" "$FRONTEND_ENV_FILE"; then
    echo -e "✅ ${GREEN}VITE_BACKEND_URL is correctly set in frontend .env file${NC}"
  else
    echo -e "⚠️  ${YELLOW}VITE_BACKEND_URL might not be correctly set in frontend .env file${NC}"
    echo -e "   Check $FRONTEND_ENV_FILE to ensure it contains: VITE_BACKEND_URL=$DEFAULT_BACKEND_URL"
  fi
else
  echo -e "ℹ️  ${GREEN}Frontend .env file not found at $FRONTEND_ENV_FILE${NC}"
  echo -e "   The app will use the default backend URL: $DEFAULT_BACKEND_URL"
  echo -e "   This is the expected behavior for fresh installations."
fi

# Step 5: Test actual API request
echo -e "\n${BOLD}5. Testing an actual API request...${NC}"
if curl -s "$DEFAULT_BACKEND_URL/api/file-types" > /dev/null; then
  echo -e "✅ ${GREEN}Successfully made API request to /api/file-types${NC}"
else
  echo -e "⚠️  ${YELLOW}API request to /api/file-types failed${NC}"
  echo -e "   This endpoint might not be available in your environment."
fi

# Summary
echo -e "\n${BOLD}Connection Test Summary${NC}"
echo "----------------------"
if [ "$backend_running" = true ]; then
  if [ "$frontend_running" = true ]; then
    echo -e "✅ ${GREEN}Both frontend and backend are running and properly connected!${NC}"
    exit 0
  else
    echo -e "⚠️  ${YELLOW}Backend is running, but frontend may have issues.${NC}"
    exit 1
  fi
else
  echo -e "❌ ${RED}Backend connection issues detected.${NC}"
  exit 1
fi