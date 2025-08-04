# Creative Agent Monorepo

This monorepo contains two main projects:

## Projects

### go_backend
A Go-based general agent with CLI interface and HTTP API endpoints.

**Key features:**
- Command-line interface for direct prompt processing
- HTTP API server functionality  
- Database integration with SQLite
- Multiple LLM provider support (Anthropic, OpenAI, Azure, etc.)
- File operations and tool integrations

**Getting started:**
```bash
cd go_backend
make build
# CLI mode
./mix -p "Your prompt here"
# Or HTTP server mode
./mix --http-port 8080
```

### mix_tauri_app
A Tauri-based desktop application with React frontend for chat functionality.

**Key features:**
- Cross-platform desktop app built with Tauri
- React frontend with TypeScript
- Chat interface with AI integration
- Modern UI components

**Getting started:**
```bash
cd mix_tauri_app
npm install
npm run tauri dev
```

## Configuration

The system requires explicit model configuration for both main and sub-agents. 

**Step 1:** Create a configuration file (`.mix.json`) in your home directory or project root:

```json
{
  "agents": {
    "main": {
      "model": "claude-4-sonnet",
      "maxTokens": 4096
    },
    "sub": {
      "model": "claude-4-sonnet", 
      "maxTokens": 2048
    }
  }
}
```

**Step 2:** Set the required API key as an environment variable:
```bash
export ANTHROPIC_API_KEY="your-api-key-here"
```

**Important:** API keys must always come from environment variables, never store them in configuration files. The system automatically detects available providers from environment variables and creates the necessary provider configurations.

The system will fail immediately if agents are not configured or required API keys are missing.

## Development

Each project maintains its own build system and dependencies. Refer to the individual README files in each project directory for specific development instructions.

## Structure

```
├── go_backend/          # Go backend service
├── mix_tauri_app/  # Tauri desktop application
├── .gitignore          # Monorepo gitignore
└── README.md           # This file
```