# Mix Web Demo - Refined Implementation Plan

## Overview

Create a minimal web version of Mix that **looks and works exactly like the desktop app**, but without multiple sessions, settings, and slash commands. This is a pure React web app (not Tauri) that shares the same backend.

## Architecture Analysis

### Backend (mix_agent/) - ✅ NO CHANGES NEEDED
- SSE streaming is already session-scoped
- REST APIs are all ready
- CORS is configured
- Backend is perfect as-is!

### Current Dev Tool (mix_dev_tool/) - ✅ KEEP UNCHANGED
- Tauri desktop app
- Full-featured local development tool
- Will remain unchanged

### New Web Demo (mix_web_demo/) - 🆕 NEW APPLICATION
- Pure React web app (Vite + React + TypeScript)
- Looks identical to dev tool
- Single session, auto-created
- No sidebar, no settings, no slash commands
- Everything else identical (SSE, tool display, message rendering)

## Key Differences: Dev Tool vs Web Demo

| Feature | mix_dev_tool (Desktop) | mix_web_demo (Web) |
|---------|------------------------|---------------------|
| **Platform** | Tauri (Rust + React) | Pure React SPA |
| **Router** | TanStack Router | Simple React (no routing) |
| **Sessions** | Multiple, with sidebar | Single, auto-created |
| **Settings** | Full settings dialog | None |
| **Slash Commands** | Yes | No |
| **File Upload** | Yes | Yes (can keep) |
| **SSE Streaming** | Yes | Yes (identical) |
| **Message Display** | Full featured | Identical |
| **Tool Execution** | Full featured | Identical |
| **Plan Mode** | Yes | Yes (can keep) |
| **Edit Message** | Yes | Yes (can keep) |

## Project Structure

```
mix/
├── mix_agent/           # Backend (UNCHANGED)
├── mix_dev_tool/        # Desktop app (UNCHANGED)
└── mix_web_demo/        # 🆕 NEW: Web version
    ├── src/
    │   ├── components/
    │   │   ├── chat-interface.tsx       # Simplified chat-app.tsx (no routing, no sidebar)
    │   │   ├── conversation-display.tsx # COPIED from dev tool (no changes)
    │   │   ├── response-renderer.tsx    # COPIED from dev tool
    │   │   ├── todo-list.tsx            # COPIED from dev tool
    │   │   ├── plan-display.tsx         # COPIED from dev tool
    │   │   ├── rate-limit-display.tsx   # COPIED from dev tool
    │   │   ├── model-display.tsx        # COPIED from dev tool
    │   │   ├── provider-display.tsx     # COPIED from dev tool
    │   │   ├── media-showcase.tsx       # COPIED from dev tool
    │   │   ├── message-attachment-display.tsx # COPIED from dev tool
    │   │   ├── conversation-loader.tsx  # COPIED from dev tool
    │   │   ├── attachment-preview.tsx   # COPIED from dev tool
    │   │   ├── file-upload-button.tsx   # COPIED from dev tool (optional)
    │   │   ├── permission-dialog.tsx    # COPIED from dev tool
    │   │   ├── demo-banner.tsx          # 🆕 NEW: Demo mode indicator
    │   │   └── ui/                      # COPIED from dev tool (all UI components)
    │   │       ├── button.tsx
    │   │       ├── card.tsx
    │   │       ├── scroll-area.tsx
    │   │       └── kibo-ui/             # COPIED from dev tool (all AI components)
    │   │           └── ai/
    │   │               ├── message.tsx
    │   │               ├── input.tsx
    │   │               ├── reasoning.tsx
    │   │               ├── tool.tsx
    │   │               └── ...
    │   ├── hooks/
    │   │   ├── usePersistentSSE.ts      # COPIED from dev tool (no changes)
    │   │   ├── useSessionMessages.ts    # COPIED from dev tool (no changes)
    │   │   ├── useSession.ts            # COPIED from dev tool (only useActiveSession)
    │   │   ├── useFileUpload.ts         # COPIED from dev tool (optional)
    │   │   ├── useFileReference.ts      # COPIED from dev tool
    │   │   ├── useRewindSession.ts      # COPIED from dev tool
    │   │   ├── usePreferences.ts        # COPIED from dev tool
    │   │   ├── useSessionExport.ts      # COPIED from dev tool (optional)
    │   │   ├── useMessageHistory.ts     # COPIED from dev tool
    │   │   ├── useMessageHistoryNavigation.ts # COPIED from dev tool
    │   │   ├── use-copy-to-clipboard.ts # COPIED from dev tool
    │   │   └── useDemoSession.ts        # 🆕 NEW: Auto-create session on load
    │   ├── lib/
    │   │   ├── mix-sdk.ts               # COPIED from dev tool
    │   │   ├── cache-keys.ts            # COPIED from dev tool
    │   │   └── utils.ts                 # COPIED from dev tool
    │   ├── stores/
    │   │   ├── index.ts                 # COPIED from dev tool
    │   │   └── attachmentSlice.ts       # COPIED from dev tool
    │   ├── types/
    │   │   ├── common.ts                # COPIED from dev tool
    │   │   ├── message.ts               # COPIED from dev tool
    │   │   └── media.ts                 # COPIED from dev tool
    │   ├── utils/
    │   │   ├── attachmentUtils.ts       # COPIED from dev tool
    │   │   ├── sessionUtils.ts          # COPIED from dev tool
    │   │   ├── assetServer.ts           # COPIED from dev tool
    │   │   ├── videoUrlDetection.ts     # COPIED from dev tool
    │   │   ├── fileTypes.ts             # COPIED from dev tool
    │   │   ├── backendUrl.ts            # COPIED from dev tool (modify for env var)
    │   │   └── platform.ts              # COPIED from dev tool (modify for web)
    │   ├── styles/
    │   │   └── App.css                  # COPIED from dev tool
    │   ├── App.tsx                      # 🆕 NEW: Main app component
    │   └── main.tsx                     # 🆕 NEW: Entry point
    ├── public/
    ├── index.html
    ├── package.json
    ├── tsconfig.json
    ├── vite.config.ts
    ├── tailwind.config.js               # COPIED from dev tool
    ├── postcss.config.js                # COPIED from dev tool
    └── README.md
```

