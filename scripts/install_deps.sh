#!/bin/bash

# Color codes for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Detect the operating system
OS="$(uname)"
echo -e "${BOLD}Detected OS: $OS${NC}"

# Install FFmpeg based on OS
install_ffmpeg() {
  if ! command -v ffmpeg >/dev/null 2>&1; then
    echo -e "📦 ${BOLD}Installing FFmpeg...${NC}"
    if [ "$OS" = "Darwin" ]; then
      # macOS
      if command -v brew >/dev/null 2>&1; then
        brew install ffmpeg
      else
        echo -e "${YELLOW}Homebrew not found. Skipping FFmpeg installation.${NC}"
        echo -e "${YELLOW}Please install FFmpeg manually: https://ffmpeg.org/download.html${NC}"
      fi
    elif [ "$OS" = "Linux" ]; then
      # Linux
      if command -v apt-get >/dev/null 2>&1; then
        sudo apt-get update
        sudo apt-get install -y ffmpeg
      elif command -v yum >/dev/null 2>&1; then
        sudo yum install -y ffmpeg
      else
        echo -e "${YELLOW}Neither apt-get nor yum found. Skipping FFmpeg installation.${NC}"
        echo -e "${YELLOW}Please install FFmpeg manually: https://ffmpeg.org/download.html${NC}"
      fi
    else
      echo -e "${RED}Unsupported OS: $OS. Please install FFmpeg manually.${NC}"
    fi
  else
    echo -e "✅ ${GREEN}FFmpeg is already installed${NC}"
  fi
}

# Install Go based on OS
install_go() {
  if ! command -v go >/dev/null 2>&1; then
    echo -e "📦 ${BOLD}Installing Go...${NC}"
    if [ "$OS" = "Darwin" ]; then
      # macOS
      if command -v brew >/dev/null 2>&1; then
        brew install go
      else
        echo "Installing Go via official installer..."
        curl -L "https://go.dev/dl/go1.22.0.darwin-amd64.pkg" -o /tmp/go-installer.pkg && \
        sudo installer -pkg /tmp/go-installer.pkg -target / && \
        rm /tmp/go-installer.pkg
      fi
    elif [ "$OS" = "Linux" ]; then
      # Linux
      if command -v apt-get >/dev/null 2>&1; then
        sudo apt-get update
        sudo apt-get install -y golang-go
      elif command -v yum >/dev/null 2>&1; then
        sudo yum install -y golang
      else
        echo "Installing Go via official tarball..."
        wget -q -O /tmp/go.tar.gz "https://go.dev/dl/go1.22.0.linux-amd64.tar.gz" && \
        sudo tar -C /usr/local -xzf /tmp/go.tar.gz && \
        echo "export PATH=\$PATH:/usr/local/go/bin" >> $HOME/.bashrc && \
        export PATH=$PATH:/usr/local/go/bin && \
        rm /tmp/go.tar.gz
      fi
    else
      echo -e "${RED}Unsupported OS: $OS. Please install Go manually.${NC}"
    fi
    # Add Go to PATH if needed
    export PATH="/usr/local/go/bin:$PATH"
  else
    echo -e "✅ ${GREEN}Go is already installed${NC}"
  fi
}

# Install Rust/Cargo
install_rust() {
  if ! command -v cargo >/dev/null 2>&1; then
    echo -e "📦 ${BOLD}Installing Rust/Cargo...${NC}"
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
    source "$HOME/.cargo/env"
    export PATH="$HOME/.cargo/bin:$PATH"
  else
    echo -e "✅ ${GREEN}Rust/Cargo is already installed${NC}"
  fi
}

# Install Bun
install_bun() {
  if ! command -v bun >/dev/null 2>&1; then
    echo -e "📦 ${BOLD}Installing Bun...${NC}"
    curl -fsSL https://bun.sh/install | bash
    export PATH="$HOME/.bun/bin:$PATH"
    
    # Update PATH in shell profiles for future sessions
    if [ -f "$HOME/.bashrc" ] && ! grep -q "export PATH=.*\.bun/bin" "$HOME/.bashrc"; then
      echo 'export PATH="$HOME/.bun/bin:$PATH"' >> "$HOME/.bashrc"
    fi
    
    if [ "$OS" = "Darwin" ] && [ -f "$HOME/.zshrc" ] && ! grep -q "export PATH=.*\.bun/bin" "$HOME/.zshrc"; then
      echo 'export PATH="$HOME/.bun/bin:$PATH"' >> "$HOME/.zshrc"
    fi
    
    # Verify bun is in PATH after installation
    if ! command -v bun >/dev/null 2>&1; then
      echo -e "${YELLOW}Bun installed but not found in PATH.${NC}"
      echo -e "${YELLOW}Please run this command manually:${NC}"
      echo -e "${BOLD}export PATH=\"$HOME/.bun/bin:\$PATH\"${NC}"
    else
      echo -e "${GREEN}Bun installed and found in PATH at: $(which bun)${NC}"
    fi
  else
    echo -e "✅ ${GREEN}Bun is already installed${NC}"
  fi
}

