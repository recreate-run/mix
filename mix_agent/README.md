# Mix Codebase Analysis

## Executive Summary

**Mix** is a sophisticated AI-powered assistant for general-purpose task automation and AI assistance. It provides CLI-only mode for scripting and structured data queries, along with an HTTP API for web integrations. The system is designed as a standalone tool with a clean, minimal interface that prioritizes simplicity and reliability.


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
