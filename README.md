# Mix

[![Twitter Follow](https://img.shields.io/twitter/follow/Sarath?style=social)](https://x.com/intent/user?screen_name=sarath_suresh_m)
[![Twitter Follow](https://img.shields.io/twitter/follow/Vaibhav?style=social)](https://x.com/intent/user?screen_name=Vaibhav30665241)
[![Documentation](https://img.shields.io/badge/Documentation-📕-blue)](https://recreate.run/docs/backend)

Mix is an open-source, local agent for multimodal tasks. Claude code users will feel at home.

## Quick start

Then, run

```bash
git clone https://github.com/recreate-run/mix.git
make dev
```

- Yse `/login` command to authenticate with model providers or set API keys . Anthropic and Openrouter are curretly supported. We recommend claude-sonnet-4.  
- If you have a claude code account already, you can login with your anthropic account. Otherwise, you can set the anthropic API key using `/login`,
- The web search tool requires a brave API key (freely available) and the web search tool requires a gemeini API key (freely available). Please set it in the `.env` for the best performance.

<https://github.com/user-attachments/assets/be6ca94c-dc91-4129-86a7-f00e3e5407b5>

📋 Key Features

- Uses ffmpeg and local apps like blender instead of clunky cloud based editors
- All project data is stored plain text and native media files - absolutely no lock-in.create a  
- The backend is an HTTP server, meaning that the frontend is just one of possible clients. Our python and typescript SDK's  (similar to claude code SDK) is launching soon.

## Agentic Coding

This project is optimized for AI-assisted development with integrated tooling and workflows.

**CLAUDE.md**: Contains AI-specific development guidelines that override default behavior.

### Unified Development Environment

- **Shoreman Process Manager**: `scripts/shoreman.sh` runs both frontend and backend simultaneously
- **Auto-reload**: Backend uses Go Air for hot reloading, frontend uses Vite's built-in HMR
- **Unified Logging**: All process output is aggregated with timestamps and color-coded by service
- **Console Log Forwarding**: Browser console logs are forwarded to terminal via `mix_playground/src/vite-console-forward-plugin.ts`

### Development Monitoring

```bash
make tail-log    # View last 100 lines of unified development logs
```

All development output (backend compilation, frontend builds, runtime logs, browser console) flows through a single log file for streamlined AI-assisted debugging.

## Structure

```
├── mix_agent/          # Go backend service
├── mix_playground/  # Tauri desktop application
├── .gitignore          # Monorepo gitignore
└── README.md           # This file
```

## Tools

1. Blender, pixelmator tools
2. Multimodal analyzer

## Roadmap

- Support for other models for the main agent and multimodal analyzer
- Image generation and video generation tool
- Pixelmator for image editing
- Mix SDK (similar to claude code SDK)

## Contributing

Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for information on how to contribute to the project, including our branch strategy and pull request workflow.

## Tech Stack

<p align="center">
  <img alt="Tauri" src="https://img.shields.io/badge/-Tauri-24C8DB?style=flat-square&logo=tauri&logoColor=white" />
  <img alt="TanStack Query" src="https://img.shields.io/badge/-TanStack%20Query-FF4154?style=flat-square&logo=react-query&logoColor=white" />
  <img alt="Radix UI" src="https://img.shields.io/badge/-Radix%20UI-161618?style=flat-square&logo=radix-ui&logoColor=white" />
  <img alt="Vite" src="https://img.shields.io/badge/-Vite-646CFF?style=flat-square&logo=vite&logoColor=white" />
  <img alt="GSAP" src="https://img.shields.io/badge/-GSAP-0ae448?style=flat-square&logo=greensock&logoColor=white" />
  <img alt="SQLite" src="https://img.shields.io/badge/-SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white" />
  <img alt="FFmpeg" src="https://img.shields.io/badge/-FFmpeg-007808?style=flat-square&logo=ffmpeg&logoColor=white" />
</p>

## Thanks

1. All third part software that we've used in the project, especadd. ially ffmpeg, gsap and the archived opencode project
