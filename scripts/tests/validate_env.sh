#!/bin/bash
# Environment Variable Validation Script
# This script validates that all required environment variables are set
# and provides warnings for missing variables.

# Color codes for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color
BOLD='\033[1m'

echo -e "${BOLD}Environment Variable Validation${NC}"
echo "-----------------------------"

# Function to check if variable exists in .env file
variable_in_env_file() {
  local var_name="$1"
  local env_file="$2"
  if [ -f "$env_file" ]; then
    grep -q "^${var_name}=" "$env_file" && return 0
    grep -q "^${var_name} =" "$env_file" && return 0
    return 1
  else
    return 1
  fi
}

# Function to check environment variable
# $1: variable name
# $2: severity (error|warning)
# $3: description
# $4: default value (optional)
check_variable() {
  local var_name="$1"
  local severity="${2:-error}"
  local description="$3"
  local default_value="$4"
  local has_default=false
  
  if [ -n "$default_value" ]; then
    has_default=true
  fi

  # Check if environment variable is set
  if [ -z "${!var_name}" ]; then
    # Check if it exists in .env file but just hasn't been loaded
    if variable_in_env_file "$var_name" ".env"; then
      echo -e "⚠️  ${YELLOW}WARNING:${NC} $var_name is in .env file but not loaded in environment"
      echo -e "   → ${YELLOW}Description:${NC} $description"
      echo -e "   → ${YELLOW}Resolution:${NC} Run 'source .env' or restart make dev"
      ENV_WARNINGS=$((ENV_WARNINGS + 1))
    elif [ "$severity" == "error" ] && [ "$has_default" == "false" ]; then
      echo -e "❌ ${RED}ERROR:${NC} Required variable $var_name is not set"
      echo -e "   → ${RED}Description:${NC} $description"
      echo -e "   → ${RED}Resolution:${NC} Add $var_name to .env file"
      ENV_ERRORS=$((ENV_ERRORS + 1))
    elif [ "$severity" == "warning" ]; then
      echo -e "⚠️  ${YELLOW}WARNING:${NC} $var_name is not set"
      echo -e "   → ${YELLOW}Description:${NC} $description"
      if [ "$has_default" == "true" ]; then
        echo -e "   → ${YELLOW}Default:${NC} Will use '$default_value' as fallback"
      else
        echo -e "   → ${YELLOW}Resolution:${NC} Add $var_name to .env file if needed"
      fi
      ENV_WARNINGS=$((ENV_WARNINGS + 1))
    else
      echo -e "⚠️  ${YELLOW}NOTICE:${NC} $var_name is not set but has default value '$default_value'"
      ENV_WARNINGS=$((ENV_WARNINGS + 1))
    fi
  else
    echo -e "✅ ${GREEN}OK:${NC} $var_name is set"
  fi
}

# Initialize counters
ENV_ERRORS=0
ENV_WARNINGS=0

echo "Checking frontend environment variables..."
check_variable "VITE_BACKEND_URL" "warning" "Backend URL for frontend API requests" "http://localhost:8088"

echo -e "\nSummary:"
echo -e "--------"
if [ $ENV_ERRORS -eq 0 ] && [ $ENV_WARNINGS -eq 0 ]; then
  echo -e "✅ ${GREEN}All environment variables validated successfully!${NC}"
  exit 0
elif [ $ENV_ERRORS -eq 0 ]; then
  echo -e "⚠️  ${YELLOW}$ENV_WARNINGS warning(s) found but no critical errors.${NC}"
  echo -e "   Development can continue but some features may not work correctly."
  exit 0
else
  echo -e "❌ ${RED}$ENV_ERRORS error(s) and $ENV_WARNINGS warning(s) found.${NC}"
  echo -e "   Please fix the errors before continuing."
  exit 1
fi