# Mix Web Demo Implementation Plan

## Overview

Create a minimal, hosted web version of Mix for public demos and quick trials. This will be a **separate web application** (`mix_web_demo/`) alongside the existing desktop app (`mix_dev_tool/`), sharing the same backend.

## Current Architecture Analysis

### Backend (mix_agent/)
**Status: ✅ No changes needed**

The backend is already properly architected:
- ✅ SSE streaming is **session-scoped** (not broadcast to all clients)
  - See `internal/http/sse.go:102` - `BroadcastEvent(sessionID string, ...)` filters by session
  - Connection registry maintains `map[sessionID][]*Connection`
- ✅ REST APIs already exist for all needed operations:
  - `POST /api/sessions` - Create session
  - `GET /api/sessions/{id}/messages` - Get messages
  - `POST /api/sessions/{id}/messages` - Send message
  - `GET /stream?sessionId={id}` - SSE streaming
- ✅ CORS is already configured (all routes have `Access-Control-Allow-Origin: *`)

**Only Global Broadcasts:**
- `session_created` and `session_deleted` events (for sidebar updates)
- These can be safely ignored in the web demo frontend

### Desktop App (mix_dev_tool/)
**Status: ✅ Keep unchanged - full featured local development tool**

Existing features to keep in desktop app:
- Settings dialog for provider authentication
- Model selection via command palette
- Session sidebar with history
- File uploads and attachments
- Full slash command support
- Session export functionality

## Web Demo Requirements

### Essential Features (Keep)
- ✅ Single chat interface
- ✅ Auto-created anonymous session on load
- ✅ Real-time SSE streaming (already session-specific)
- ✅ Message history display with tool calls
- ✅ Basic text input and submit
- ✅ Markdown rendering with code blocks
- ✅ Tool execution display (thinking, reasoning, etc.)

### Features to Remove
- ❌ Settings icon/dialog
- ❌ Model selection UI
- ❌ Slash commands and command palette
- ❌ Session sidebar/switching
- ❌ File uploads (optional - could add back if useful)
- ❌ Authentication (OAuth, API keys)
- ❌ Session export
- ❌ File reference system (`@file`)
- ❌ Plan mode toggle

## Implementation Plan

### 1. Project Structure

```
mix/
├── mix_agent/           # Backend (UNCHANGED)
├── mix_dev_tool/        # Desktop app (UNCHANGED)
└── mix_web_demo/        # NEW: Minimal web version
    ├── src/
    │   ├── components/
    │   │   ├── ChatInterface.tsx     # Main component (simplified ChatApp)
    │   │   ├── MessageDisplay.tsx    # Reuse ConversationDisplay
    │   │   ├── ChatInput.tsx         # Simplified input
    │   │   └── DemoBanner.tsx        # New: "Demo mode" banner
    │   ├── hooks/
    │   │   ├── useDemoSession.ts     # New: Auto-create session on load
    │   │   ├── useMessages.ts        # From mix_dev_tool/useSessionMessages
    │   │   └── useStreaming.ts       # From mix_dev_tool/usePersistentSSE
    │   ├── lib/
    │   │   ├── api-client.ts         # Mix SDK instance
    │   │   └── utils.ts              # Shared utilities
    │   ├── types/
    │   │   └── message.ts            # Type definitions
    │   └── main.tsx                  # Entry point
    ├── public/
    ├── index.html
    ├── package.json
    ├── tsconfig.json
    ├── vite.config.ts
    └── README.md
```

### 2. Components to Reuse (from mix_dev_tool)

**Copy with minimal changes:**
- `ConversationDisplay.tsx` → `MessageDisplay.tsx`
  - Remove edit message functionality
  - Remove plan mode actions
- `response-renderer.tsx` → Keep as is
- `todo-list.tsx` → Keep as is
- `ui/kibo-ui/ai/*` → Keep all AI components
- UI components: `button`, `card`, `scroll-area`, etc.

**Simplify heavily:**
- `ChatApp.tsx` → `ChatInterface.tsx`
  - Remove: settings, commands, file upload, sidebar integration
  - Keep: message display, input, SSE streaming
  - Remove: all slash command handling
  - Remove: file reference system

**Create new:**
- `DemoBanner.tsx` - Show demo limitations and usage info
- `ChatInput.tsx` - Simplified version without file upload/commands
- `useDemoSession.ts` - Auto-create session on mount

### 3. Hooks to Reuse/Adapt

**Reuse as-is:**
- `usePersistentSSE.ts` - Already session-scoped, perfect!
- `useSessionMessages.ts` - Fetch message history

**Simplify:**
- Remove: `useFileReference`, `useMessageHistoryNavigation`
- Remove: All command-related hooks

