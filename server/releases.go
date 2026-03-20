package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
)

//go:embed skill/SKILL.md
var skillFS embed.FS

// Set via the server package or main. Defaults to "dev".
var CurrentVersion = "dev"

const githubRepo = "justaashir/raptor"

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"version": CurrentVersion})
}

func (s *Server) handleSkill(w http.ResponseWriter, r *http.Request) {
	data, err := skillFS.ReadFile("skill/SKILL.md")
	if err != nil {
		http.Error(w, "skill not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(data)
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

# Install Claude Code skill (fetched from server so it stays up to date)
SKILL_DIR="$HOME/.claude/skills/raptor"
mkdir -p "$SKILL_DIR"
curl -fsSL "%s/api/skill" -o "$SKILL_DIR/SKILL.md"

echo ""
echo "raptor installed to $INSTALL_DIR/raptor"
echo "Claude Code skill installed to $SKILL_DIR/SKILL.md"
echo ""
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo "Add to your PATH:  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi
`, ghURL, serverBaseURL(r))

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(script))
}

// serverBaseURL derives the server's public URL from the request.
func serverBaseURL(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil {
		if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
			scheme = fwd
		} else {
			scheme = "http"
		}
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
