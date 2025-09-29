#!/usr/bin/env bash

# Enhanced release script for Mix Agent
# Handles versioning, validation, testing, and release creation

set -eo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="mix"
GO_VERSION_MIN="1.24.0"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Parse command line arguments
minor=false
dry_run=false
skip_tests=false
force=false

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --minor         Increment minor version (x.Y.z -> x.Y+1.0)"
    echo "  --dry-run       Test the release process without creating actual release"
    echo "  --skip-tests    Skip running tests before release"
    echo "  --force         Force release even if validation fails"
    echo "  --help          Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                    # Patch release (x.y.Z -> x.y.Z+1)"
    echo "  $0 --minor           # Minor release (x.Y.z -> x.Y+1.0)"
    echo "  $0 --dry-run         # Test release process"
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --minor) minor=true; shift 1;;
    --dry-run) dry_run=true; shift 1;;
    --skip-tests) skip_tests=true; shift 1;;
    --force) force=true; shift 1;;
    --help) usage; exit 0;;
    *) echo "Unknown parameter: $1"; usage; exit 1;;
  esac
done

# Utility functions
log_info() {
    echo -e "${BLUE}INFO:${NC} $1"
}

log_success() {
    echo -e "${GREEN}SUCCESS:${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}WARNING:${NC} $1"
}

