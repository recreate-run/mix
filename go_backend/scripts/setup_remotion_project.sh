#!/bin/bash
set -e  # Exit on error

PROJECT_DIR="$1"
TEMPLATE_REPO="https://github.com/sarath-menon/remotion-template-dynamic.git"

if [ -z "$PROJECT_DIR" ]; then
    echo "Error: Project directory not provided"
    exit 1
fi

echo "Setting up Remotion project at: $PROJECT_DIR"

# Clone template repository
echo "Cloning Remotion template..."
git clone "$TEMPLATE_REPO" "$PROJECT_DIR"

# Remove git history to avoid nested repositories
echo "Removing git history..."
rm -rf "$PROJECT_DIR/.git"

# Install npm dependencies
echo "Installing npm dependencies..."
cd "$PROJECT_DIR"
npm install

echo "Remotion project setup complete"