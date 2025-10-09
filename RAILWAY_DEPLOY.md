# Railway Deployment Guide

> **Note**: The Docker deployment automatically uses `--dangerously-skip-permissions` to bypass permission prompts in server environments where no interactive UI is available.

## Quick Deploy Steps

### 1. Install Railway CLI (if not already installed)

```bash
# macOS
brew install railway

# OR using npm
npm install -g @railway/cli
```

### 2. Login to Railway

```bash
railway login
```

### 3. Initialize Railway Project

```bash
# From the project root directory
railway init
```

This will create a new Railway project or link to an existing one.

### 4. Set Environment Variables

Replace the placeholder values with your actual credentials:

```bash
# Database Configuration (Turso)
railway variables set MIX_DB_TYPE=turso
railway variables set MIX_DB_TURSO_URL="YOUR_TURSO_URL"
railway variables set MIX_DB_TURSO_AUTH_TOKEN="YOUR_TURSO_TOKEN"

# Storage Configuration (Supabase)
railway variables set STORAGE_TYPE=supabase
railway variables set STORAGE_ENDPOINT="YOUR_SUPABASE_STORAGE_URL"
railway variables set STORAGE_BUCKET="YOUR_BUCKET_NAME"
railway variables set STORAGE_ACCESS_KEY="YOUR_SUPABASE_SERVICE_ROLE_KEY"
railway variables set STORAGE_PUBLIC_URL_BASE="YOUR_SUPABASE_PUBLIC_URL"

# Environment
railway variables set ENV=production

# Analytics (Optional)
railway variables set MIX_ANALYTICS_ENABLED=true

# Shell
railway variables set SHELL=/bin/sh
```

### 5. Deploy to Railway

```bash
railway up
```

This will:
- Build your Docker image
- Deploy to Railway
- Run migrations automatically

### 6. Monitor Deployment

```bash
# View logs
railway logs

# Get deployment URL
railway domain
```

### 7. Generate a Public Domain (Optional)

```bash
railway domain
```

Or add a custom domain in the Railway dashboard.

### 8. Configure AI Provider Credentials

After deployment, configure your AI provider API keys:

```bash
# Get your Railway URL
RAILWAY_URL=$(railway domain)

# Configure Anthropic
curl -X POST https://$RAILWAY_URL/api/auth/api-key \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "anthropic",
    "apiKey": "YOUR_ANTHROPIC_API_KEY"
  }'

# Configure OpenAI
curl -X POST https://$RAILWAY_URL/api/auth/api-key \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "apiKey": "YOUR_OPENAI_API_KEY"
  }'
```

## Testing Your Deployment

### Health Check

```bash
curl https://your-app.railway.app/health
```

Expected response:
```json
{
  "status": "ok",
  "timestamp": "2025-10-09T...",
  "version": "railway-deploy",
  "environment": "production",
  "services": {
    "backend": "healthy",
    "database": "connected"
  }
}
```

### Create a Test Session

```bash
curl -X POST https://your-app.railway.app/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"title": "Test Session"}'
```

### Send a Test Message

```bash
curl -X POST https://your-app.railway.app/api/sessions/{session-id}/messages \
  -H "Content-Type: application/json" \
  -d '{"content": "Hello, can you help me?"}'
```

## Troubleshooting

### Check Logs
```bash
railway logs --follow
```

### Restart Service
```bash
railway restart
```

### Check Environment Variables
```bash
railway variables
```

### SSH into Container (for debugging)
```bash
railway shell
```

## Alternative: Deploy via Railway Dashboard

1. Go to [railway.app](https://railway.app)
2. Click "New Project"
3. Choose "Deploy from GitHub repo"
4. Connect your GitHub account and select the repository
5. Set environment variables in the dashboard
6. Deploy!

## Cost Estimate

- **Railway Starter Plan**: $5/month
- **Turso Free Tier**: $0/month (up to 500 DBs, 9GB storage)
- **Supabase Free Tier**: $0/month (1GB storage, 2GB bandwidth)

**Total**: ~$5/month

## Next Steps

After successful deployment:

1. ✅ Test all API endpoints
2. ✅ Configure AI provider credentials
3. ✅ Set up custom domain (optional)
4. ✅ Monitor logs and performance
5. ✅ Set up CI/CD pipeline (optional)
