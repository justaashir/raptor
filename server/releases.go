package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Set via the server package or main. Defaults to "dev".
var CurrentVersion = "dev"

// ReleasesDir is the directory where release binaries are stored.
var ReleasesDir = "/data/releases"

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"version": CurrentVersion})
}

func (s *Server) handleRelease(w http.ResponseWriter, r *http.Request) {
	// Path: /releases/{os}/{arch}
	path := strings.TrimPrefix(r.URL.Path, "/releases/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "usage: /releases/{os}/{arch}", http.StatusBadRequest)
		return
	}

	goos, goarch := parts[0], parts[1]

	// Validate os/arch to prevent path traversal
	validOS := map[string]bool{"darwin": true, "linux": true}
	validArch := map[string]bool{"amd64": true, "arm64": true}
	if !validOS[goos] || !validArch[goarch] {
		http.Error(w, "unsupported os/arch", http.StatusNotFound)
		return
	}

	binPath := filepath.Join(ReleasesDir, goos, goarch, "raptor")
	f, err := os.Open(binPath)
	if err != nil {
		http.Error(w, "binary not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	stat, _ := f.Stat()
	modTime := time.Time{}
	if stat != nil {
		modTime = stat.ModTime()
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=raptor-%s-%s", goos, goarch))
	http.ServeContent(w, r, "raptor", modTime, f)
}

func (s *Server) handleInstallScript(w http.ResponseWriter, r *http.Request) {
	scheme := "https"
	if r.TLS == nil {
		if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
			scheme = fwd
		} else {
			scheme = "http"
		}
	}
	baseURL := scheme + "://" + r.Host

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
curl -fsSL "%s/releases/$OS/$ARCH" -o "$INSTALL_DIR/raptor"
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
`, baseURL)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(script))
}

func (s *Server) handleUploadRelease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Path: /admin/releases/{os}/{arch}
	path := strings.TrimPrefix(r.URL.Path, "/admin/releases/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		http.Error(w, "usage: /admin/releases/{os}/{arch}", http.StatusBadRequest)
		return
	}

	goos, goarch := parts[0], parts[1]

	validOS := map[string]bool{"darwin": true, "linux": true}
	validArch := map[string]bool{"amd64": true, "arm64": true}
	if !validOS[goos] || !validArch[goarch] {
		http.Error(w, "unsupported os/arch", http.StatusBadRequest)
		return
	}

	dir := filepath.Join(ReleasesDir, goos, goarch)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	binPath := filepath.Join(dir, "raptor")
	f, err := os.Create(binPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if _, err := f.ReadFrom(r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.Chmod(binPath, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

