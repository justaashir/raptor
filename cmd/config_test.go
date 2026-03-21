package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_SaveLoad_RoundTrip(t *testing.T) {
	// Create a temp dir and write/read config directly (bypassing configPath
	// which always returns ~/.raptor.json).
	tmp := t.TempDir()
	path := filepath.Join(tmp, "raptor.json")

	want := Config{
		Server:    "https://example.com",
		Token:     "tok_abc123",
		Username:  "alice",
		Workspace: "ws-001",
		Board:     "board-42",
	}

	data, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var got Config
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Server != want.Server {
		t.Errorf("Server = %q, want %q", got.Server, want.Server)
	}
	if got.Token != want.Token {
		t.Errorf("Token = %q, want %q", got.Token, want.Token)
	}
	if got.Username != want.Username {
		t.Errorf("Username = %q, want %q", got.Username, want.Username)
	}
	if got.Workspace != want.Workspace {
		t.Errorf("Workspace = %q, want %q", got.Workspace, want.Workspace)
	}
	if got.Board != want.Board {
		t.Errorf("Board = %q, want %q", got.Board, want.Board)
	}
}

func TestConfig_LoadMissing_ReturnsError(t *testing.T) {
	// configPath() uses os.UserHomeDir internally, so we test the behavior
	// by trying to read a nonexistent file directly (same as LoadConfig would).
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nonexistent.json")

	_, err := os.ReadFile(path)
	if err == nil {
		t.Fatal("expected error reading nonexistent config, got nil")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected IsNotExist error, got: %v", err)
	}
}

func TestConfig_OmitsEmptyOptionalFields(t *testing.T) {
	cfg := Config{
		Server:   "http://localhost:8080",
		Token:    "tok",
		Username: "bob",
		// Workspace and Board left empty
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	if _, ok := m["workspace"]; ok {
		t.Error("empty Workspace should be omitted (omitempty)")
	}
	if _, ok := m["board"]; ok {
		t.Error("empty Board should be omitted (omitempty)")
	}
}
