# Mix Desktop App

AI-powered content generation and management desktop application built with Tauri, React, and TypeScript.

## Development

```bash
bun run dev    # Start development server
```

## Distribution

Build distribution-ready packages for macOS:

```bash
# Build .app bundle only
bun tauri build --bundles app

# Build .dmg installer 
bun tauri build --bundles dmg

# Build both .app and .dmg
bun tauri build --bundles all
```

Built files are located in `src-tauri/target/release/bundle/macos/`

## Recommended IDE Setup

- [VS Code](https://code.visualstudio.com/) + [Tauri](https://marketplace.visualstudio.com/items?itemName=tauri-apps.tauri-vscode) + [rust-analyzer](https://marketplace.visualstudio.com/items?itemName=rust-lang.rust-analyzer)