## Implementation Steps

### Phase 1: Project Setup (30 minutes)

#### 1.1 Create Project Directory
```bash
mkdir mix_web_demo
cd mix_web_demo
```

#### 1.2 Initialize Vite Project
```bash
npm create vite@latest . -- --template react-ts
```

#### 1.3 Install Dependencies
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
    "tailwind-merge": "^3.3.1",
    "sonner": "^1.7.2",
    "zustand": "^5.0.5"
  },
  "devDependencies": {
    "@vitejs/plugin-react": "^4.7.0",
    "typescript": "~5.6.3",
    "vite": "^6.3.6",
    "tailwindcss": "^4.1.13",
    "@types/react": "^19.0.0",
    "@types/react-dom": "^19.0.0"
  }
}
```

#### 1.4 Copy Configuration Files
```bash
# Copy from mix_dev_tool
cp ../mix_dev_tool/tailwind.config.js .
cp ../mix_dev_tool/postcss.config.js .
cp ../mix_dev_tool/tsconfig.json .
# Modify paths in tsconfig.json to match new structure
```

#### 1.5 Setup Environment Variables
```bash
# .env
VITE_BACKEND_URL=http://localhost:8080

# .env.production
VITE_BACKEND_URL=https://your-production-backend.com
```

### Phase 2: Copy Core Components (1 hour)

#### 2.1 Copy UI Components Directory
```bash
cp -r ../mix_dev_tool/src/components/ui src/components/
```

This includes:
- All shadcn/ui components (button, card, dialog, etc.)
- All kibo-ui/ai components (message, input, reasoning, tool, etc.)
- Theme provider

#### 2.2 Copy Message Display Components
```bash
# Copy these files directly (no modifications needed):
cp ../mix_dev_tool/src/components/conversation-display.tsx src/components/
cp ../mix_dev_tool/src/components/response-renderer.tsx src/components/
cp ../mix_dev_tool/src/components/todo-list.tsx src/components/
cp ../mix_dev_tool/src/components/plan-display.tsx src/components/
cp ../mix_dev_tool/src/components/rate-limit-display.tsx src/components/
cp ../mix_dev_tool/src/components/model-display.tsx src/components/
cp ../mix_dev_tool/src/components/provider-display.tsx src/components/
cp ../mix_dev_tool/src/components/media-showcase.tsx src/components/
cp ../mix_dev_tool/src/components/message-attachment-display.tsx src/components/
cp ../mix_dev_tool/src/components/conversation-loader.tsx src/components/
cp ../mix_dev_tool/src/components/attachment-preview.tsx src/components/
cp ../mix_dev_tool/src/components/permission-dialog.tsx src/components/
cp ../mix_dev_tool/src/components/file-upload-button.tsx src/components/
cp ../mix_dev_tool/src/components/status-ui.tsx src/components/
```

#### 2.3 Copy Additional Components
```bash
cp ../mix_dev_tool/src/components/CsvViewer.tsx src/components/
cp ../mix_dev_tool/src/components/LazyVideoPlayer.tsx src/components/
cp ../mix_dev_tool/src/components/video-player.tsx src/components/
cp ../mix_dev_tool/src/components/audio-waveform.tsx src/components/
cp ../mix_dev_tool/src/components/attachment-item-preview.tsx src/components/
```

### Phase 3: Copy Hooks and Utilities (30 minutes)

#### 3.1 Copy Hooks
```bash
# Core hooks (no modifications):
cp ../mix_dev_tool/src/hooks/usePersistentSSE.ts src/hooks/
cp ../mix_dev_tool/src/hooks/useSessionMessages.ts src/hooks/
cp ../mix_dev_tool/src/hooks/useFileUpload.ts src/hooks/
cp ../mix_dev_tool/src/hooks/useFileReference.ts src/hooks/
cp ../mix_dev_tool/src/hooks/useRewindSession.ts src/hooks/
cp ../mix_dev_tool/src/hooks/usePreferences.ts src/hooks/
cp ../mix_dev_tool/src/hooks/useSessionExport.ts src/hooks/
cp ../mix_dev_tool/src/hooks/useMessageHistory.ts src/hooks/
cp ../mix_dev_tool/src/hooks/useMessageHistoryNavigation.ts src/hooks/
cp ../mix_dev_tool/src/hooks/use-copy-to-clipboard.ts src/hooks/
cp ../mix_dev_tool/src/hooks/useCsv.ts src/hooks/

