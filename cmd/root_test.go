package cmd

import (
	"testing"
	"time"
)

func TestRequireWorkspace_ErrorWhenEmpty(t *testing.T) {
	old := activeWS
	defer func() { activeWS = old }()

	activeWS = ""
	if err := requireWorkspace(); err == nil {
		t.Fatal("expected error when workspace is empty, got nil")
	}
}

func TestRequireWorkspace_OKWhenSet(t *testing.T) {
	old := activeWS
	defer func() { activeWS = old }()

	activeWS = "ws-123"
	if err := requireWorkspace(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequireBoard_ErrorWhenEmpty(t *testing.T) {
	oldWS, oldBoard := activeWS, activeBoard
	defer func() { activeWS = oldWS; activeBoard = oldBoard }()

	activeWS = ""
	activeBoard = ""
	if err := requireBoard(); err == nil {
		t.Fatal("expected error when both are empty, got nil")
	}

	activeWS = "ws-123"
	activeBoard = ""
	if err := requireBoard(); err == nil {
		t.Fatal("expected error when board is empty, got nil")
	}

	activeWS = ""
	activeBoard = "board-1"
	if err := requireBoard(); err == nil {
		t.Fatal("expected error when workspace is empty, got nil")
	}
}

func TestRequireBoard_OKWhenBothSet(t *testing.T) {
	oldWS, oldBoard := activeWS, activeBoard
	defer func() { activeWS = oldWS; activeBoard = oldBoard }()

	activeWS = "ws-123"
	activeBoard = "board-1"
	if err := requireBoard(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShouldCheckUpdate_SkipsWhenRecent(t *testing.T) {
	// When last check was less than 24h ago, should return false
	cfg := Config{
		LastUpdateCheck: time.Now().Add(-1 * time.Hour).Unix(),
	}
	if shouldCheckUpdate(cfg) {
		t.Fatal("expected false when last check was 1 hour ago")
	}
}

func TestShouldCheckUpdate_ChecksWhenStale(t *testing.T) {
	// When last check was more than 24h ago, should return true
	cfg := Config{
		LastUpdateCheck: time.Now().Add(-25 * time.Hour).Unix(),
	}
	if !shouldCheckUpdate(cfg) {
		t.Fatal("expected true when last check was 25 hours ago")
	}
}

func TestShouldCheckUpdate_ChecksWhenNeverChecked(t *testing.T) {
	cfg := Config{}
	if !shouldCheckUpdate(cfg) {
		t.Fatal("expected true when never checked before")
	}
}