**Create new:**
```typescript
// useDemoSession.ts
export function useDemoSession() {
  const [sessionId, setSessionId] = useState<string | null>(null);
  const createSession = useCreateSession();

  useEffect(() => {
    // Auto-create session on mount
    const initSession = async () => {
      const session = await createSession.mutateAsync({
        title: `Demo Session ${new Date().toLocaleDateString()}`,
      });
      setSessionId(session.id);
      // Store in localStorage for page refresh
      localStorage.setItem('demo-session-id', session.id);
    };

    // Try to restore from localStorage first
    const existingSessionId = localStorage.getItem('demo-session-id');
    if (existingSessionId) {
      // Validate session still exists
      // If not, create new one
    } else {
      initSession();
    }
  }, []);

  return { sessionId, isReady: !!sessionId };
}
```

### 4. Frontend Implementation Details

#### Main App Component
```tsx
// src/main.tsx
function App() {
  const { sessionId, isReady } = useDemoSession();

  if (!isReady) {
    return <LoadingScreen />;
  }

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="dark">
        <DemoBanner />
        <ChatInterface sessionId={sessionId} />
      </ThemeProvider>
    </QueryClientProvider>
  );
}
```

#### Demo Banner
```tsx
// src/components/DemoBanner.tsx
export function DemoBanner() {
  return (
    <div className="bg-blue-500/10 border-b border-blue-500/20 p-2 text-center">
      <p className="text-sm text-blue-300">
        🎮 Demo Mode - This is a live demo session.
        <a href="https://github.com/your-repo" className="underline ml-2">
          Get the full app
        </a>
      </p>
    </div>
  );
}
```

#### Simplified Chat Interface
```tsx
// src/components/ChatInterface.tsx
export function ChatInterface({ sessionId }: { sessionId: string }) {
  const [text, setText] = useState("");
  const messages = useSessionMessages(sessionId);
  const sseStream = usePersistentSSE(sessionId);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!text.trim()) return;

    await sseStream.submitMessage({ text, attachments: [] });
    setText("");
  };

  return (
    <div className="flex h-screen flex-col">
      {/* Message Display */}
      <div className="flex-1 overflow-y-auto p-4">
        <MessageDisplay
          messages={messages.data || []}
          sseStream={sseStream}
        />
      </div>

      {/* Input */}
      <div className="border-t p-4">
        <form onSubmit={handleSubmit}>
          <div className="flex gap-2">
            <textarea
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="Ask me anything..."
              className="flex-1 rounded border p-2"
            />
            <button
              type="submit"
              disabled={!text.trim() || sseStream.processing}
            >
              Send
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
```

### 5. Configuration

#### Environment Variables
```bash
# .env
VITE_BACKEND_URL=http://localhost:8080  # For local dev
# VITE_BACKEND_URL=https://api.yourdomain.com  # For production
```

#### package.json
```json
{
  "name": "mix-web-demo",
  "version": "0.1.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^19.1.1",
    "react-dom": "^19.1.1",
    "@tanstack/react-query": "^5.89.0",
    "mix-typescript-sdk": "0.7.5",
    "react-markdown": "^10.1.0",
    "remark-gfm": "^4.0.1",
    "shiki": "^3.13.0",
    "lucide-react": "^0.540.0",
    "class-variance-authority": "^0.7.1",
    "clsx": "^2.1.1",
    "tailwind-merge": "^3.3.1"
  },
  "devDependencies": {
    "@vitejs/plugin-react": "^4.7.0",
    "typescript": "~5.6.3",
    "vite": "^6.3.6",
    "tailwindcss": "^4.1.13"
  }
}
```

### 6. Backend Integration (Optional)

If you want to serve the web demo from the same backend:

```go
// In mix_agent/internal/http/server.go

// Serve static web demo files
mux.Handle("/demo/", http.StripPrefix("/demo/",
  http.FileServer(http.Dir("../mix_web_demo/dist"))))

// Fallback for SPA routing (if using client-side routing)
mux.HandleFunc("/demo", func(w http.ResponseWriter, r *http.Request) {
  http.ServeFile(w, r, "../mix_web_demo/dist/index.html")
})
```

Then build and serve:
```bash
cd mix_web_demo
npm run build

cd ../mix_agent
go run main.go
# Access at http://localhost:8080/demo
```

### 7. Deployment Options

#### Option A: Same Backend (Recommended)
- Build frontend: `cd mix_web_demo && npm run build`
- Serve from Go backend at `/demo` route
- Single deployment, shared backend

#### Option B: Separate Deployment
- Frontend: Deploy to Vercel/Netlify/Cloudflare Pages
- Backend: Keep at existing location
- Frontend connects via `VITE_BACKEND_URL`

#### Option C: Docker Compose
```yaml
version: '3.8'
services:
  backend:
    build: ./mix_agent
    ports:
      - "8080:8080"

  web-demo:
    build: ./mix_web_demo
    ports:
      - "3000:3000"
    environment:
      - VITE_BACKEND_URL=http://backend:8080
```

