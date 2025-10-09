#!/bin/bash

# Script to deploy to Railway dev environment
# Builds binary locally and deploys (fast, no source code upload)

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

# Build Linux binary locally
echo "🔨 Building Linux binary from source..."
cd mix_agent
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o ../mix-linux ./main.go
cd ..

# Verify binary was created
if [ ! -f "mix-linux" ]; then
    echo "❌ Failed to build binary"
    exit 1
fi

echo "✅ Binary built: mix-linux ($(du -h mix-linux | cut -f1))"
ls -lh mix-linux

# Backup original Dockerfile
echo "📝 Backing up original Dockerfile..."
if [ -f "Dockerfile" ]; then
    mv Dockerfile Dockerfile.backup
fi

# Create deployment Dockerfile (temporarily named as Dockerfile)
echo "📝 Creating deployment Dockerfile..."
cat > Dockerfile <<'EOF'
FROM alpine:latest

# Install dependencies
RUN apk --no-cache add ca-certificates bash ffmpeg

WORKDIR /app

# Copy the pre-built binary
COPY mix-linux ./mix

EXPOSE 8080

# Create startup script
RUN printf '#!/bin/bash\nset -e\nPORT=${PORT:-8080}\necho "Starting Mix Agent (DEV) on port $PORT..."\nexec ./mix --http-port "$PORT" --http-host 0.0.0.0\n' > /app/start.sh && \
    chmod +x /app/start.sh && \
    chmod +x /app/mix

ENTRYPOINT ["/bin/bash", "/app/start.sh"]
EOF

# Verify everything is ready before deploying
echo "🔍 Verifying deployment files..."
if [ ! -f "mix-linux" ]; then
    echo "❌ Binary not found before deployment!"
    exit 1
fi
if [ ! -f "Dockerfile" ]; then
    echo "❌ Dockerfile not found!"
    exit 1
fi

echo "Files ready:"
ls -lh mix-linux Dockerfile

# Temporarily remove .dockerignore (so mix-linux is included)
echo "🔧 Temporarily removing .dockerignore..."
if [ -f ".dockerignore" ]; then
    mv .dockerignore .dockerignore.backup
fi

# Deploy to Railway
echo "📦 Deploying to Railway dev..."
railway up --detach

# Cleanup
echo "🧹 Cleaning up..."
rm -f mix-linux

# Restore original Dockerfile
if [ -f "Dockerfile.backup" ]; then
    mv Dockerfile.backup Dockerfile
fi

# Restore .dockerignore
if [ -f ".dockerignore.backup" ]; then
    mv .dockerignore.backup .dockerignore
fi

echo ""
echo "🎉 Deployment started!"
echo "Check status: railway logs"
echo "Check service: https://mix-agent-dev.up.railway.app/health"
echo ""
