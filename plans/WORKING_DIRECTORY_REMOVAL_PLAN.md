# Working Directory Removal Plan

## Overview

Remove the complex arbitrary per-session working directory concept and replace with a predictable centralized file storage system using session subfolders. This eliminates directory creation complexity while maintaining session isolation and all required functionality.

## Final Architecture Decisions

- **Session-scoped storage**: Each session gets subfolder `/storage/{session-id}/`
- **Organized storage structure**: Session subfolders for isolation and organization
- **No database tracking**: Filesystem-based file management with session folders
- **Tools work in session folders**: Tools operate in their session's storage subfolder
- **No backward compatibility**: Clean removal of legacy working directory concept
- **Directory scanning**: Within session folders for file discovery
- **Preserve existing features**: Keep thumbnail generation and GSAP animations separate

## Implementation Plan

### Phase 1: Database & Model Cleanup

#### 1.1 Database Schema Changes

- [ ] Create migration to DROP `working_directory` column from `sessions` table
- [ ] Remove `WorkingDirectory` field from all Go structs:
  - `Session` struct in `/go_backend/internal/session/session.go`
  - `SessionData` in `/go_backend/internal/http/rest_sessions.go`
  - All database model structs in `/go_backend/internal/db/models.go`
- [ ] Remove `WorkingDirectory` from all SQLC queries in `/go_backend/internal/db/sql/sessions.sql`
- [ ] Regenerate SQLC code

#### 1.2 Session Management Cleanup

- [ ] Remove directory creation logic from `session.Create()` in `/go_backend/internal/session/session.go:64-96`
- [ ] Remove `validateWorkingDirectory()` function and all calls to it
- [ ] Remove working directory validation from session conversion methods
- [ ] Simplify session forking to not inherit working directory

### Phase 2: Centralized Storage System

#### 2.1 Storage Directory Setup

- [ ] Create centralized storage initialization in app startup
- [ ] Create `/storage/` directory in launch directory during app initialization
- [ ] Add storage path configuration to config system

#### 2.2 Session Storage Directory Creation

- [ ] **Replace working directory creation with session folder creation**:
  - During session creation, create `/storage/{session-id}/` directory
  - No subdirectories needed (no input/, output/, gsap_animations/)
  - Simple flat structure within each session folder
- [ ] Update session creation logic to use storage directories instead of working directories
- [ ] Remove all the complex directory structure creation (input/images/, input/videos/, etc.)

#### 2.3 Storage Path Resolution

- [ ] Add utility functions to resolve session storage paths:
  - `GetSessionStoragePath(sessionID string) string` → `/storage/{session-id}/`  
  - `ValidateSessionID(sessionID string) bool` → UUID format validation
  - Security validation to prevent path traversal attacks

#### 2.4 File Upload API

Create new session-aware file management endpoints:

```go
// New endpoints to implement:
POST   /api/sessions/{id}/files/upload     - Upload files to session storage
GET    /api/sessions/{id}/files           - List files in session folder
GET    /api/sessions/{id}/files/{filename} - Serve specific file from session
DELETE /api/sessions/{id}/files/{filename} - Delete file from session
```

**Upload Implementation**:

- Store files directly in `/storage/{session-id}/uploaded-file.txt`
- No UUID prefixing needed (session folder provides isolation)
- Return file info and session-relative path
- Support multiple file uploads per session

**File Listing Implementation**:

- Scan `/storage/{session-id}/` directory for specific session
- Return file metadata (name, size, modified date) for that session only
- Optional caching for performance

### Phase 3: Complete Asset Server Rewrite

#### 3.1 New Stateless Asset Server Design

- [ ] **Complete rewrite** of asset server to be stateless and session-aware
- [ ] Remove all working directory state (`currentWorkDir`, mutex, `SetWorkingDirectory()`)
- [ ] Remove `WORKING_DIR` environment variable dependency entirely
- [ ] New architecture: Parse session ID from URL path and serve from session folder

#### 3.2 New Session-Based File Serving

- [ ] Replace `ServeHTTP()` with session-aware routing:
  - Parse session ID from `/api/sessions/{session-id}/files/{filename}` URLs
  - Serve files from `/storage/{session-id}/{filename}` path
  - Return 404 if session folder doesn't exist
- [ ] Remove all input/output distinction logic completely
- [ ] Implement path traversal security (prevent `../` attacks)
- [ ] Validate session ID format and file path safety

#### 3.3 Thumbnail & GSAP Integration

