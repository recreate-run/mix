# Release Guide

This document outlines the GitHub-based release process for Mix.

## How to Create a Release

### 1. Go to GitHub Actions
1. Visit [GitHub Actions](https://github.com/your-username/mix/actions)
2. Click on "Release" workflow
3. Click "Run workflow"
4. Enter version (e.g., `v1.0.0`)
5. Click "Run workflow"

### 2. Wait for Build
The workflow will automatically:
- Build for all platforms (macOS, Linux, Windows)
- Create desktop app installers (.dmg, .deb, .AppImage, .msi, .exe)
- Build CLI binaries for all platforms
- Generate SHA256 checksums
- Create a **draft release** with all assets

### 3. Review and Publish
1. Go to [GitHub Releases](https://github.com/your-username/mix/releases)
2. Find your draft release
3. Review the generated release notes
4. Edit description if needed
5. Click "Publish release"

## Release Artifacts

Each release includes:

**Desktop Applications:**
- `Mix.dmg` - macOS installer (Universal)
- `mix_*.deb` - Linux Debian package
- `mix_*.AppImage` - Linux portable app
- `Mix_*.msi` - Windows installer
- `Mix_*.exe` - Windows executable

**CLI Binaries:**
- `mix-x86_64-apple-darwin` - macOS Intel
- `mix-aarch64-apple-darwin` - macOS Apple Silicon
- `mix-x86_64-unknown-linux-gnu` - Linux
- `mix-x86_64-pc-windows-msvc.exe` - Windows

**Verification:**
- `SHA256SUMS.txt` - Checksums for all files

## Local Development Builds

For local testing, you can still use:

```bash
make release          # Current platform only
make release-macos    # macOS builds
make release-linux    # Linux builds  
make release-windows  # Windows builds
```

## Pre-Release Checklist

Before triggering a release:

- [ ] Update version in `tauri_app/src-tauri/tauri.conf.json`
- [ ] Update CHANGELOG.md
- [ ] Ensure all tests pass locally
- [ ] Commit and push all changes
- [ ] Verify CI builds are passing

## Notes

- All releases are created as **drafts** first for review
- Desktop apps are automatically signed (if certificates are configured)
- CLI binaries are statically linked for maximum portability
- The workflow builds on GitHub runners, so no local toolchain setup required