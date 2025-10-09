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

echo "✅ Binary built: mix-linux"

# Create temporary deployment Dockerfile
echo "📝 Creating temporary Dockerfile..."
cat > Dockerfile.deploy <<'EOF'
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

# Create temporary railway.json pointing to our Dockerfile
echo "⚙️  Creating temporary railway config..."
cat > railway.deploy.json <<'EOF'
{
  "$schema": "https://railway.app/railway.schema.json",
  "build": {
    "builder": "DOCKERFILE",
    "dockerfilePath": "Dockerfile.deploy"
  },
  "deploy": {
    "healthcheckPath": "/health",
    "healthcheckTimeout": 100,
    "restartPolicyType": "ON_FAILURE",
    "restartPolicyMaxRetries": 10
  }
}
EOF

# Temporarily rename original files
if [ -f "railway.json" ]; then
    mv railway.json railway.json.backup
fi

# Use our deployment config
mv railway.deploy.json railway.json

# Deploy to Railway
echo "📦 Deploying to Railway dev..."
railway up --detach

# Cleanup
echo "🧹 Cleaning up..."
rm -f mix-linux
rm -f Dockerfile.deploy

# Restore original railway.json
if [ -f "railway.json.backup" ]; then
    mv railway.json.backup railway.json
else
    rm -f railway.json
fi

echo ""
echo "🎉 Deployment started!"
echo "Check status: railway logs"
echo "Check service: https://mix-agent-dev.up.railway.app/health"
echo ""
