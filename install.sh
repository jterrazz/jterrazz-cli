#!/bin/sh
set -e

REPO="jterrazz/jterrazz-cli"
BINARY="j"
INSTALL_DIR="$HOME/.jterrazz/bin"
REPO_DIR="$HOME/Developer/jterrazz-cli"

# Detect platform
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

echo "Installing $BINARY ($OS/$ARCH)..."

# Get latest release tag
TAG="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)"
if [ -z "$TAG" ]; then
  echo "Failed to fetch latest release" && exit 1
fi

# Download binary
URL="https://github.com/$REPO/releases/download/$TAG/$BINARY-$OS-$ARCH"
mkdir -p "$INSTALL_DIR"
curl -fsSL "$URL" -o "$INSTALL_DIR/$BINARY"
chmod +x "$INSTALL_DIR/$BINARY"
echo "✓ Installed $INSTALL_DIR/$BINARY ($TAG)"

# Clone repo for dotfiles/blueprints
if [ ! -d "$REPO_DIR" ]; then
  mkdir -p "$HOME/Developer"
  git clone "https://github.com/$REPO.git" "$REPO_DIR"
  echo "✓ Cloned repo to $REPO_DIR"
fi

# Setup shell
ZSHRC_LINE="source $REPO_DIR/dotfiles/applications/zsh/zshrc.sh"
if [ -f "$HOME/.zshrc" ]; then
  if ! grep -q "zshrc.sh" "$HOME/.zshrc"; then
    printf '\n# jterrazz-cli\n%s\n' "$ZSHRC_LINE" >> "$HOME/.zshrc"
    echo "✓ Added source to ~/.zshrc"
  fi
fi

echo "✓ Done — run 'source ~/.zshrc' then 'j help'"
