package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServer_VersionEndpoint(t *testing.T) {
	CurrentVersion = "1.2.3"
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/version", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var info struct {
		Version string `json:"version"`
	}
	json.NewDecoder(w.Body).Decode(&info)
	if info.Version != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %s", info.Version)
	}
}

func TestServer_InstallScript(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/install.sh", nil)
	req.Host = "example.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "#!/bin/sh") {
		t.Fatal("expected shell script")
	}
	if !strings.Contains(body, "github.com/justaashir/raptor/releases/latest/download") {
		t.Fatal("expected GitHub releases download URL")
	}
}

func TestServerBaseURL_EnvVar(t *testing.T) {
	t.Setenv("SERVER_BASE_URL", "https://custom.example.com")

	req := httptest.NewRequest("GET", "/install.sh", nil)
	req.Host = "other.example.com"

	got := serverBaseURL(req)
	if got != "https://custom.example.com" {
		t.Fatalf("expected env var value, got %q", got)
	}
}

func TestServerBaseURL_InvalidHost(t *testing.T) {
	t.Setenv("SERVER_BASE_URL", "")

	req := httptest.NewRequest("GET", "/install.sh", nil)
	req.Host = "bad host <script>"

	got := serverBaseURL(req)
	if got != "https://raptor.raptorthree.com" {
		t.Fatalf("expected fallback URL for invalid host, got %q", got)
	}
}
