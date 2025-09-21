# 🎯 Complete Mix Repository Learning Plan

**Status**: ⏳ Ready to Start  
**Last Updated**: 2025-01-19  
**Agent Instructions**: Update status checkboxes as you complete each section. Mark with ✅ when done, 🔄 when in progress, ❌ if blocked.

---

## **Phase 1: Foundation & Setup (Days 1-2)**

### Level 1: Understanding the Big Picture
**Objective**: Get oriented with what Mix is and does

- [ ] **Read Documentation** ⭐ START HERE
  - [ ] Read `README.md` - understand it's an AI coding assistant like Claude Code
  - [ ] Read `CLAUDE.md` - critical development guidelines 
  - [ ] Study the tech stack badges - recognize the tools

- [ ] **Architecture Overview**
  - [ ] **Backend**: Go HTTP server with SQLite database
  - [ ] **Frontend**: Tauri 2.0 desktop app with React 19
  - [ ] **Communication**: REST API + Server-Sent Events (SSE)
  - [ ] **Purpose**: Local AI agent for coding tasks with file management

- [ ] **Quick Setup** 
  ```bash
  make install     # Install dependencies
  make dev        # Start both frontend and backend
  ```

### Level 2: Development Environment
**Files to understand**: `Makefile`, `CLAUDE.md`, `scripts/shoreman.sh`

- [ ] **Development Commands**
  - [ ] `make dev` - runs everything (backend + frontend)
  - [ ] `make tail-log` - see unified logs 
  - [ ] `make frontend-typecheck` - check TypeScript
  - [ ] Never stop the dev server - it auto-reloads!

- [ ] **Logging System**
  - [ ] All output goes to `dev.log`
  - [ ] Shoreman runs multiple processes
  - [ ] Console logs forwarded from browser to terminal

---

## **Phase 2: Backend Deep Dive (Days 3-5)**

### Level 3: Go Backend Fundamentals
**Key directories**: `go_backend/cmd/`, `go_backend/internal/`

- [ ] **Entry Points** 📍 `go_backend/main.go:1`
  - [ ] Simple main function with panic recovery
  - [ ] Delegates to Cobra CLI in `cmd/root.go:24`

- [ ] **Application Modes** 📍 `go_backend/cmd/root.go:45`
  - [ ] CLI mode: `./mix -p "your prompt"`
  - [ ] HTTP mode: `./mix --http-port 8080` 
  - [ ] Query mode: `./mix --query`

- [ ] **Database Layer** 📍 `go_backend/internal/db/`
  - [ ] SQLite with SQLC for type-safe queries
  - [ ] Key tables: sessions, messages, files, user_preferences
  - [ ] Study migrations in `migrations/` folder

### Level 4: Core Backend Components

- [ ] **HTTP Server** 📍 `go_backend/internal/http/server.go:89`
  - [ ] RESTful API endpoints
  - [ ] CORS middleware for frontend communication
  - [ ] SSE streaming at `/stream` endpoint

- [ ] **LLM Provider System** 📍 `go_backend/internal/llm/provider/provider.go:25`
  - [ ] Abstract interface supporting multiple providers
  - [ ] Anthropic, OpenAI, Gemini, etc.
  - [ ] Provider caching per session

- [ ] **Tool System** 📍 `go_backend/internal/llm/tools/tools.go:15`
  - [ ] `BaseTool` interface with `Info()` and `Run()`
  - [ ] Built-in tools: file ops, bash, planning, media
  - [ ] MCP protocol for external tools

### Level 5: Data Flow Understanding

- [ ] **Agent Processing** 📍 `go_backend/internal/llm/agent/agent.go:78`
  - [ ] Message → LLM Provider → Tool Execution → Response
  - [ ] Streaming events for real-time UI updates
  - [ ] Session-scoped provider caching

- [ ] **Session Management** 📍 `go_backend/internal/session/session.go:45`
  - [ ] Lifecycle management with storage directories
  - [ ] Fork functionality for branching conversations
  - [ ] Token usage and cost tracking

---

## **Phase 3: Frontend Deep Dive (Days 6-8)**

### Level 6: React Architecture Fundamentals
**Key directories**: `tauri_app/src/components/`, `tauri_app/src/hooks/`

