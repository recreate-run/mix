# Mix

[![Twitter Follow](https://img.shields.io/twitter/follow/Sarath?style=social)](https://x.com/intent/user?screen_name=sarath_suresh_m)
[![Twitter Follow](https://img.shields.io/twitter/follow/Vaibhav?style=social)](https://x.com/intent/user?screen_name=Vaibhav30665241)
[![Documentation](https://img.shields.io/badge/Documentation-📕-blue)](https://recreate.run/docs/mix)

Mix is a multimodal agents SDK. It comes with a GUI playground for visualization and debugging .

- Native video/audio/PDF analysis tools (via Gemini for vision, Claude for reasoning)
- Multi-model routing instead of single-provider lock-in
- One-command Supabase setup for cloud deployment (vs localhost-only)
- HTTP architecture that enables visual DevTools alongside agent workflows
- Go backend: 50-80% lower memory footprint than Node.js—efficient for concurrent agent sessions.  Python and typescript clients are available

## Quickstart with DevTools

Mix comes with an interactive browser-based DevTools playground for testing and debugging workflows. Built with the [Mix TypeScript SDK](https://github.com/recreate-run/mix-typescript-sdk).

```bash
git clone https://github.com/recreate-run/mix.git
cd mix
make dev
```

This will start the Mix backend and open the DevTools interface in your browser.

**Authentication:** Configure LLM providers using the settings dialog (⚙️ gear icon at bottom left)

1. **Anthropic** - Claude Sonnet 4 via OAuth (Claude Code account) or API key
2. **Gemini** - Gemini API key for `ReadMedia` tool ([Get free key](https://aistudio.google.com/api-keys))
3. **Brave** - API key for `Search` tool ([Get free key](https://brave.com/search/api/))

**Try these prompts:**

```
Find the top cat video and create a 5 sec tiktok video from it. Add a title animation and export to a video
```

<https://github.com/user-attachments/assets/967a5bde-c90c-4d47-b5f6-0b11c75a1c35>

```
First, find the top 3 karpathy LLM videos, then find the most important 10 second section from each video. After that, download the sections and show it.
```

<https://github.com/user-attachments/assets/87ad85d1-4805-47fb-9a0b-490ae0f529f3>

```
Look at my portfolio in the data in @{file_info.url} and find the top winners and losers in Q4. Show the three most relevant plots.

```

<https://github.com/user-attachments/assets/fcc37dc1-1c89-4b3c-97b9-abdf64be4c24>

## Quickstart with CLI

Run Mix directly from the command line without DevTools. Requires API keys in `.env` file.

```bash
# Build the binary
make build

# Run a prompt
./mix_agent/build/mix -p "what is 2 + 2"

# JSON output
./mix_agent/build/mix -p "list 3 fruits" -f json

# Quiet mode (hide spinner)
./mix_agent/build/mix -q -p "hello world"
```

See `mix --help` for all CLI options.

## Quickstart with SDK

Build AI workflows with the Python SDK. Uses [uv](https://docs.astral.sh/uv/) for fast package management.

**Prerequisites:** Mix backend must be running (see DevTools quickstart above)

```bash
git clone https://github.com/recreate-run/mix-python-sdk.git
cd mix-python-sdk
```

**Run examples:**

```bash
# Create TikTok-style videos
uv run examples/tiktok_video.py

# Multimodal web search
uv run examples/web_search_multimodal.py

# Customize system prompts
uv run examples/system_prompt_change.py
```

## Database Support

Mix supports multiple database backends via environment configuration:

- **SQLite** (default) - Zero-config file-based storage for local development
- **PostgreSQL** - Production-ready with Docker Compose setup (`task db-up`)
- **Turso** - Distributed edge SQLite for global deployments

Switch providers by setting `MIX_DB_TYPE` in `.env`. See [DATABASE_SETUP.md](DATABASE_SETUP.md) for configuration details.

## Roadmap

- Image/Video generation
- REST clients in other languages

## Tech Stack

<p align="center">
  <img alt="React" src="https://img.shields.io/badge/-React-61DAFB?style=flat-square&logo=react&logoColor=black" />
  <img alt="TanStack Query" src="https://img.shields.io/badge/-TanStack%20Query-FF4154?style=flat-square&logo=react-query&logoColor=white" />
  <img alt="Radix UI" src="https://img.shields.io/badge/-Radix%20UI-161618?style=flat-square&logo=radix-ui&logoColor=white" />
  <img alt="Vite" src="https://img.shields.io/badge/-Vite-646CFF?style=flat-square&logo=vite&logoColor=white" />
  <img alt="SQLite" src="https://img.shields.io/badge/-SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white" />
  <img alt="FFmpeg" src="https://img.shields.io/badge/-FFmpeg-007808?style=flat-square&logo=ffmpeg&logoColor=white" />
</p>

## Thanks

1. All third party software that we've used in the project, especially ffmpeg and the archived opencode project