# Install UV based on OS
install_uv() {
  if ! command -v uv >/dev/null 2>&1; then
    echo -e "📦 ${BOLD}Installing UV (Python package installer)...${NC}"
    if [ "$OS" = "Darwin" ] && command -v brew >/dev/null 2>&1; then
      brew install uv
    else
      # Works on both macOS and Linux
      curl -LsSf https://astral.sh/uv/install.sh | sh
      export PATH="$HOME/.local/bin:$PATH"
    fi
  else
    echo -e "✅ ${GREEN}UV is already installed${NC}"
  fi
}

# Install ripgrep based on OS
install_ripgrep() {
  if ! command -v rg >/dev/null 2>&1; then
    echo -e "📦 ${BOLD}Installing ripgrep...${NC}"
    if [ "$OS" = "Darwin" ] && command -v brew >/dev/null 2>&1; then
      brew install ripgrep
    elif [ "$OS" = "Linux" ]; then
      if command -v apt-get >/dev/null 2>&1; then
        sudo apt-get update
        sudo apt-get install -y ripgrep
      elif command -v yum >/dev/null 2>&1; then
        sudo yum install -y ripgrep
      else
        echo -e "${YELLOW}Package manager not found. Installing from binary...${NC}"
        RIPGREP_VERSION="13.0.0"
        wget -q https://github.com/BurntSushi/ripgrep/releases/download/${RIPGREP_VERSION}/ripgrep-${RIPGREP_VERSION}-x86_64-unknown-linux-musl.tar.gz -O /tmp/ripgrep.tar.gz
        tar -xzf /tmp/ripgrep.tar.gz -C /tmp
        sudo cp /tmp/ripgrep-${RIPGREP_VERSION}-x86_64-unknown-linux-musl/rg /usr/local/bin/
        rm -rf /tmp/ripgrep*
      fi
    else
      echo -e "${RED}Unsupported OS: $OS. Please install ripgrep manually.${NC}"
    fi
  else
    echo -e "✅ ${GREEN}ripgrep is already installed${NC}"
  fi
}

# Install yt-dlp using direct download method
install_ytdlp() {
  if ! command -v yt-dlp >/dev/null 2>&1; then
    echo -e "📦 ${BOLD}Installing yt-dlp...${NC}"
    
    # Create ~/.local/bin directory if it doesn't exist
    mkdir -p "$HOME/.local/bin"
    
    # Download yt-dlp to ~/.local/bin
    if curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o "$HOME/.local/bin/yt-dlp"; then
      # Make executable
      chmod a+rx "$HOME/.local/bin/yt-dlp"
      
      # Add ~/.local/bin to PATH if not already there
      if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
        export PATH="$HOME/.local/bin:$PATH"
        echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> "$HOME/.bashrc"
        if [ "$OS" = "Darwin" ]; then
          echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> "$HOME/.zshrc" 2>/dev/null || true
        fi
      fi
      
      echo -e "✅ ${GREEN}yt-dlp installed successfully${NC}"
    else
      echo -e "${RED}Failed to download yt-dlp${NC}"
    fi
  else
    echo -e "✅ ${GREEN}yt-dlp is already installed, updating to latest version...${NC}"
    yt-dlp -U
  fi
}

# Main installation process
echo -e "${BOLD}Installing system dependencies...${NC}"

# Install each dependency
install_ffmpeg
install_go
install_rust
install_bun
install_uv
install_ripgrep
# install_ytdlp

# # Install tools
# echo -e "${BOLD}Installing tools...${NC}"

echo -e "✅ ${GREEN}System dependencies installed!${NC}"

# Final verification for bun to ensure it's available after installation
if ! command -v bun >/dev/null 2>&1; then
  echo -e "${RED}WARNING: Bun is still not in PATH after installation.${NC}"
  echo -e "${YELLOW}You may need to restart your terminal session or manually add bun to your PATH:${NC}"
  echo -e "${BOLD}export PATH=\"$HOME/.bun/bin:\$PATH\"${NC}"
  echo -e "${YELLOW}Then run 'make dev' again.${NC}"
  exit 1
else
  echo -e "✅ ${GREEN}Bun is confirmed working and available in PATH${NC}"
fi