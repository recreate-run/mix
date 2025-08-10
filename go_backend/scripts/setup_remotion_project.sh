#!/bin/bash
set -e  # Exit on error

PROJECT_DIR="$1"
TEMPLATE_REPO="https://github.com/sarath-menon/remotion_starter_template.git"

if [ -z "$PROJECT_DIR" ]; then
    echo "Error: Project directory not provided"
    exit 1
fi

echo "Setting up Remotion project at: $PROJECT_DIR"

# Clone template repository
echo "Cloning Remotion template..."
git clone "$TEMPLATE_REPO" "$PROJECT_DIR"

# Keep git history for updates via git pull
echo "Keeping git history for future updates..."

# Install npm dependencies
echo "Installing npm dependencies..."
cd "$PROJECT_DIR"
npm install

echo "Remotion project setup complete"