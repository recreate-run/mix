#!/bin/bash

# Load environment variables from .env file
if [ -f .env ]; then
  echo "Loading environment variables from .env file"
  
  # Export each non-comment line in the .env file
  while IFS= read -r line; do
    # Skip comments and empty lines
    if [[ ! "$line" =~ ^\s*# && -n "$line" ]]; then
      # Export the variable
      export "$line"
      # Extract variable name for logging (without value for security)
      var_name=$(echo "$line" | cut -d= -f1)
      echo "Loaded: $var_name"
    fi
  done < .env
else
  echo "No .env file found - using default environment settings"
fi