## Implementation Phases

### Phase 1: Setup & Copy Components (1-2 hours)
- [ ] Create `mix_web_demo/` directory structure
- [ ] Setup Vite + React + TypeScript
- [ ] Copy and simplify UI components from mix_dev_tool
- [ ] Setup Tailwind CSS
- [ ] Configure `mix-typescript-sdk`

### Phase 2: Core Functionality (2-3 hours)
- [ ] Implement `useDemoSession` hook
- [ ] Create `ChatInterface` component (simplified ChatApp)
- [ ] Integrate `usePersistentSSE` for streaming
- [ ] Add message display (reuse ConversationDisplay)
- [ ] Implement basic input/submit

### Phase 3: Polish & Testing (1-2 hours)
- [ ] Add `DemoBanner` component
- [ ] Style and responsiveness
- [ ] Test with local backend
- [ ] Handle edge cases (no backend, disconnection, errors)

### Phase 4: Deployment (1 hour)
- [ ] Build production bundle
- [ ] Configure backend static file serving (if needed)
- [ ] Deploy and test live

**Total Estimated Time: 5-8 hours**

## Testing Checklist

### Functional Tests
- [ ] Auto-create session on page load
- [ ] Send message and receive streaming response
- [ ] Display thinking/reasoning/tool calls correctly
- [ ] Multiple browser tabs don't interfere (session isolation)
- [ ] Page refresh restores session from localStorage
- [ ] Handle backend disconnection gracefully
- [ ] Handle message cancellation

### Browser Compatibility
- [ ] Chrome/Edge (latest)
- [ ] Firefox (latest)
- [ ] Safari (latest)
- [ ] Mobile browsers (iOS Safari, Chrome Android)

### Performance
- [ ] SSE connection stays alive during streaming
- [ ] No memory leaks on long sessions
- [ ] Smooth scrolling with many messages

## Security Considerations

Since this is a public demo without authentication:

1. **Rate Limiting** (Future Enhancement)
   - Backend should implement IP-based rate limiting
   - E.g., 10 messages per hour per IP
   - Return 429 status when limit exceeded

2. **Session Cleanup**
   - Demo sessions should auto-expire after 1 hour of inactivity
   - Implement cleanup job in backend

3. **Tool Restrictions** (Future Enhancement)
   - Consider disabling dangerous tools (file system access, bash)
   - Or run in sandboxed environment

4. **Data Privacy**
   - Demo sessions should not persist sensitive data
   - Consider purging demo session data periodically

## Future Enhancements (Post-MVP)

### Phase 2 Features
- [ ] Session persistence (store session ID in URL)
- [ ] Share session via link
- [ ] Rate limiting indicator (X/10 messages used)
- [ ] Add basic file upload for demos
- [ ] "Try on Desktop" CTA button

### Phase 3 Features
- [ ] Analytics tracking (PostHog)
- [ ] Demo session gallery (example prompts)
- [ ] Onboarding tour/tooltips

## Key Differences: mix_dev_tool vs mix_web_demo

| Feature | mix_dev_tool (Desktop) | mix_web_demo (Web) |
|---------|------------------------|---------------------|
| Architecture | Tauri (Rust + React) | Pure React SPA |
| Authentication | Full OAuth + API keys | None (anonymous) |
| Sessions | Multiple, managed | Single, auto-created |
| Model Selection | Yes, via commands | No (backend default) |
| File Upload | Yes | No (MVP) |
| Commands | Full slash commands | None |
| Settings | Full settings dialog | None |
| Sidebar | Session history | None |
| Local Backend | Required | Remote backend |
| Use Case | Development, power users | Quick demos, trials |

## Success Criteria

A successful web demo implementation should:
- ✅ Load and create session in < 2 seconds
- ✅ Display streaming responses in real-time
- ✅ Work on all major browsers (Chrome, Firefox, Safari, Edge)
- ✅ Mobile-responsive (works on phones/tablets)
- ✅ Handle 100+ concurrent users without issues
- ✅ Zero configuration for end users (just open URL)
- ✅ Clear CTA to download/install full desktop app

## Non-Goals (Out of Scope)

- ❌ User accounts or authentication
- ❌ Persistent session history across devices
- ❌ Advanced features (file uploads, custom models, etc.)
- ❌ Offline support
- ❌ Desktop app parity (this is intentionally minimal)

## Conclusion

This plan creates a minimal, focused web demo by:
1. **Reusing existing backend** (no changes needed - SSE already session-scoped)
2. **Copying and simplifying frontend components** from mix_dev_tool
3. **Removing complex features** (auth, commands, settings, sidebar)
4. **Auto-creating anonymous sessions** for zero-config UX

The result is a lightweight web app perfect for quick demos and trials, while keeping the full-featured desktop app unchanged for power users.
