# Mix Codebase Analysis

## Executive Summary

**Mix** is a sophisticated AI-powered assistant for general-purpose task automation and AI assistance. It provides CLI-only mode for scripting and structured data queries, along with an HTTP API for web integrations. The system is designed as a standalone tool with a clean, minimal interface that prioritizes simplicity and reliability.

## Project Overview

### Core Technology Stack
- **Language**: Go 1.24.0
- **Database**: SQLite with SQLC for type-safe queries
- **CLI Framework**: Cobra for command-line interface
- **Configuration**: Viper for configuration management
- **Build System**: Go modules with custom build scripts

### Architecture Philosophy
- **Minimal Complexity**: Simple, maintainable code with clean interfaces
- **Dual Interface Model**: CLI for scripting and HTTP API for web applications
- **Data Query System**: JSON-RPC structured data access via CLI
- **Multi-Provider AI**: Support for multiple AI backends (Anthropic, OpenAI, etc.)
- **Tool Extensibility**: MCP protocol for external integrations

## Main Components

### 1. Entry Points (`main.go`, `cmd/`)
- Single main entry point with Cobra CLI framework
- **Default**: CLI mode with explicit prompt flag required
- CLI-only mode for scripting and automation (`-p "prompt"`)
- **Data query mode** for structured data access (`--query <type>`)
- Graceful shutdown and panic recovery

### 2. Core Application (`internal/app/`)
- Central app orchestrator managing all services
- Session management and message handling
- CLI-only execution flow for scripting
- Automatic permission approval for non-interactive sessions

### 3. LLM Integration (`internal/llm/`)
- **Agents**: Main coder agent with tool orchestration
- **Models**: Support for multiple AI providers (OpenAI, Anthropic, Azure, Gemini, Groq, etc.)
- **Providers**: Provider-specific implementations for each AI service
- **Tools**: Comprehensive AI assistant tools (bash, file ops, grep, edit, etc.)
- **Prompts**: Embedded markdown prompts with templating support

#### Session Isolation Architecture

**Session-Specific Resources (Isolated):**
- **System Prompts**: Each session gets a unique provider with session-aware system prompts
- **Session Context**: Variables like `session_id`, `session_workdir` are injected per-session
- **Request Processing**: Each request creates an isolated provider with session context

**Shared Resources (By Design):**
- **Tools**: All sessions share the same tool instances (`[]tools.BaseTool`)
- **Agent State**: `activeRequests` sync.Map tracks cross-session state
- **Providers**: Title and summarize providers are shared across sessions
- **Services**: Database (sessions/messages) and event pub/sub are shared
- **Global Caches**: Project context and model definitions are cached globally

This architecture ensures prompt isolation while sharing tools and services efficiently.

### 4. Data Query Interface (`internal/api/`)
- **JSON-RPC query system** for structured data access
- **CLI interface**: `--query <type> --output-format json`
- **Query types**: `sessions`, `tools`, `mcp`, `commands`
- Perfect for **native app integration** (Swift, Electron, etc.)

### 5. Data Layer (`internal/db/`)
- SQLite database with proper migrations
- Three core entities: Sessions, Messages, Files
- SQLC for type-safe database operations
- Automatic timestamping and relationship management

### 6. Supporting Services
- **Permissions**: Request approval system (can be bypassed with `--dangerously-skip-permissions` for trusted environments)
- **Logging**: Structured logging throughout
- **File Operations**: Safe file manipulation with history tracking
- **Session Management**: Conversation state management
- **Message Handling**: Multi-part message processing with attachments

## Database Schema
```sql
sessions (conversations)
├── messages (user/assistant exchanges)
└── files (file versions and content tracking)
```

## CLI Data Query Interface

Mix provides a powerful structured data access system through its CLI interface that enables programmatic interaction with sessions, tools, and system state. This command-line API is designed for seamless integration with native applications and scripts.

### CLI Query Interface

Get structured JSON data directly via stdout:

```bash
# Get all sessions
./build/mix --query sessions --output-format json

# Get available tools (including MCP tools)
./build/mix --query tools --output-format json

# Get MCP server status and their tools
./build/mix --query mcp --output-format json

# Get available slash commands
./build/mix --query commands --output-format json
```

### HTTP Server Interface

Mix also provides an HTTP JSON-RPC server for web-based integrations:

```bash
# Start HTTP server on default port (localhost:8080)
./build/mix --http-port 8080

# Start HTTP server on custom host and port
./build/mix --http-port 3000 --http-host 0.0.0.0

# Start HTTP server with permissions skipped (for development/trusted environments)
./build/mix --http-port 8080 --dangerously-skip-permissions

# Run HTTP server with debug logging
./build/mix --http-port 8080 --debug
```