- [ ] **Keep thumbnail generation within asset server** - it should generate thumbnails for session files
- [ ] Update thumbnail logic to work with session-based file paths (`/storage/{session-id}/{filename}`)  
- [ ] Store thumbnails in session folders (e.g., `/storage/{session-id}/.thumbnails/`)
- [ ] Keep GSAP animations in separate global directory with separate serving logic
- [ ] Asset server handles both session files AND their thumbnails

### Phase 4: Tool System Updates

#### 4.1 Context Changes

- [ ] Keep `WorkingDirectoryContextKey` but rename to `SessionStorageDirectoryContextKey`
- [ ] Update `GetWorkingDirectory(ctx)` to `GetSessionStorageDirectory(ctx)`
- [ ] Context now provides `/storage/{session-id}/` path instead of arbitrary working directory

#### 4.2 Tool Behavior Updates

**File Path Resolution Strategy**: Tools work directly in session storage folders

- [ ] **Write Tool**: Write files directly to `/storage/{session-id}/filename`
- [ ] **Edit Tool**: Edit files in session storage directory
- [ ] **LS Tool**: List session's storage directory (`/storage/{session-id}/`)
- [ ] **Bash Tool**: Run in session's storage directory as working directory
- [ ] **Grep/Glob Tools**: Search within session's storage directory

#### 4.3 Agent Context Updates

- [ ] Update working directory context in agent to use session storage directory
- [ ] Modify `/go_backend/internal/llm/agent/agent.go:193,434,847` to set session storage path
- [ ] Tool execution context gets session's `/storage/{session-id}/` path

### Phase 5: Complete HTTP API Overhaul

#### 5.1 **BREAKING: Remove All Legacy Endpoints**

- [ ] **Immediately remove** `/input/*` and `/output/*` handlers (no backward compatibility)
- [ ] Remove `HandleInputAssets()` and `HandleOutputAssets()` functions completely
- [ ] Remove all asset-related routes from server.go
- [ ] **This will break frontend until Phase 6 is complete**

#### 5.2 New URL Structure Implementation

- [ ] **Replace asset serving with new session-based routing**:

  ```
  OLD: /input/video.mp4, /output/image.jpg
  NEW: /api/sessions/{session-id}/files/{filename}
  ```

- [ ] Implement session ID extraction from URL path
- [ ] Route to new stateless asset server with session context

#### 5.3 Session API Cleanup  

- [ ] Remove `sessionStorageDirectory` field from all session API responses
- [ ] Remove `WorkingDirectory` from `CreateSessionRequest` struct
- [ ] Update session creation to not require working directory parameter

#### 5.4 New Session File Management Endpoints

- [ ] `POST /api/sessions/{id}/files/upload` - Upload files with multipart form
- [ ] `GET /api/sessions/{id}/files` - List files in session storage folder
- [ ] `GET /api/sessions/{id}/files/{filename}` - Serve file from session storage  
- [ ] `DELETE /api/sessions/{id}/files/{filename}` - Delete file from session
- [ ] All endpoints validate session ID and prevent path traversal

### Phase 6: **BREAKING: Complete Frontend Rewrite**

#### 6.1 **Asset URL System Complete Overhaul**

- [ ] **Complete rewrite** of `convertToAssetServerUrl()` in `/tauri_app/src/utils/assetServer.ts:8-19`:
  - **OLD**: `convertToAssetServerUrl(absolutePath: string, sessionStorageDirectory: string)`
  - **NEW**: `convertToAssetServerUrl(filename: string, sessionId: string)`
  - **NEW URL FORMAT**: `${getBackendUrl()}/api/sessions/${sessionId}/files/${filename}`
  - **Remove**: Working directory validation and path stripping logic
  - **Add**: Session ID validation and filename sanitization
- [ ] **Update helper functions** that depend on `convertToAssetServerUrl()`:
  - `getMediaSrc()` in `/tauri_app/src/components/playlist-sidebar.tsx:56`
  - `generatePreviewUrl()` calls throughout codebase

#### 6.2 TypeScript Types Breaking Changes

- [ ] **Remove `sessionStorageDirectory` field** from these exact interfaces:
  - `Session` interface in `/tauri_app/src/types/common.ts:13-17`
  - `SessionData` interface in `/tauri_app/src/types/common.ts:19-34`
  - `VideoPlayerProps` interface in `/tauri_app/src/types/media.ts:12-19`
  - `CreateSessionParams` interface in `/tauri_app/src/hooks/useSession.ts:6-9`
