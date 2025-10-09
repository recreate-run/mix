#!/bin/bash

# Script to deploy to Railway dev environment from local
# This builds from source instead of using release binaries

set -e

echo "🚀 Deploying to Railway dev environment..."
echo ""

# Check if on dev branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "dev" ]; then
    echo "⚠️  Warning: You're on branch '$CURRENT_BRANCH', not 'dev'"
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check Railway status
echo "📡 Checking Railway connection..."
railway status || {
    echo "❌ Not connected to Railway. Please run: railway login"
    exit 1
}

# Ensure we're on dev environment
echo "🔄 Switching to dev environment..."
railway environment dev || {
    echo "❌ Failed to switch to dev environment"
    exit 1
}

# Temporarily hide Dockerfile and railway.json so Railway builds from source
echo "🔧 Temporarily hiding Dockerfile and railway.json..."
if [ -f "Dockerfile" ]; then
    mv Dockerfile Dockerfile.prod
fi
if [ -f "railway.json" ]; then
    mv railway.json railway.json.prod
fi

# Deploy to Railway
echo "📦 Deploying to Railway dev..."
railway up --detach

# Restore files
echo "✅ Restoring Dockerfile and railway.json..."
if [ -f "Dockerfile.prod" ]; then
    mv Dockerfile.prod Dockerfile
fi
if [ -f "railway.json.prod" ]; then
    mv railway.json.prod railway.json
fi

echo ""
echo "🎉 Deployment started!"
echo "Check status: railway logs"
echo "Check service: https://mix-agent-dev.up.railway.app/health"
