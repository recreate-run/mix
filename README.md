# Mix

[![Twitter Follow](https://img.shields.io/twitter/follow/Sarath?style=social)](https://x.com/intent/user?screen_name=sarath_suresh_m)
[![Twitter Follow](https://img.shields.io/twitter/follow/Vaibhav?style=social)](https://x.com/intent/user?screen_name=Vaibhav30665241)
[![Documentation](https://img.shields.io/badge/Documentation-📕-blue)](https://recreate.run/docs/mix-agent)

Mix is an open-source, local agent for multimodal tasks. Claude code users will feel at home.

- **Mix Agent**: Go backend HTTP server with CLI interface
- **Mix Playground**: Desktop app built with Tauri and React  
- **SDKs**: TypeScript and Python SDKs for integration

## Quick start

```bash
git clone https://github.com/recreate-run/mix.git
make dev
```

Authenticate LLM providers and tool API's usring the settings dialog (gear icon at the bottom left)

1. We recommend claude-sonnet-4 via the oauth (claude code account) or the anthropic API key using the Settings dialog.
2. Set Gemini API key for the `ReadMedia tool. Available for free from [AI Studio](https://aistudio.google.com/api-keys)
3. Set Brave API key for the `Search` tool. Available for free from [Brave](https://brave.com/search/api/)

Try out some examples in the playground

```bash
Find the top cat video and create a 5 sec tiktok video from it. Add a title animation and export to a video
```

```bash
Find the top cat video and create a 5 sec tiktok video from it. Add a title animation and export to a video
```

```bash
Find the top cat video and create a 5 sec tiktok video from it. Add a title animation and export to a video
```

<https://github.com/user-attachments/assets/be6ca94c-dc91-4129-86a7-f00e3e5407b5>

📋 Key Features

- Uses ffmpeg and local apps like blender instead of clunky cloud based editors
- All project data is stored plain text and native media files - absolutely no lock-in.create a  
- The backend is an HTTP server, meaning that the frontend is just one of possible clients. Our python and typescript SDK's  (similar to claude code SDK) is launching soon.

## Roadmap

- Support for other models for the main agent and multimodal analyzer
- Image generation and video generation tool
- Pixelmator for image editing
- Mix SDK (similar to claude code SDK)

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
