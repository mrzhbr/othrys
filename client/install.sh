#!/usr/bin/env bash
# Othrys Client Installer
#
# Installs the othrys CLI and Claude Code plugin commands.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/you/othrys/main/client/install.sh | bash
#
# Or locally:
#   bash client/install.sh
#

set -euo pipefail

REPO_URL="https://github.com/you/othrys"  # TODO: update with real repo
INSTALL_DIR="${OTHRYS_INSTALL_DIR:-$HOME/.othrys}"
BIN_DIR="${OTHRYS_BIN_DIR:-$HOME/.local/bin}"

info()    { echo "[othrys] $*"; }
success() { echo "[othrys] ✓ $*"; }
error()   { echo "[othrys] ✗ $*" >&2; exit 1; }

# ── Detect platform ──────────────────────────────────────────────────
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) error "Unsupported architecture: $ARCH" ;;
esac

info "Platform: ${OS}/${ARCH}"

# ── Check prerequisites ─────────────────────────────────────────────
command -v git >/dev/null 2>&1 || error "git is required"
command -v curl >/dev/null 2>&1 || error "curl is required"

# ── Install CLI binary ──────────────────────────────────────────────
mkdir -p "$INSTALL_DIR" "$BIN_DIR"

# Check if we're in the othrys repo (local install)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

if [ -f "$REPO_ROOT/go.mod" ] && grep -q "othrys" "$REPO_ROOT/go.mod" 2>/dev/null; then
  info "Building from source..."
  if ! command -v go >/dev/null 2>&1; then
    error "Go is required to build from source. Install from https://go.dev/dl/"
  fi
  (cd "$REPO_ROOT" && go build -o "$BIN_DIR/othrys" ./cmd/cli)
  success "CLI built: $BIN_DIR/othrys"
else
  # TODO: Download pre-built binary from GitHub releases
  info "Downloading pre-built binary..."
  error "Pre-built binaries not yet available. Clone the repo and run: bash client/install.sh"
fi

# ── Install Claude Code commands ─────────────────────────────────────
COMMANDS_SRC="$REPO_ROOT/claude-plugin/commands"
if [ -d "$COMMANDS_SRC" ]; then
  mkdir -p "$INSTALL_DIR/commands"
  cp "$COMMANDS_SRC"/othrys-*.md "$INSTALL_DIR/commands/"
  success "Claude Code commands installed to $INSTALL_DIR/commands/"
  info ""
  info "To use in a project, symlink or copy the commands:"
  info "  mkdir -p .claude/commands"
  info "  cp $INSTALL_DIR/commands/othrys-*.md .claude/commands/"
  info ""
  info "Then in Claude Code:"
  info "  /othrys-connect <server-url> <api-key> <project-id>"
else
  info "Claude Code commands not found — skipping."
fi

# ── Add to PATH ──────────────────────────────────────────────────────
if ! echo "$PATH" | grep -q "$BIN_DIR"; then
  info ""
  info "Add this to your shell profile (~/.zshrc or ~/.bashrc):"
  info "  export PATH=\"$BIN_DIR:\$PATH\""
fi

# ── Done ─────────────────────────────────────────────────────────────
echo ""
success "Othrys client installed!"
echo ""
info "Quick start:"
info "  1. Copy commands into your project:"
info "     mkdir -p .claude/commands && cp $INSTALL_DIR/commands/othrys-*.md .claude/commands/"
info "  2. Open Claude Code and run:"
info "     /othrys-connect <server-url> <api-key> <project-id>"
info ""
info "Or use the CLI directly:"
info "  othrys --help"