log_error() {
    echo -e "${RED}ERROR:${NC} $1"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Validation functions
validate_environment() {
    log_info "Validating environment..."

    # Check if we're in the right directory
    if [[ ! -f "$PROJECT_ROOT/mix_agent/main.go" ]]; then
        log_error "Not in project root or main.go not found"
        exit 1
    fi

    # Check Go installation and version
    if ! command_exists go; then
        log_error "Go is not installed"
        exit 1
    fi

    local go_version=$(go version | grep -o 'go[0-9.]*' | sed 's/go//')
    log_info "Go version: $go_version"

    # Check Git
    if ! command_exists git; then
        log_error "Git is not installed"
        exit 1
    fi

    # Check if working directory is clean
    if [[ -n $(git status --porcelain) ]] && [[ "$force" != true ]]; then
        log_error "Working directory is not clean. Commit or stash changes first."
        log_info "Use --force to override this check"
        exit 1
    fi

    # Check if on main/master branch
    local current_branch=$(git branch --show-current)
    if [[ "$current_branch" != "main" && "$current_branch" != "master" ]] && [[ "$force" != true ]]; then
        log_warning "Not on main/master branch (current: $current_branch)"
        log_info "Use --force to override this check"
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi

    # Check GoReleaser if not dry run
    if [[ "$dry_run" != true ]] && ! command_exists goreleaser; then
        log_error "GoReleaser is not installed. Install with: brew install goreleaser"
        exit 1
    fi

    log_success "Environment validation passed"
}

run_tests() {
    if [[ "$skip_tests" == true ]]; then
        log_warning "Skipping tests as requested"
        return 0
    fi

    log_info "Running tests..."

    # Change to mix_agent directory for Go operations
    cd "$PROJECT_ROOT/mix_agent"

    # Run Go tests
    if ! go test ./...; then
        log_error "Go tests failed"
        exit 1
    fi

    # Run linting if available
    if command_exists golangci-lint; then
        log_info "Running linter..."
        if ! golangci-lint run ./...; then
            log_error "Linting failed"
            exit 1
        fi
    else
        log_warning "golangci-lint not found, skipping linting"
    fi

    # Test macOS builds
    log_info "Testing macOS build process..."

    # Test Intel build
    if ! GOOS=darwin GOARCH=amd64 go build -o "/tmp/${BINARY_NAME}-mac-intel-test" ./main.go; then
        log_error "macOS Intel build test failed"
        exit 1
    fi
    rm -f "/tmp/${BINARY_NAME}-mac-intel-test"

    # Test Apple Silicon build
    if ! GOOS=darwin GOARCH=arm64 go build -o "/tmp/${BINARY_NAME}-mac-arm-test" ./main.go; then
        log_error "macOS Apple Silicon build test failed"
        exit 1
    fi
    rm -f "/tmp/${BINARY_NAME}-mac-arm-test"

    cd "$PROJECT_ROOT"
    log_success "All tests passed"
}

get_next_version() {
    # Fetch all tags
    git fetch --force --tags >/dev/null 2>&1

    # Get the latest Git tag
    local latest_tag=$(git tag --sort=committerdate | grep -E '^v?[0-9]+\.[0-9]+\.[0-9]+$' | tail -1)

    # If there is no tag, start with v0.1.0
    if [ -z "$latest_tag" ]; then
        echo "v0.1.0"
        return
    fi

    log_info "Latest tag: $latest_tag"

    # Remove 'v' prefix if present
    local version="${latest_tag#v}"

    # Split the version into major, minor, and patch numbers
    IFS='.' read -ra VERSION <<< "$version"
    local major="${VERSION[0]}"
    local minor="${VERSION[1]}"
    local patch="${VERSION[2]}"

    if [ "$minor" = true ]; then
        # Increment the minor version and reset patch to 0
        ((minor++))
        echo "v${major}.${minor}.0"
    else
        # Increment the patch version
        ((patch++))
        echo "v${major}.${minor}.${patch}"
    fi
}

create_release() {
    local new_version="$1"

    log_info "Creating release: $new_version"

    if [[ "$dry_run" == true ]]; then
        log_info "DRY RUN: Would create tag $new_version and trigger release"
        log_info "DRY RUN: Would run: goreleaser build --snapshot --clean"

        # Test GoReleaser build
        cd "$PROJECT_ROOT"
        if command_exists goreleaser; then
            goreleaser build --snapshot --clean
            log_success "DRY RUN: GoReleaser build test completed"
        fi
        return 0
    fi

    # Create and push tag
    git tag "$new_version"
    git push --tags

    log_success "Tag $new_version created and pushed"
    log_info "GitHub Actions will automatically create the release"
    log_info "Monitor the release at: https://github.com/$(git config --get remote.origin.url | sed 's/.*github.com[:\/]\(.*\)\.git/\1/')/actions"
}

generate_changelog() {
    local previous_tag="$1"
    local new_version="$2"

    log_info "Generating changelog..."

    if [[ -z "$previous_tag" ]]; then
        log_info "No previous tag found, listing all commits"
        git log --oneline --pretty=format:"- %s (%an)" > /tmp/changelog.md
    else
        git log "${previous_tag}..HEAD" --oneline --pretty=format:"- %s (%an)" > /tmp/changelog.md
    fi

    if [[ -s /tmp/changelog.md ]]; then
        echo ""
        echo "=== CHANGELOG FOR $new_version ==="
        cat /tmp/changelog.md
        echo "================================="
        echo ""
    else
        log_warning "No changes found for changelog"
    fi
}

main() {
    log_info "Starting macOS release process for Mix Agent"

    # Change to project root
    cd "$PROJECT_ROOT"

    # Validate environment
    validate_environment

    # Run tests
    run_tests

    # Get version information
    local current_tag=$(git tag --sort=committerdate | grep -E '^v?[0-9]+\.[0-9]+\.[0-9]+$' | tail -1)
    local new_version=$(get_next_version)

    # Generate changelog
    generate_changelog "$current_tag" "$new_version"

    # Confirm release
    if [[ "$dry_run" != true ]]; then
        echo ""
        log_info "Ready to create release: $new_version"
        read -p "Continue with release? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Release cancelled by user"
            exit 0
        fi
    fi

    # Create release
    create_release "$new_version"

    if [[ "$dry_run" == true ]]; then
        log_success "Dry run completed successfully"
    else
        log_success "Release $new_version initiated successfully!"
        log_info "The release will be built automatically by GitHub Actions"
        log_info "Check the progress at: https://github.com/$(git config --get remote.origin.url | sed 's/.*github.com[:\/]\(.*\)\.git/\1/')/releases"
    fi
}

# Trap errors and cleanup
trap 'log_error "Script failed on line $LINENO"' ERR

# Run main function
main "$@"