#### HTTP API Usage

The HTTP server provides two main endpoints:

**JSON-RPC Endpoint (`/rpc`)** - Request/response API:

```bash
# Get sessions via HTTP
curl -X POST http://localhost:8080/rpc \
  -H "Content-Type: application/json" \
  -d '{"method": "sessions.list", "id": 1}'

# Create new session via HTTP
curl -X POST http://localhost:8080/rpc \
  -H "Content-Type: application/json" \
  -d '{"method": "sessions.create", "params": {"title": "New Session"}, "id": 1}'

# Send message to session
curl -X POST http://localhost:8080/rpc \
  -H "Content-Type: application/json" \
  -d '{"method": "messages.send", "params": {"sessionId": "uuid", "content": "Hello"}, "id": 1}'
```

**SSE Streaming Endpoint (`/stream`)** - Real-time agent responses:

```bash
# Stream agent response via GET
curl -N -H "Accept: text/event-stream" \
  "http://localhost:8080/stream?sessionId=uuid&content=Hello"

# Stream agent response via POST
curl -N -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -d '{"sessionId": "uuid", "content": "Hello"}' \
  http://localhost:8080/stream
```

**SSE Event Types:**
- `connected` - Connection established with session ID
- `tool` - Tool execution events (with status: pending/running/completed)
- `complete` - Response finished (includes final content)
- `error` - Error occurred

*Note: Only agent progress (tool executions) streams in real-time. Final content is delivered in the completion event for better performance.*

### Native Integration Examples

**Via HTTP:**
```swift
// HTTP JSON-RPC request from Swift
struct RPCRequest: Codable {
    let method: String
    let params: [String: Any]?
    let id: Int
}

let request = RPCRequest(method: "sessions.list", params: nil, id: 1)
let url = URL(string: "http://localhost:8080/rpc")!
var urlRequest = URLRequest(url: url)
urlRequest.httpMethod = "POST"
urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
urlRequest.httpBody = try JSONEncoder().encode(request)

let (data, _) = try await URLSession.shared.data(for: urlRequest)
let sessions = try JSONDecoder().decode([SessionData].self, from: data)
```

#### JavaScript/Web Integration

**SSE Streaming with EventSource:**
```javascript
// Real-time agent progress streaming in web applications
const sessionId = 'your-session-id';
const content = 'Hello, can you help me?';
const url = `http://localhost:8080/stream?sessionId=${sessionId}&content=${encodeURIComponent(content)}`;

const eventSource = new EventSource(url);

eventSource.addEventListener('connected', (event) => {
    const data = JSON.parse(event.data);
    console.log('Connected to session:', data.sessionId);
});


eventSource.addEventListener('tool', (event) => {
    const data = JSON.parse(event.data);
    console.log(`Tool ${data.name} (${data.status}):`, data.input);
    // Update UI with tool execution progress
});

eventSource.addEventListener('complete', (event) => {
    const data = JSON.parse(event.data);
    console.log('Response complete:', data.content); // Final content available here
    eventSource.close();
});

eventSource.addEventListener('error', (event) => {
    const data = JSON.parse(event.data);
    console.error('Error:', data.error);
    eventSource.close();
});
```

#### Session Management API (2-Way Communication)

```bash
# Create a new session
echo '{"method": "sessions.create", "params": {"title": "New Analysis", "setCurrent": true}, "id": 1}' | \
./build/mix --query json --output-format json

# Select a different session
echo '{"method": "sessions.select", "params": {"id": "session-uuid"}, "id": 1}' | \
./build/mix --query json --output-format json

# Get current session
echo '{"method": "sessions.current", "id": 1}' | \
./build/mix --query json --output-format json

# Delete a session
echo '{"method": "sessions.delete", "params": {"id": "session-uuid"}, "id": 1}' | \
./build/mix --query json --output-format json
```

Both CLI and HTTP interfaces provide full 2-way communication for session management, enabling programmatic control of Mix from external applications or scripts. The HTTP interface offers better performance for web-based integrations, while the CLI interface is ideal for shell scripts and simple integrations.

### Query Response Formats

**Sessions Response:**
```json
[{
  "id": "uuid",
  "title": "Session Title", 
  "messageCount": 5,
  "promptTokens": 1000,
  "completionTokens": 800,
  "cost": 0.0023,
  "createdAt": "2024-07-22T18:57:03+02:00"
}]
```

**MCP Response:**
```json
[{
  "name": "blender",
  "connected": true,
  "status": "connected",
  "tools": [{"name": "execute_blender_code", "description": "..."}]
}]
```

**Tools Response:**
```json
[{"name": "bash", "description": "Execute shell commands"}]
```