# Extract only useActiveSession and useCreateSession from useSession.ts
# (Remove useSessionsList and other multi-session hooks)
```

#### 3.2 Copy Utilities
```bash
cp -r ../mix_dev_tool/src/utils src/
# Modify backendUrl.ts to use VITE_BACKEND_URL env var
# Modify platform.ts to always return 'web'
```

#### 3.3 Copy Types, Stores, and Lib
```bash
cp -r ../mix_dev_tool/src/types src/
cp -r ../mix_dev_tool/src/stores src/
cp -r ../mix_dev_tool/src/lib src/
cp -r ../mix_dev_tool/src/styles src/
```

### Phase 4: Create New Components (1 hour)

#### 4.1 Create Demo Banner Component
```tsx
// src/components/demo-banner.tsx
export function DemoBanner() {
  return (
    <div className="border-b border-blue-500/20 bg-blue-500/10 p-3 text-center">
      <p className="text-blue-300 text-sm">
        🎮 Demo Mode - Exploring Mix in your browser.{" "}
        <a
          href="https://github.com/recreate-run/mix"
          className="underline hover:text-blue-200"
          target="_blank"
          rel="noopener noreferrer"
        >
          Get the full desktop app
        </a>
      </p>
    </div>
  );
}
```

#### 4.2 Create useDemoSession Hook
```tsx
// src/hooks/useDemoSession.ts
import { useEffect, useState } from "react";
import { useCreateSession } from "./useSession";
import { mix } from "@/lib/mix-sdk";

const DEMO_SESSION_KEY = "mix-demo-session-id";

