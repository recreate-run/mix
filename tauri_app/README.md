# Mix Desktop App

AI-powered content generation and management desktop application built with Tauri, React, and TypeScript.

## Development

```bash
bun run dev    # Start development server
```

## App Icons

To update the app icon, replace `app-icon.png` in the root directory and regenerate icons:

```bash
# Generate all platform icons
bun run tauri icon

# Clean up non-macOS icons (optional for macOS-only builds)
cd src-tauri/icons
rm -rf ios android
rm -f icon.ico Square*.png StoreLogo.png 64x64.png icon.png
```

Generated icons are stored in `src-tauri/icons/` and automatically referenced in the bundle configuration.

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

## Release Process

### Auto-Update Setup
- Signing keys configured with public key validation
- GitHub Actions workflow builds signed macOS binaries (Intel + Apple Silicon)
- Users receive automatic update notifications in-app

### Creating a Release
1. Update version in `package.json` and `src-tauri/tauri.conf.json`
2. Create and push a git tag: `git tag v1.0.1 && git push origin v1.0.1`
3. GitHub Actions automatically builds and releases signed update bundles
4. Users get notified of updates automatically

### Required GitHub Secrets
- `TAURI_SIGNING_PRIVATE_KEY`: Content of `~/.tauri/mix.key`
- `TAURI_SIGNING_PRIVATE_KEY_PASSWORD`: `mazhaneer123`

### Manual Build (for testing)
```bash
export TAURI_SIGNING_PRIVATE_KEY="$(cat ~/.tauri/mix.key)"
export TAURI_SIGNING_PRIVATE_KEY_PASSWORD="mazhaneer123"
bun tauri build
```

## Recommended IDE Setup

- [VS Code](https://code.visualstudio.com/) + [Tauri](https://marketplace.visualstudio.com/items?itemName=tauri-apps.tauri-vscode) + [rust-analyzer](https://marketplace.visualstudio.com/items?itemName=rust-lang.rust-analyzer)
