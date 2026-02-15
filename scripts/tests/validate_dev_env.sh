#!/bin/bash
# Development Environment Validation Script
# This script validates that the development environment is properly set up
# and all required services can start correctly.

# Color codes for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color
BOLD='\033[1m'

echo -e "${BOLD}Development Environment Validation${NC}"
echo "-------------------------------"

# Initialize error counter
ERRORS=0

# Step 1: Check if required tools are installed
echo -e "\n${BOLD}1. Checking required tools...${NC}"
check_tool() {
  local tool=$1
  local required=${2:-true}

  if command -v "$tool" > /dev/null 2>&1; then
    echo -e "✅ ${GREEN}$tool is installed${NC}"
    return 0
  else
    if [ "$required" = "true" ]; then
      echo -e "❌ ${RED}$tool is NOT installed${NC}"
      ERRORS=$((ERRORS + 1))
      return 1
    else
      echo -e "⚠️  ${YELLOW}$tool is NOT installed (will be installed by make dev)${NC}"
      return 0
    fi
  fi
}

check_tool go true
check_tool bun true
check_tool cargo true
check_tool uv true

# Step 2: Validate Procfile configuration
echo -e "\n${BOLD}2. Validating Procfile configuration...${NC}"
if [ -f "Procfile" ]; then
  echo -e "✅ ${GREEN}Procfile exists${NC}"

  # Check backend command
  if grep -q "backend:" Procfile; then
    echo -e "✅ ${GREEN}Procfile contains backend service${NC}"
  else
    echo -e "❌ ${RED}Procfile is missing backend service${NC}"
    ERRORS=$((ERRORS + 1))
  fi

  # Check frontend command
  if grep -q "frontend:" Procfile; then
    echo -e "✅ ${GREEN}Procfile contains frontend service${NC}"
  else
    echo -e "❌ ${RED}Procfile is missing frontend service${NC}"
    ERRORS=$((ERRORS + 1))
  fi
else
  echo -e "❌ ${RED}Procfile not found${NC}"
  ERRORS=$((ERRORS + 1))
fi

# Step 3: Validate scripts needed by Procfile
echo -e "\n${BOLD}3. Validating scripts used in Procfile...${NC}"
if grep -q "./scripts/run_backend.sh" Procfile; then
  if [ -x "./scripts/run_backend.sh" ]; then
    echo -e "✅ ${GREEN}run_backend.sh exists and is executable${NC}"
  else
    echo -e "❌ ${RED}run_backend.sh doesn't exist or is not executable${NC}"
    ERRORS=$((ERRORS + 1))
  fi
fi

# Step 4: Check environment variables
echo -e "\n${BOLD}4. Validating environment variables...${NC}"
# Run the env validation script
./scripts/tests/validate_env.sh
ENV_EXIT_CODE=$?
if [ $ENV_EXIT_CODE -ne 0 ]; then
  ERRORS=$((ERRORS + 1))
fi

# Step 5: Build directory structure check
echo -e "\n${BOLD}5. Checking build directories...${NC}"
if [ -d "./mix_agent/build/debug" ]; then
  echo -e "✅ ${GREEN}Backend build directory exists${NC}"
else
  echo -e "⚠️  ${YELLOW}Backend build directory doesn't exist, will be created at runtime${NC}"
  # Not incrementing errors as this will be created
fi

if [ -d "./mix_dev_tool/target" ]; then
  echo -e "✅ ${GREEN}Frontend build directory exists${NC}"
else
  echo -e "⚠️  ${YELLOW}Frontend build directory doesn't exist, will be created at runtime${NC}"
  # Not incrementing errors as this will be created
fi

# Summary
echo -e "\n${BOLD}Development Environment Validation Summary${NC}"
echo "----------------------------------------"
if [ $ERRORS -eq 0 ]; then
  echo -e "✅ ${GREEN}All checks passed! Development environment is ready.${NC}"
  exit 0
else
  echo -e "❌ ${RED}Found $ERRORS issue(s) that need to be resolved.${NC}"
  echo -e "${YELLOW}Please fix these issues before running 'make dev'.${NC}"
  exit 1
fi
