package model

import "testing"

func TestBoard_StatusList_Default(t *testing.T) {
	b := Board{Statuses: "todo,in_progress,done"}
	got := b.StatusList()
	if len(got) != 3 || got[0] != "todo" || got[1] != "in_progress" || got[2] != "done" {
		t.Fatalf("expected default statuses, got %v", got)
	}
}

func TestBoard_StatusList_Custom(t *testing.T) {
	b := Board{Statuses: "backlog,dev,review,shipped"}
	got := b.StatusList()
	if len(got) != 4 || got[0] != "backlog" || got[3] != "shipped" {
		t.Fatalf("expected custom statuses, got %v", got)
	}
}

func TestBoard_StatusList_Empty_ReturnsDefaults(t *testing.T) {
	b := Board{Statuses: ""}
	got := b.StatusList()
	if len(got) != 3 {
		t.Fatalf("expected 3 default statuses, got %v", got)
	}
}

func TestBoard_ValidStatus(t *testing.T) {
	b := Board{Statuses: "backlog,active,shipped"}
	if !b.ValidStatus("backlog") {
		t.Fatal("expected backlog to be valid")
	}
	if !b.ValidStatus("shipped") {
		t.Fatal("expected shipped to be valid")
	}
	if b.ValidStatus("todo") {
		t.Fatal("expected todo to be invalid for this board")
	}
}

func TestValidRole_OwnerAndMember(t *testing.T) {
	if !ValidRole("owner") {
		t.Fatal("expected owner to be valid")
	}
	if !ValidRole("member") {
		t.Fatal("expected member to be valid")
	}
}

func TestValidRole_RejectsAdmin(t *testing.T) {
	if ValidRole("admin") {
		t.Fatal("expected admin to be invalid — admin role removed")
	}
}
