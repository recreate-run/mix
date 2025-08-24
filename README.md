# Mix

[![Twitter Follow](https://img.shields.io/twitter/follow/Sarath?style=social)](https://x.com/intent/user?screen_name=sarath_suresh_m)
[![Twitter Follow](https://img.shields.io/twitter/follow/Vaibhav?style=social)](https://x.com/intent/user?screen_name=Vaibhav30665241)
[![Documentation](https://img.shields.io/badge/Documentation-📕-blue)](https://recreate.run/docs/backend)

Mix is an open-source desktop platform for multimodal AI agents, multimodal claude code.

📋 Key Features

- All project data is stores plain text and native media files - absolutely no lock-in.
- The backend is an HTTP server, meaning that the frontend is just one of possible clients. Our SDK with stdio interface (similar to claude code SDK) is launching soon.

## Quick Install

1. The agent uses claude sonnet 4. You can authenticate with your claude code account using the `/login` command in the UI after installation ,  or set the  `ANHROPIC_API_KEY` in the `.env` file. Other models might work but they're untested.
2. A `GEMINI_API_KEY`  is required, since gemini-2.5-flash is used to analyse images,videos and audio., set it in the `.env` file. Ypu can get one free from google ai studio.

```bash
make install
```

Then, run

```bash
make dev
```

This starts bith frontend and backend together with unified logging to the same terminal. See agentic coding section below

## Configuration

The system requires explicit model configuration for both main and sub-agents.

### Configuration Hierarchy

Mix uses a **global → local** configuration hierarchy:

1. **Global config**: `~/.mix.json` - System-wide defaults
2. **Local config**: `./.mix.json` - Project-specific overrides (merges with global)

## Local Development

Install dependencies first

```bash
make install
```

### Frontend

```bash
cd tauri_app
bun run tauri dev
```

### Backend

To use in HTTP server mode (to use with the frontend)

```bash
cd go_backend
./mix --http-port 8080
```

Use in CLI mode

```bash
./mix -p "Your prompt here"
```

## Agentic Coding

This project is optimized for AI-assisted development with integrated tooling and workflows.

**CLAUDE.md**: Contains AI-specific development guidelines that override default behavior.

### Unified Development Environment

- **Shoreman Process Manager**: `scripts/shoreman.sh` runs both frontend and backend simultaneously
- **Auto-reload**: Backend uses Go Air for hot reloading, frontend uses Vite's built-in HMR
- **Unified Logging**: All process output is aggregated with timestamps and color-coded by service
- **Console Log Forwarding**: Browser console logs are forwarded to terminal via `tauri_app/src/vite-console-forward-plugin.ts`

### Development Monitoring

```bash
make tail-log    # View last 100 lines of unified development logs
```

All development output (backend compilation, frontend builds, runtime logs, browser console) flows through a single log file for streamlined AI-assisted debugging.

## Structure

```
├── go_backend/          # Go backend service
├── mix_tauri_app/  # Tauri desktop application
├── .gitignore          # Monorepo gitignore
└── README.md           # This file
```

## Demos

1. Creating remotion titles
2. Use blender for video editing
3. Asking highlights in a video
4. Analyze session recording

## Tools

1. Blender, pixelmator tools
2. Multimodal analyzer

## Roadmap

- Image generation and vide generation tool
- Storyboard and scene generation
- Pixelmator for image editing

## Tech Stack

<p align="center">
  <img alt="Tauri" src="https://img.shields.io/badge/-Tauri-24C8DB?style=flat-square&logo=tauri&logoColor=white" />
  <img alt="TanStack Query" src="https://img.shields.io/badge/-TanStack%20Query-FF4154?style=flat-square&logo=react-query&logoColor=white" />
  <img alt="Radix UI" src="https://img.shields.io/badge/-Radix%20UI-161618?style=flat-square&logo=radix-ui&logoColor=white" />
  <img alt="Vite" src="https://img.shields.io/badge/-Vite-646CFF?style=flat-square&logo=vite&logoColor=white" />
  <img alt="Remotion" src="https://img.shields.io/badge/-Remotion-4338CA?style=flat-square&logo=data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTEyIDJMMTMuMDkgOC4yNkwyMCA5TDEzLjA5IDE1Ljc0TDEyIDIyTDEwLjkxIDE1Ljc0TDQgOUwxMC45MSA4LjI2TDEyIDJaIiBmaWxsPSJ3aGl0ZSIvPgo8L3N2Zz4K&logoColor=white" />
  <img alt="SQLite" src="https://img.shields.io/badge/-SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white" />
  <img alt="FFmpeg" src="https://img.shields.io/badge/-FFmpeg-007808?style=flat-square&logo=ffmpeg&logoColor=white" />
</p>

## Thanks

1. All third part software that we've used in the project, especially ffmpeg and remotion
