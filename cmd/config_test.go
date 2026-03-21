package cmd

import (
	"encoding/json"
	"testing"
)

func TestConfig_RoundTrip(t *testing.T) {
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

	var got Config
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got != want {
		t.Errorf("round-trip mismatch:\n got: %+v\nwant: %+v", got, want)
	}
}
