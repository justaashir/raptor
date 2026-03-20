package server

import (
	"embed"
	"fmt"
	"net/http"
	"os"
	"regexp"

	"github.com/labstack/echo/v4"
)

//go:embed skill/SKILL.md
var skillFS embed.FS

var CurrentVersion = "dev"

const githubRepo = "justaashir/raptor"

func (s *Server) handleVersion(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"version": CurrentVersion})
}

func (s *Server) handleSkill(c echo.Context) error {
	data, err := skillFS.ReadFile("skill/SKILL.md")
	if err != nil {
		return c.String(http.StatusInternalServerError, "skill not found")
	}
	return c.Blob(http.StatusOK, "text/plain", data)
}

func (s *Server) handleInstallScript(c echo.Context) error {
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
`, ghURL, serverBaseURL(c.Request()))

	return c.Blob(http.StatusOK, "text/plain", []byte(script))
}

var validHost = regexp.MustCompile(`^[a-zA-Z0-9._:-]+$`)

func serverBaseURL(r *http.Request) string {
	if base := os.Getenv("SERVER_BASE_URL"); base != "" {
		return base
	}
	scheme := "https"
	if r.TLS == nil {
		if fwd := r.Header.Get("X-Forwarded-Proto"); fwd == "http" || fwd == "https" {
			scheme = fwd
		} else {
			scheme = "http"
		}
	}
	host := r.Host
	if !validHost.MatchString(host) {
		return "https://raptor.raptorthree.com"
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}
