package tui

import (
	"testing"
	"time"
)

func TestFormatAge_JustNow(t *testing.T) {
	got := FormatAge(time.Now())
	if got != "now" {
		t.Fatalf("FormatAge(now) = %q, want %q", got, "now")
	}
}

func TestFormatAge_Minutes(t *testing.T) {
	got := FormatAge(time.Now().Add(-30 * time.Minute))
	if got != "30m" {
		t.Fatalf("FormatAge(30min ago) = %q, want %q", got, "30m")
	}
}

func TestFormatAge_Hours(t *testing.T) {
	got := FormatAge(time.Now().Add(-5 * time.Hour))
	if got != "5h" {
		t.Fatalf("FormatAge(5h ago) = %q, want %q", got, "5h")
	}
}

func TestFormatAge_Days(t *testing.T) {
	got := FormatAge(time.Now().Add(-3 * 24 * time.Hour))
	if got != "3d" {
		t.Fatalf("FormatAge(3d ago) = %q, want %q", got, "3d")
	}
}

func TestFormatAge_Weeks(t *testing.T) {
	got := FormatAge(time.Now().Add(-14 * 24 * time.Hour))
	if got != "2w" {
		t.Fatalf("FormatAge(2w ago) = %q, want %q", got, "2w")
	}
}

func TestFormatAge_Months(t *testing.T) {
	got := FormatAge(time.Now().Add(-60 * 24 * time.Hour))
	if got != "2mo" {
		t.Fatalf("FormatAge(2mo ago) = %q, want %q", got, "2mo")
	}
}

func TestFormatAge_OneMinute(t *testing.T) {
	got := FormatAge(time.Now().Add(-90 * time.Second))
	if got != "1m" {
		t.Fatalf("FormatAge(90s ago) = %q, want %q", got, "1m")
	}
}
