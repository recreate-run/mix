# Mix Web Demo

A lightweight web version of Mix for public demos and quick trials. This is a pure React web application that shares the same backend as the desktop app but with a simplified interface.

## Features

✅ **Single chat interface** - Focus on one conversation at a time
✅ **Auto-created anonymous session** - Zero configuration needed
✅ **Real-time SSE streaming** - See responses as they're generated
✅ **Full message display** - Including thinking, reasoning, and tool execution
✅ **File upload support** - Upload files directly from the browser
✅ **Message editing** - Rewind and edit previous messages
✅ **Session export** - Download conversation transcripts
✅ **Mobile responsive** - Works on phones and tablets

## What's Different from Desktop App

| Feature | Desktop (mix_dev_tool) | Web Demo (mix_web_demo) |
|---------|------------------------|-------------------------|
| Platform | Tauri (Rust + React) | Pure React SPA |
| Sessions | Multiple with sidebar | Single, auto-created |
| Settings | Full settings dialog | None |
| Slash Commands | Yes | No |
| Model Selection | Yes | No (uses backend default) |
| File Upload | Native file picker | Browser file input |

## Development

### Prerequisites

- Node.js 18+ or Bun
- Backend running at configured URL

### Environment Variables

Create a `.env` file:

```bash
VITE_BACKEND_URL=http://localhost:8080
```

For production, use `.env.production`:

```bash
VITE_BACKEND_URL=https://your-backend-url.com
```

### Install Dependencies

```bash
bun install
```

### Development Server

```bash
bun run dev
```

The app will be available at `http://localhost:3000`

### Build for Production

```bash
bun run build
```

The production build will be in the `dist/` directory.

### Type Checking

```bash
bun run typecheck
```

## Architecture

### Key Components

- **App.tsx** - Main application wrapper with theme and query client
- **DemoBanner.tsx** - Demo mode indicator at the top
- **ChatInterface.tsx** - Simplified chat interface (no routing/commands)
- **ConversationDisplay.tsx** - Message display with tool execution
- **useDemoSession.ts** - Auto-creates and manages single session
- **usePersistentSSE.ts** - Handles real-time SSE streaming

### Data Flow

1. **Session Creation**: `useDemoSession` auto-creates a session on mount and stores it in localStorage
2. **SSE Connection**: `usePersistentSSE` establishes connection to backend
3. **Message Flow**: User → ChatInterface → SSE → Backend → SSE → ConversationDisplay
4. **State Management**: React Query for server state, Zustand for attachment state

## Deployment

### Option A: Serve from Backend

Build the frontend and serve it from your Go backend:

```bash
bun run build

# In your Go backend
mux.Handle("/demo/", http.StripPrefix("/demo/", http.FileServer(http.Dir("../mix_web_demo/dist"))))
```

### Option B: Deploy to Vercel/Netlify

```bash
# Install Vercel CLI
npm i -g vercel

# Deploy
vercel --prod
```

Set `VITE_BACKEND_URL` environment variable in your deployment platform.

### Option C: Docker

```dockerfile
FROM node:20-alpine
WORKDIR /app
COPY package.json bun.lock ./
RUN npm install -g bun && bun install
COPY . .
RUN bun run build

FROM nginx:alpine
COPY --from=0 /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

## Browser Support

- ✅ Chrome/Edge (latest)
- ✅ Firefox (latest)
- ✅ Safari (latest)
- ✅ Mobile Safari (iOS)
- ✅ Chrome Android

## Technical Notes

### Web-Only Limitations

The following desktop features are disabled in the web version:

- **Native file picker** - Uses browser file input instead
- **Native save dialog** - Downloads files directly to browser
- **Tauri APIs** - All `@tauri-apps/*` imports are stubbed or commented out

### Performance

- **SSE Connection**: Persistent connection for real-time updates
- **Code Splitting**: Automatic code splitting via Vite
- **Lazy Loading**: Video player and media components are lazy-loaded
- **Bundle Size**: ~1MB gzipped (including syntax highlighting for all languages)

## Troubleshooting

### Backend Connection Issues

If you see connection errors, check:

1. Backend is running and accessible
2. `VITE_BACKEND_URL` is set correctly
3. CORS is enabled on backend (should be by default)

### Session Not Persisting

The session ID is stored in `localStorage` with key `mix-demo-session-id`. Clear it to create a new session:

```javascript
localStorage.removeItem('mix-demo-session-id');
```

### Build Errors

If you encounter build errors:

1. Clear node_modules and lockfile: `rm -rf node_modules bun.lock`
2. Reinstall: `bun install`
3. Rebuild: `bun run build`

## Contributing

This web demo is designed to be minimal and focused. Major feature additions should go to the desktop app (`mix_dev_tool/`) instead.

For bugs and improvements to the web demo:
1. Test locally with `bun run dev`
2. Run type checking with `bun run typecheck`
3. Build successfully with `bun run build`
4. Test in multiple browsers

## License

Same as the main Mix project.
