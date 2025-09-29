# Release Guide for Mix Agent

This document describes the complete release process for the Mix Agent Go server binary with cross-platform support.

## Supported Platforms

- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Linux**: amd64, arm64
- **Windows**: amd64, arm64

## Release Methods

### 1. Automated Release (Recommended)

#### Via GitHub Actions (Manual Trigger)
```bash
# Go to GitHub Actions → Release workflow → Run workflow
# Choose version type: patch (default) or minor
```

#### Via Git Tags
```bash
# Create and push a version tag
git tag v1.0.0
git push --tags

# This automatically triggers the release workflow
```

### 2. Enhanced Release Script

Use the enhanced release script with validation and testing:

```bash
# Patch release (x.y.Z -> x.y.Z+1)
./scripts/release.sh

# Minor release (x.Y.z -> x.Y+1.0)
./scripts/release.sh --minor

# Dry run (test without creating release)
./scripts/release.sh --dry-run

# Skip tests (not recommended)
./scripts/release.sh --skip-tests

# Force release (bypass validation)
./scripts/release.sh --force
```

### 3. Manual Build Commands

#### Build for Current Platform
```bash
make build
```

#### Build for All Platforms
```bash
make build-all
```

#### Build for Specific Platforms
```bash
make build-darwin-amd64    # macOS Intel
make build-darwin-arm64    # macOS Apple Silicon
make build-linux-amd64     # Linux x86_64
make build-linux-arm64     # Linux ARM64
make build-windows-amd64   # Windows x86_64
make build-windows-arm64   # Windows ARM64
```

#### GoReleaser Commands
```bash
# Full release
make release

# Test build (snapshot)
make release-test

# Snapshot release
make release-snapshot
```

## Release Workflow

### Automated Process
1. **Validation**: Environment checks, Go version, clean working directory
2. **Testing**: Run Go tests, linting, and build verification
3. **Versioning**: Automatic version increment based on latest git tag
4. **macOS Build**: GoReleaser builds for both macOS architectures
5. **Release Creation**: GitHub release with artifacts and checksums
6. **Artifact Upload**: Binaries, archives, and checksums uploaded to release

### Manual Process
1. Run `./scripts/release.sh --dry-run` to validate
2. Review changelog and version
3. Run `./scripts/release.sh` to create release
4. Monitor GitHub Actions for build completion

## Build Outputs

### File Structure
```
dist/
├── mix-darwin-amd64.tar.gz
├── mix-darwin-arm64.tar.gz
├── mix-linux-amd64.tar.gz
├── mix-linux-arm64.tar.gz
├── mix-windows-amd64.zip
├── mix-windows-arm64.zip
├── checksums.txt
└── ...
```

### Binary Names
- **macOS**: `mix-darwin-{arch}`
- **Linux**: `mix-linux-{arch}`
- **Windows**: `mix-windows-{arch}.exe`

## Configuration

### GoReleaser (.goreleaser.yml)
- Cross-platform build configuration
- Archive formats (tar.gz for Unix, zip for Windows)
- Checksum generation
- Package manager integration (commented out)

### GitHub Actions (.github/workflows/release.yml)
- Multi-job workflow with validation
- Matrix builds for testing
- Artifact management
- Release automation

### Makefile
- Individual platform build targets
- GoReleaser integration
- Development and release commands

## Troubleshooting

### Common Issues

#### GoReleaser Not Found
```bash
# Install GoReleaser
brew install goreleaser

# Or use GitHub Actions (recommended)
```

#### Build Failures
```bash
# Check Go version (requires 1.24.0+)
go version

# Clean and retry
make clean
make build-all
```

#### Permission Issues
```bash
# Make release script executable
chmod +x scripts/release.sh
```

### Validation Failures
The release script validates:
- Clean working directory
- Correct branch (main/master)
- Go tests passing
- Build successful
- GoReleaser configuration valid

Use `--force` to bypass validation (not recommended).

## Development vs Release

### Development Build
- Built by Air for hot reloading
- Located at `mix_agent/build/debug/mix`
- Includes debug symbols
- Current platform only

### Release Build
- Optimized with `-s -w` flags
- Both macOS architectures (Intel + Apple Silicon)
- Compressed tar.gz archives
- Version information embedded
- Production-ready

## Version Management

### Version Format
- Semantic versioning: `vMAJOR.MINOR.PATCH`
- Example: `v1.2.3`

### Version Sources
1. Git tags (authoritative)
2. Automatic increment by release script
3. Manual specification via GitHub Actions

### Version Injection
Build-time version information is injected via ldflags:
```go
// mix/internal/version/version.go
var Version string // Set at build time
```

## Next Steps

To enable package manager distribution:

1. Uncomment package manager sections in `.goreleaser.yml`
2. Set up Homebrew tap repository
3. Configure AUR package
4. Add secrets for publishing tokens

This provides a complete, production-ready release system for the Mix Agent.