- [ ] **Main App Structure** 📍 `tauri_app/src/main.tsx:15`
  - [ ] TanStack Router setup
  - [ ] PostHog analytics initialization
  - [ ] Theme and query providers

- [ ] **Routing System** 📍 `tauri_app/src/routes/`
  - [ ] `index.tsx` - auto-redirect logic
  - [ ] `$sessionId.tsx` - main chat interface
  - [ ] Type-safe routing with TanStack Router

- [ ] **Chat Interface** 📍 `tauri_app/src/components/chat-app.tsx:150` (1,490 lines!)
  - [ ] Heart of the application - understand streaming completion logic (lines 1056-1091)
  - [ ] Message display, input handling, file attachments
  - [ ] Real-time streaming from SSE with tool call conversion
  - [ ] Media showcase integration and timeline handling

### Level 7: State Management & Data Flow

- [ ] **TanStack Query** 📍 `tauri_app/src/hooks/`
  - [ ] `useSessionMessages.ts` - loads conversation history
  - [ ] `usePersistentSSE.ts` - real-time message streaming
  - [ ] `useSessionsList.ts` - session management
  - [ ] Smart caching with `cache-keys.ts`

- [ ] **Zustand Store** 📍 `tauri_app/src/stores/attachmentSlice.ts:45`
  - [ ] Manages file attachments (10-item limit)
  - [ ] URL auto-detection
  - [ ] Bidirectional sync with text input

### Level 8: UI Components & Patterns

- [ ] **Design System** 📍 `tauri_app/src/components/ui/`
  - [ ] Radix UI foundation with custom styling
  - [ ] `sidebar.tsx` - collapsible session navigation
  - [ ] `command.tsx` - slash command palette

- [ ] **Specialized Components**
  - [ ] `conversation-display.tsx` - message timeline
  - [ ] `response-renderer.tsx` - rich message rendering
  - [ ] `attachment-preview.tsx` - file display system

---

## **Phase 4: Integration & Advanced Features (Days 9-11)**

### Level 9: Frontend-Backend Communication

- [ ] **SDK Integration** 📍 `tauri_app/src/lib/mix-sdk.ts:12`
  ```typescript
  export const mix = new Mix({
    serverURL: getBackendUrl(),
  });
  ```

- [ ] **Message Flow** 📍 `tauri_app/src/types/message.ts:25`
  - [ ] UI Messages vs Backend Messages
  - [ ] Conversion layer between formats
  - [ ] SSE streaming for real-time updates

- [ ] **Command System** 📍 `tauri_app/src/handlers/`
  - [ ] Slash commands: `/login`, `/model`, `/status`
  - [ ] OAuth authentication flows
  - [ ] Model selection and configuration

### Level 10: Advanced Features

- [ ] **File System** 📍 `go_backend/internal/storage/storage.go:45`
  - [ ] Session-scoped storage in `storage/{session-id}/`
  - [ ] OS-level path traversal protection
  - [ ] File references with `@filename` syntax

- [ ] **Permission System** 📍 `go_backend/internal/permission/permission.go:25`
  - [ ] Tool execution permissions
  - [ ] Real-time permission dialogs
  - [ ] Session-scoped permission tracking

- [ ] **Real-time Features**
  - [ ] SSE streaming for live chat
  - [ ] Auto-scroll to new messages
  - [ ] Cancel/resume message processing

---

## **Phase 5: Development Patterns & Best Practices (Days 12-14)**

### Level 11: Code Organization Patterns

- [ ] **Frontend Patterns**
  - [ ] Hooks for data fetching (`useSessionMessages`)
  - [ ] Components for UI (`conversation-display`)
  - [ ] Stores for client state (`attachmentSlice`)
  - [ ] Utils for business logic (`messageUtils.ts`)

- [ ] **Backend Patterns**
  - [ ] Services for business logic (`session.go`)
  - [ ] Handlers for HTTP endpoints (`rest_sessions.go`)
  - [ ] Models for data access (`models.go`)
  - [ ] Tools for agent capabilities (`tools.go`)

### Level 12: Key Development Decisions

