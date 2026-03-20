package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Set via the server package or main. Defaults to "dev".
var CurrentVersion = "dev"

const githubRepo = "justaashir/raptor"

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"version": CurrentVersion})
}

func (s *Server) handleInstallScript(w http.ResponseWriter, r *http.Request) {
	ghURL := fmt.Sprintf("https://github.com/%s/releases/latest/download", githubRepo)

	script := fmt.Sprintf(`#!/bin/sh
set -e

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
esac

INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

echo "Downloading raptor for $OS/$ARCH..."
curl -fsSL "%s/raptor-${OS}-${ARCH}" -o "$INSTALL_DIR/raptor"
chmod +x "$INSTALL_DIR/raptor"

# Install Claude Code skill
SKILL_DIR="$HOME/.claude/skills/raptor"
mkdir -p "$SKILL_DIR"
cat > "$SKILL_DIR/SKILL.md" << 'SKILLEOF'
---
name: raptor
description: Manage the Raptor kanban board via natural language
user-invocable: true
allowed-tools:
  - Bash(raptor *)
---

# Raptor Board Assistant

You manage a multiplayer kanban board using the raptor CLI.

## Available Commands

- raptor list [--status todo|in_progress|done] — List tickets (optionally filter by status)
- raptor add "title" [-c "markdown content"] — Create a new ticket
- raptor show <id> — Show ticket details
- raptor move <id> <status> — Move ticket (todo, in_progress, done)
- raptor edit <id> [-t "title"] [-c "content"] [-s status] — Edit a ticket
- raptor rm <id> — Delete a ticket

## Workflow

1. Always run raptor list first to see current board state
2. Use short ticket IDs (first 8 chars) — raptor accepts partial IDs
3. When adding tickets, include meaningful descriptions with -c flag using markdown
4. Confirm destructive actions (rm, move to done) before executing
5. After making changes, run raptor list to show the updated board

## Output Format

raptor list outputs a formatted table. raptor show outputs ticket details with rendered markdown.
SKILLEOF

echo ""
echo "raptor installed to $INSTALL_DIR/raptor"
echo "Claude Code skill installed to $SKILL_DIR/SKILL.md"
echo ""
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo "Add to your PATH:  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi
`, ghURL)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(script))
}
