package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestServer_ReleaseDownload(t *testing.T) {
	// Set up temp releases dir
	tmp := t.TempDir()
	ReleasesDir = tmp

	dir := filepath.Join(tmp, "darwin", "arm64")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "raptor"), []byte("fake-binary"), 0o755)

	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/releases/darwin/arm64", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if w.Body.String() != "fake-binary" {
		t.Fatalf("expected fake-binary, got %q", w.Body.String())
	}
}

func TestServer_ReleaseDownload_NotFound(t *testing.T) {
	ReleasesDir = t.TempDir()
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/releases/linux/amd64", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestServer_ReleaseDownload_InvalidOS(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/releases/windows/amd64", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestServer_UploadRelease(t *testing.T) {
	tmp := t.TempDir()
	ReleasesDir = tmp
	srv := newTestServer(t)

	body := strings.NewReader("binary-content")
	req := httptest.NewRequest("PUT", "/admin/releases/linux/arm64", body)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	data, err := os.ReadFile(filepath.Join(tmp, "linux", "arm64", "raptor"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "binary-content" {
		t.Fatalf("expected binary-content, got %q", string(data))
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
	if !strings.Contains(body, "https://example.com/releases/") {
		t.Fatal("expected download URL with correct host")
	}
}