- [ ] **Update session creation** in `useCreateSession()` hook to not pass `sessionStorageDirectory`
- [ ] **Add session context** to media components that need file access

#### 6.3 Critical Component Updates (20+ files)

**Major components requiring sessionStorageDirectory parameter removal:**

- [ ] **ConversationDisplay** (`/tauri_app/src/components/conversation-display.tsx`):
  - Remove `sessionStorageDirectory` prop from lines: 11, 19, 69, 91, 98, 114, 134, 161, 169, 175, 180, 195, 403-404, 429
  - Replace with `sessionId` prop and use session-based URLs
  
- [ ] **ChatApp** (`/tauri_app/src/components/chat-app.tsx`):
  - Remove `session?.sessionStorageDirectory` references on lines: 192, 529, 650, 660, 666, 729, 736
  - Pass `session.id` instead to child components
  
- [ ] **PlaylistSidebar** (`/tauri_app/src/components/playlist-sidebar.tsx`):
  - Remove `sessionStorageDirectory` parameter from lines: 17-18, 56, 63, 81, 95, 108, 129
  - Update `getMediaSrc()` calls to use session ID
  
- [ ] **VideoPlayer** (`/tauri_app/src/components/video-player.tsx`):
  - Remove `sessionStorageDirectory` from lines: 21, 83
  - Use session-based URL generation
  
- [ ] **CommandFileReference** (`/tauri_app/src/components/command-file-reference.tsx`):
  - Remove `sessionStorageDirectory` from lines: 35, 41, 45, 50, 119, 312
  - Update thumbnail generation for session-based storage
  
- [ ] **MessageAttachmentDisplay** (`/tauri_app/src/components/message-attachment-display.tsx`):
  - Remove `sessionStorageDirectory` from lines: 15, 20, 32, 37
  - Use session ID for attachment URLs
  
- [ ] **AttachmentPreview** (`/tauri_app/src/components/attachment-preview.tsx`):
  - Remove `sessionStorageDirectory` from lines: 25, 33, 98, 100
  - Update preview URL generation

#### 6.4 New File Management UI Implementation

- [ ] **Update existing file upload system**:
  - Modify attachment store (`/tauri_app/src/stores/attachmentSlice.ts`) to use session-scoped uploads
  - Update `createFileAttachment()` in `/tauri_app/src/utils/attachmentUtils.ts` for session context
  
- [ ] **Enhance file browsing UI**:
  - Update `FileReferencePopup` (`/tauri_app/src/components/file-reference-popup.tsx`) to browse session files via API
  - Modify `useFileReference` hook (`/tauri_app/src/hooks/useFileReference.ts`) to call session file APIs
  
- [ ] **Update @ file reference resolution**:
  - Modify file discovery to use `GET /api/sessions/{id}/files` API instead of filesystem access
  - Update `@filename` matching logic to work with session file listings
  - Ensure file reference autocomplete works with session-scoped files
  - Update file reference validation to check session file existence
  
- [ ] **Add file management actions**:
  - Implement file deletion UI using `DELETE /api/sessions/{id}/files/{filename}`
  - Add file upload progress indicators for `POST /api/sessions/{id}/files/upload`
  - Create file listing UI using `GET /api/sessions/{id}/files`

#### 6.5 Session Management Updates  

- [ ] **Update session hooks** (`/tauri_app/src/hooks/useSession.ts`):
  - Remove `sessionStorageDirectory` from `CreateSessionParams`
  - Update `useActiveSession()` to not expect `sessionStorageDirectory` in response
  - Ensure session ID is properly passed to file operations

#### 6.6 Asset Reference Updates

**Complete audit and replacement of asset URL generation:**

- [ ] **Thumbnail generation**: Update all `?thumb=200` URL generation to use session-based endpoints
- [ ] **Video thumbnails**: Update time-based thumbnails to use session endpoints with `?time=` parameter  
- [ ] **Media previews**: Replace all media preview URLs to use `/api/sessions/{sessionId}/files/{filename}` format
- [ ] **File attachments**: Update @ file reference system to resolve to session-based URLs
- [ ] **GSAP animations**: Ensure GSAP animations continue using global endpoints (no session context needed)

#### 6.7 Error Handling & Validation

- [ ] **Add session validation**: Ensure all file operations validate session ID format
- [ ] **Update error messages**: Replace working directory error messages with session-based errors
- [ ] **Add loading states**: Implement loading indicators for session file operations
- [ ] **Handle file not found**: Update 404 handling for session-scoped file access