export function useDemoSession() {
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [isReady, setIsReady] = useState(false);
  const createSession = useCreateSession();

  useEffect(() => {
    const initSession = async () => {
      // Try to restore from localStorage
      const existingSessionId = localStorage.getItem(DEMO_SESSION_KEY);

      if (existingSessionId) {
        try {
          // Validate session still exists
          const session = await mix.sessions.get({ id: existingSessionId });
          if (session) {
            setSessionId(existingSessionId);
            setIsReady(true);
            return;
          }
        } catch (error) {
          // Session doesn't exist, create new one
          localStorage.removeItem(DEMO_SESSION_KEY);
        }
      }

      // Create new session
      try {
        const newSession = await createSession.mutateAsync({
          title: `Demo Session - ${new Date().toLocaleDateString()}`,
        });
        setSessionId(newSession.id);
        localStorage.setItem(DEMO_SESSION_KEY, newSession.id);
        setIsReady(true);
      } catch (error) {
        console.error("Failed to create demo session:", error);
      }
    };

    initSession();
  }, [createSession]);

  return { sessionId, isReady };
}
```

#### 4.3 Create ChatInterface Component
```tsx
// src/components/chat-interface.tsx
// This is a simplified version of chat-app.tsx with:
// - No routing imports
// - No CommandSlash component
// - No slash command handling
// - No sidebar integration
// - Removed all slash command state and handlers
// - Keep: SSE streaming, file upload, message display, edit, export
//
// Copy the entire chat-app.tsx and remove:
// 1. All imports related to TanStack Router
// 2. All CommandSlash related code
// 3. All slash command state (showSlashCommands, selectedCommandIndex, showCommands)
// 4. All slash command handlers (handleSlashCommandNavigation, shouldShowSlashCommands)
// 5. CommandSlash component from render
// 6. Slash command keyboard handling
//
// Keep everything else identical!
```

#### 4.4 Create Main App Component
```tsx
// src/App.tsx
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "@/components/ui/theme-provider";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Toaster } from "sonner";
import { ChatInterface } from "@/components/chat-interface";
import { DemoBanner } from "@/components/demo-banner";
import { useDemoSession } from "@/hooks/useDemoSession";
import { LoadingScreen } from "@/components/loading/LoadingScreen";

const queryClient = new QueryClient();

function App() {
  const { sessionId, isReady } = useDemoSession();

  if (!isReady || !sessionId) {
    return <LoadingScreen />;
  }

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="dark" storageKey="mix-web-demo-theme">
        <TooltipProvider>
          <div className="flex h-screen flex-col">
            <DemoBanner />
            <div className="flex-1 overflow-hidden">
              <ChatInterface sessionId={sessionId} />
            </div>
          </div>
          <Toaster position="top-right" />
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

export default App;
```

#### 4.5 Create Entry Point
```tsx
// src/main.tsx
import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "./styles/App.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
```

### Phase 5: Modify Utilities for Web (30 minutes)

#### 5.1 Update backendUrl.ts
```ts
// src/utils/backendUrl.ts
export const getBackendUrl = (): string => {
  return import.meta.env.VITE_BACKEND_URL || "http://localhost:8080";
};
```

#### 5.2 Update platform.ts
```ts
// src/utils/platform.ts
export const getPlatform = () => "web" as const;
export const isTauri = () => false;
export const isWeb = () => true;
```

#### 5.3 Update mix-sdk.ts
```ts
// src/lib/mix-sdk.ts
import { Mix } from "mix-typescript-sdk";
import { getBackendUrl } from "@/utils/backendUrl";

export const mix = new Mix({
  serverURL: getBackendUrl(),
});
```

### Phase 6: Vite Configuration (15 minutes)

```ts
// vite.config.ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 3000,
    proxy: {
      // Proxy API calls to backend during development
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
      "/stream": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: "dist",
    sourcemap: true,
  },
});
```

### Phase 7: Testing (30 minutes)

#### 7.1 Local Development Testing
```bash
# Terminal 1: Start backend
cd mix_agent
go run main.go

# Terminal 2: Start web demo
cd mix_web_demo
npm run dev
```

Test:
- [ ] Auto-create session on page load
- [ ] Send message and receive streaming response
- [ ] Display thinking/reasoning correctly
- [ ] Display tool calls correctly
- [ ] Display todos correctly
- [ ] Display plan mode correctly
- [ ] File upload works (if included)
- [ ] Edit/rewind message works
- [ ] Export session works (if included)
- [ ] Page refresh restores session
- [ ] Handle backend disconnection gracefully
- [ ] Multiple browser tabs work independently (session isolation)

#### 7.2 Browser Compatibility
- [ ] Chrome/Edge (latest)
- [ ] Firefox (latest)
- [ ] Safari (latest)
- [ ] Mobile Safari (iOS)
- [ ] Chrome Android

### Phase 8: Build and Deploy (30 minutes)

#### 8.1 Build Production Bundle
```bash
npm run build
```

#### 8.2 Option A: Serve from Go Backend
```go
// In mix_agent/internal/http/server.go