- [ ] **Architecture Choices**
  - [ ] **Database-first**: Preferences in DB, not config files
  - [ ] **Session-scoped**: Each conversation isolated
  - [ ] **Streaming-first**: Real-time updates via SSE
  - [ ] **Tool extensibility**: MCP protocol support

- [ ] **Performance Optimizations**
  - [ ] Query caching with TanStack Query
  - [ ] Provider caching per session
  - [ ] Attachment limits (10 max)
  - [ ] Component lazy loading

---

## **Phase 6: Hands-On Practice (Days 15-21)**

### Level 13: Making Small Changes

- [ ] **Add a Simple Tool** 
  - [ ] Create new tool in `go_backend/internal/llm/tools/`
  - [ ] Register in `tools.go`
  - [ ] Test with CLI mode

- [ ] **Add UI Component**
  - [ ] Create component in `tauri_app/src/components/`
  - [ ] Use Radix UI primitives
  - [ ] Follow existing patterns

- [ ] **Modify API Endpoint**
  - [ ] Add handler in `go_backend/internal/http/`
  - [ ] Update routes in `server.go`
  - [ ] Test with frontend

### Level 14: Understanding Data Flows

- [ ] **Message Lifecycle**
  1. [ ] User types in chat input
  2. [ ] Frontend sends to `/api/sessions/{id}/messages`
  3. [ ] Backend processes with LLM provider
  4. [ ] Tools execute if needed
  5. [ ] Response streams via SSE
  6. [ ] Frontend updates UI

- [ ] **File Upload Flow**
  1. [ ] User drags file to chat
  2. [ ] Upload to `/api/sessions/{id}/files/upload`
  3. [ ] File stored in `storage/{session-id}/`
  4. [ ] Reference added to message
  5. [ ] Tools can access file

### Level 15: Debugging & Development

- [ ] **Common Debugging**
  - [ ] Check `dev.log` for all output
  - [ ] Use `make tail-log` to see recent logs
  - [ ] Frontend errors appear in terminal
  - [ ] Backend errors logged with context

- [ ] **Development Workflow**
  - [ ] Never stop `make dev` - it auto-reloads
  - [ ] Use `make frontend-typecheck` for TS errors
  - [ ] Run `make test-all` for environment validation

---

## **🎓 Graduation Criteria**

You'll know you're comfortable when you can:

- [ ] ✅ **Explain the architecture**: Describe how frontend talks to backend
- [ ] ✅ **Add a simple feature**: Create a new tool or UI component  
- [ ] ✅ **Debug issues**: Use logs and understand error patterns
- [ ] ✅ **Navigate codebase**: Find relevant files quickly
- [ ] ✅ **Understand data flow**: Trace a message from input to response

## **📚 Key Reference Files**

Always keep these handy:
- `CLAUDE.md` - Development guidelines
- `Makefile` - All available commands  
- `tauri_app/src/lib/cache-keys.ts` - Data caching patterns
- `go_backend/internal/db/models.go` - Database schema
- `tauri_app/src/types/` - TypeScript interfaces

---

## **📊 Progress Tracking**

**Current Phase**: [ ] Phase 1 [ ] Phase 2 [ ] Phase 3 [ ] Phase 4 [ ] Phase 5 [ ] Phase 6  
**Days Completed**: ___/21  
**Confidence Level**: [ ] Beginner [ ] Intermediate [ ] Advanced [ ] Expert

**Notes for Next Agent**:
_Add any insights, blockers, or important discoveries here..._

---

## **🚀 Quick Start for New Agents**

If you're a new agent picking up this plan:

1. **First Action**: Run `make dev` to start the development environment
2. **Check Status**: Review the checkboxes above to see current progress
3. **Start Learning**: Begin with the first unchecked item
4. **Update Progress**: Mark items ✅ as you complete them
5. **Add Notes**: Document any important findings at the bottom

**Emergency Commands**:
- `make tail-log` - See what's happening
- `make test-all` - Check if environment is working
- `make help` - See all available commands

**Key Insight**: This is a sophisticated AI coding assistant with Go backend + React frontend. The chat interface (📍 `tauri_app/src/components/chat-app.tsx:1056-1091`) handles streaming responses with tool calls and media showcase integration.