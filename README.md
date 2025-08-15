# Mix

We’re building "claude code" for complex multimodal workflows. It’s a desktop app that uses existing tools like Blender and Figma. This enables workflows like: “generate videos and edit videos in blender, generate sound effects and edit in Logic Pro, post process in after-effects”.  Startups use it to automate marketing video generation, analyzing session recordings etc. The backend can also be embedded into other products via our SDK (coming soon).

📋 Key Features

- Just authenticate with your claude code account ($20 account works) to get started.
- All project data is stores plain text and native media files - absolutely no lock-in.
- The backend is an HTTP server, meaning that the frontend is just one of possible clients. Our SDK with stdio interface (similar to claude code SDK) is launching soon.


## Stack

Quick Install

**Frontend:**
**Backend:**

📦 Installation

## Configuration

The system requires explicit model configuration for both main and sub-agents. 

**Step 1:** Create a configuration file (`.mix.json`) in your home directory or project root:

```json
{
  "agents": {
    "main": {
      "model": "claude-4-sonnet",
      "maxTokens": 4096
    },
    "sub": {
      "model": "claude-4-sonnet", 
      "maxTokens": 2048
    }
  }
}
```

🔐 Authentication Options

- Claude Sonnet 4 as the default model for the agent backbone. Other models might work but they're untested. We recommend authenticating with your claude code code account ($20 plan is enough to get started). You can also use it with an API key.
- Gemini 2.5 FLashis used via the multimodal analyzer tool, to analyse images,videos and audio. You can authenticate via your google acount (like Gemini CLI) or use your gemini API key



**Important:** API keys must always come from environment variables, never store them in configuration files. The system automatically detects available providers from environment variables and creates the necessary provider configurations.

The system will fail immediately if agents are not configured or required API keys are missing.

## Development

Each project maintains its own build system and dependencies. Refer to the individual README files in each project directory for specific development instructions.



## Development

### Start frontend and backend together with unified logging

```bash
make dev
```

### Separate setup

#### Frontend

```bash
cd mix_tauri_app
npm install
npm run tauri dev
```

#### Backend 

To use in HTTP server mode (to use with the frontend)

```bash
./mix --http-port 8080
```

Use in CLI mode

```bash
./mix -p "Your prompt here"
```

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

## Thanks

1. ffmpeg
2. remotion