// Serve web demo static files
mux.Handle("/demo/", http.StripPrefix("/demo/",
  http.FileServer(http.Dir("../mix_web_demo/dist"))))

// SPA fallback for client-side routing (if needed)
mux.HandleFunc("/demo", func(w http.ResponseWriter, r *http.Request) {
  http.ServeFile(w, r, "../mix_web_demo/dist/index.html")
})
```

#### 8.3 Option B: Deploy to Vercel/Netlify
```bash
# Install Vercel CLI
npm i -g vercel

# Deploy
cd mix_web_demo
vercel --prod
```

Set environment variables:
- `VITE_BACKEND_URL=https://your-backend.com`

#### 8.4 Option C: Docker Compose
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
      - "3000:80"
    environment:
      - VITE_BACKEND_URL=http://backend:8080
```

## Key Implementation Notes

### What to Copy As-Is (No Changes)
1. **All UI Components** - They're framework-agnostic
2. **All AI Components** (kibo-ui) - Pure React, no Tauri deps
3. **usePersistentSSE** - Works perfectly for web
4. **ConversationDisplay** - No changes needed
5. **ResponseRenderer** - No changes needed
6. **All message/tool rendering** - Framework-agnostic

### What to Remove
1. **TanStack Router** - Not needed (single page)
2. **AppSidebar** - No sidebar in web demo
3. **Slash Commands** - Remove CommandSlash component
4. **Settings Dialog** - Remove settings-dialog.tsx
5. **Command Palette** - Remove command-slash.tsx and all related code
6. **Multi-session code** - Only keep single session hooks

### What to Simplify
1. **chat-app.tsx → chat-interface.tsx** - Remove routing, slash commands, command palette
2. **useSession.ts** - Only keep `useActiveSession` and `useCreateSession`
3. **backendUrl.ts** - Use env var instead of Tauri API
4. **platform.ts** - Always return 'web'

### What to Create New
1. **demo-banner.tsx** - Show demo mode indicator
2. **useDemoSession.ts** - Auto-create and persist session
3. **App.tsx** - Main app wrapper
4. **main.tsx** - Entry point

## Estimated Time

- **Phase 1: Setup** - 30 minutes
- **Phase 2: Copy Components** - 1 hour
- **Phase 3: Copy Hooks/Utils** - 30 minutes
- **Phase 4: Create New Components** - 1 hour
- **Phase 5: Modify Utils** - 30 minutes
- **Phase 6: Vite Config** - 15 minutes
- **Phase 7: Testing** - 30 minutes
- **Phase 8: Deploy** - 30 minutes

**Total: 5 hours**

## Success Criteria

✅ Web demo looks **identical** to desktop app
✅ Single session auto-created on load
✅ SSE streaming works perfectly
✅ All tool displays work (thinking, reasoning, todos, plans)
✅ Message editing/rewinding works
✅ File upload works (if included)
✅ Works on all major browsers
✅ Mobile responsive
✅ Fast load time (< 2 seconds)
✅ Clear CTA to desktop app

## Future Enhancements

### Phase 2 (Post-MVP)
- [ ] Share session via URL
- [ ] Rate limiting indicator
- [ ] Session export/import
- [ ] Analytics tracking (PostHog)
- [ ] Example prompts gallery

### Phase 3
- [ ] WebSocket fallback for SSE
- [ ] Offline support (PWA)
- [ ] Multi-language support
- [ ] Accessibility improvements (a11y)

## Conclusion

This plan creates a production-ready web demo by:
1. **Copying 95% of frontend code** from mix_dev_tool (no reinventing the wheel!)
2. **Removing complexity** (routing, sidebar, commands, settings)
3. **Adding auto-session** for zero-config UX
4. **Keeping everything else identical** (SSE, message display, tool rendering)

The result: A lightweight web demo that **looks and works exactly like the desktop app**, perfect for quick trials and demos!
