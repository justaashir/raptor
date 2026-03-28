package cmd

import (
	"testing"
)

func TestFormatTicketTable_EmptySlice_ReturnsHeaderOnly(t *testing.T) {
	got := formatTicketTable(nil)
	want := "ID\tSTATUS\tASSIGNEE\tTITLE\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