### Phase 7: Configuration & Environment

#### 7.1 Environment Variables

- [ ] Remove `WORKING_DIR` environment variable dependency
- [ ] Add `STORAGE_DIR` configuration (defaulting to `{launch_dir}/storage/`)
- [ ] Update documentation and configuration examples

#### 7.2 App Initialization

- [ ] Add storage directory creation to app startup
- [ ] Remove working directory validation from config loading
- [ ] Update asset server initialization to use storage directory

### Phase 8: Testing & Cleanup

#### 8.1 Update Tests

- [ ] Remove all working directory-related test fixtures
- [ ] Update integration tests to use new file storage API
- [ ] Update tool tests to work with centralized storage
- [ ] Add tests for new file upload/management functionality

#### 8.2 Documentation Cleanup

- [ ] Remove all references to working directories from documentation
- [ ] Update API documentation for new file endpoints
- [ ] Update README and setup instructions

#### 8.3 Code Cleanup

- [ ] Remove unused imports and constants related to working directories
- [ ] Clean up any remaining references to session-specific directories
- [ ] Remove legacy code and comments

## Implementation Order

⚠️ **WARNING: This implementation will completely break the application until all phases are complete. No incremental rollout possible.**

### **Critical Implementation Sequence**

1. **Database/Models** (Phase 1) - Remove working directory from all database operations
2. **Storage System** (Phase 2) - Implement session folder creation and utilities  
3. **Asset Server Complete Rewrite** (Phase 3) - **APPLICATION BREAKS HERE** until Phase 6 complete
4. **Tools Update** (Phase 4) - Update tools to use session storage folders
5. **HTTP API Complete Overhaul** (Phase 5) - Remove all legacy endpoints, add session-based APIs
6. **Frontend Complete Rewrite** (Phase 6) - **APPLICATION WORKS AGAIN** after this phase
7. **Configuration** (Phase 7) - Clean up environment variables
8. **Testing & Cleanup** (Phase 8) - Validation and polish

### **Development Strategy**

- **Feature branch required** - this cannot be done on main branch
- **Complete all phases** before merging - no partial implementation possible
- **Expect application to be completely broken** from Phase 3 through Phase 6
- **Frontend and backend must be updated together** in single deployment

## Key Benefits

- **Simplified Architecture**: Predictable session storage folders instead of arbitrary working directories
- **Reduced Complexity**: Organized storage structure, no random directory creation
- **Session Isolation**: Each session has its own folder while maintaining organization
- **Easier Deployment**: Predictable storage layout, no cleanup of arbitrary directories
- **Tool Compatibility**: Tools continue working in session context with organized storage
- **Preserved Features**: Thumbnails and GSAP animations remain unchanged

## Risk Mitigation

- **File Conflicts**: Session subfolders provide isolation, no naming conflicts between sessions
- **Security**: Path validation prevents traversal outside session folders
- **Performance**: Session-specific directory scanning, optional caching for file listing
- **Tool Compatibility**: Session storage directory maintains all current tool functionality
- **Migration**: Clean removal with session-based organization

## Critical Issues Addressed in This Revision

1. **Asset Server Architecture**: Complete rewrite to stateless, session-aware design instead of trying to modify stateful working directory approach
2. **Session Storage Creation**: Explicit session folder creation during session creation to prevent tool failures
3. **URL Routing Logic**: Clear session ID extraction from URLs with proper routing to session folders
4. **Frontend Breaking Changes**: Acknowledged complete breakage and planned complete rewrite instead of assuming compatibility
5. **No Partial Implementation**: Planned as complete overhaul requiring feature branch and coordinated deployment

## Post-Implementation Verification

- [ ] All tools work correctly with session storage folders (`/storage/{session-id}/`)
- [ ] New asset server correctly parses session IDs from URLs and serves from session folders
- [ ] Session file upload, list, serve, delete operations work end-to-end
- [ ] No references to arbitrary working directories remain in codebase  
- [ ] Frontend completely rewritten to use session-based file APIs
- [ ] Asset serving works for all file types within session folders
- [ ] Thumbnails integrated into session-based asset server, GSAP animations continue in separate system
- [ ] Each session operates in its own isolated `/storage/{session-id}/` folder
- [ ] Session storage directories created automatically during session creation
- [ ] Path traversal security prevents access outside session folders
